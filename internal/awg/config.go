package awg

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type ParsedPeer struct {
	Name         string
	PublicKey    string
	PresharedKey string
	AllowedIPs   []string
	KeepAlive    int
}

type ParsedInterface struct {
	Name              string
	ConfigPath        string
	PrivateKey        string
	ListenPort        int
	Address           string
	MTU               int
	DNS               string
	Jc                int
	Jmin              int
	Jmax              int
	S1                int
	S2                int
	S3                int
	S4                int
	H1                string
	H2                string
	H3                string
	H4                string
	I1                string
	I2                string
	I3                string
	I4                string
	I5                string
	ExternalInterface string
	PostUp            string
	PostDown          string
	Peers             []ParsedPeer
	Running           bool
}

func ConfigDir() string {
	if dir := strings.TrimSpace(os.Getenv("XUI_AWG_CONFIG_DIR")); dir != "" {
		return dir
	}
	return defaultConfigDir
}

func ConfigDirs() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	for _, dir := range []string{
		ConfigDir(),
		"/etc/amnezia/amneziawg",
		"/opt/amnezia/amneziawg",
		"/etc/amnezia/wireguard",
	} {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		out = append(out, dir)
	}
	return out
}

func DiscoverInterfaces() ([]ParsedInterface, error) {
	if !IsInstalled() {
		return nil, nil
	}
	byName := map[string]ParsedInterface{}
	configPaths, err := listConfigPaths()
	if err != nil {
		return nil, err
	}
	for _, path := range configPaths {
		parsed, perr := ParseConfigFile(path)
		if perr != nil {
			continue
		}
		if parsed.Name == "" {
			parsed.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		parsed.ConfigPath = path
		byName[parsed.Name] = parsed
	}
	for _, name := range listRunningInterfaces() {
		entry, ok := byName[name]
		if !ok {
			entry = ParsedInterface{Name: name}
		}
		entry.Running = true
		if strings.TrimSpace(entry.PrivateKey) == "" || len(entry.Peers) == 0 {
			if enriched, err := enrichInterfaceFromRuntime(name, entry); err == nil {
				entry = enriched
			}
		}
		if entry.ConfigPath == "" {
			for _, dir := range ConfigDirs() {
				candidate := filepath.Join(dir, name+".conf")
				if parsed, perr := ParseConfigFile(candidate); perr == nil && strings.TrimSpace(parsed.PrivateKey) != "" {
					parsed.Name = name
					parsed.ConfigPath = candidate
					parsed.Running = true
					entry = mergeParsedInterface(entry, parsed)
					break
				}
			}
		}
		byName[name] = entry
	}
	out := make([]ParsedInterface, 0, len(byName))
	for _, entry := range byName {
		out = append(out, entry)
	}
	return out, nil
}

func listConfigPaths() ([]string, error) {
	out := make([]string, 0, 8)
	seen := map[string]struct{}{}
	for _, dir := range ConfigDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read awg config dir %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".conf") {
				continue
			}
			path := filepath.Join(dir, name)
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			out = append(out, path)
		}
	}
	return out, nil
}

func listRunningInterfaces() []string {
	cmd := exec.Command("awg", "show", "interfaces")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Fields(strings.TrimSpace(string(output)))
	return lines
}

func enrichInterfaceFromRuntime(name string, base ParsedInterface) (ParsedInterface, error) {
	if strings.TrimSpace(name) == "" {
		return base, fmt.Errorf("interface name is required")
	}
	if text, err := exportInterfaceConfig(name); err == nil && strings.TrimSpace(text) != "" {
		parsed, perr := ParseConfigText(text, name)
		if perr == nil {
			parsed.Running = true
			return mergeParsedInterface(base, parsed), nil
		}
	}
	port, _ := readAwgInt(name, "listen-port")
	if port > 0 {
		base.ListenPort = port
	}
	peers, err := dumpPeers(name)
	if err == nil {
		for _, row := range peers {
			base.Peers = append(base.Peers, ParsedPeer{
				PublicKey:  row.PublicKey,
				AllowedIPs: row.AllowedIPs,
				KeepAlive:  row.KeepAlive,
			})
		}
	}
	if strings.TrimSpace(base.PrivateKey) == "" && len(base.Peers) == 0 && base.ListenPort <= 0 {
		return base, fmt.Errorf("unable to read runtime config for %s", name)
	}
	base.Name = name
	base.Running = true
	return base, nil
}

func exportInterfaceConfig(name string) (string, error) {
	cmd := exec.Command("awg", "showconf", name)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func readAwgInt(iface, field string) (int, error) {
	cmd := exec.Command("awg", "show", iface, field)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(output)))
}

func mergeParsedInterface(base, extra ParsedInterface) ParsedInterface {
	out := base
	if strings.TrimSpace(extra.ConfigPath) != "" {
		out.ConfigPath = extra.ConfigPath
	}
	if strings.TrimSpace(extra.PrivateKey) != "" {
		out.PrivateKey = extra.PrivateKey
	}
	if extra.ListenPort > 0 {
		out.ListenPort = extra.ListenPort
	}
	if strings.TrimSpace(extra.Address) != "" {
		out.Address = extra.Address
	}
	if extra.MTU > 0 {
		out.MTU = extra.MTU
	}
	if strings.TrimSpace(extra.DNS) != "" {
		out.DNS = extra.DNS
	}
	if extra.Jc > 0 {
		out.Jc = extra.Jc
	}
	if extra.Jmin > 0 {
		out.Jmin = extra.Jmin
	}
	if extra.Jmax > 0 {
		out.Jmax = extra.Jmax
	}
	if len(extra.Peers) > 0 {
		out.Peers = extra.Peers
	}
	if strings.TrimSpace(extra.PrivateKey) != "" {
		out.S1 = extra.S1
		out.S2 = extra.S2
		out.S3 = extra.S3
		out.S4 = extra.S4
	}
	if strings.TrimSpace(extra.H1) != "" {
		out.H1 = extra.H1
	}
	if strings.TrimSpace(extra.H2) != "" {
		out.H2 = extra.H2
	}
	if strings.TrimSpace(extra.H3) != "" {
		out.H3 = extra.H3
	}
	if strings.TrimSpace(extra.H4) != "" {
		out.H4 = extra.H4
	}
	if strings.TrimSpace(extra.I1) != "" {
		out.I1 = extra.I1
	}
	if strings.TrimSpace(extra.I2) != "" {
		out.I2 = extra.I2
	}
	if strings.TrimSpace(extra.I3) != "" {
		out.I3 = extra.I3
	}
	if strings.TrimSpace(extra.I4) != "" {
		out.I4 = extra.I4
	}
	if strings.TrimSpace(extra.I5) != "" {
		out.I5 = extra.I5
	}
	if strings.TrimSpace(extra.PostUp) != "" {
		out.PostUp = extra.PostUp
	}
	if strings.TrimSpace(extra.PostDown) != "" {
		out.PostDown = extra.PostDown
	}
	if len(extra.Peers) > 0 {
		out.Peers = extra.Peers
	}
	out.Name = extra.Name
	out.Running = extra.Running || base.Running
	return out
}

func ParseConfigText(text, name string) (ParsedInterface, error) {
	tmp, err := os.CreateTemp("", "awg-import-*.conf")
	if err != nil {
		return ParsedInterface{}, err
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.WriteString(text); err != nil {
		_ = tmp.Close()
		return ParsedInterface{}, err
	}
	_ = tmp.Close()
	parsed, err := ParseConfigFile(path)
	if err != nil {
		return ParsedInterface{}, err
	}
	if parsed.Name == "" {
		parsed.Name = name
	}
	return parsed, nil
}

func ParseConfigFile(path string) (ParsedInterface, error) {
	file, err := os.Open(path)
	if err != nil {
		return ParsedInterface{}, fmt.Errorf("open awg config: %w", err)
	}
	defer file.Close()

	var parsed ParsedInterface
	parsed.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	section := ""
	var peer *ParsedPeer
	var peerComment string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			if strings.HasPrefix(line, "#") && section == "peer" && peer != nil {
				comment := strings.TrimSpace(strings.TrimPrefix(line, "#"))
				if comment != "" && peerComment == "" {
					peerComment = comment
				}
			}
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if section == "peer" && peer != nil {
				if peer.PublicKey != "" {
					if peer.Name == "" {
						peer.Name = peerComment
					}
					parsed.Peers = append(parsed.Peers, *peer)
				}
				peer = nil
				peerComment = ""
			}
			section = strings.ToLower(strings.Trim(line, "[]"))
			if section == "peer" {
				peer = &ParsedPeer{}
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch section {
		case "interface":
			assignInterfaceField(&parsed, key, value)
		case "peer":
			if peer == nil {
				peer = &ParsedPeer{}
			}
			assignPeerField(peer, key, value)
		}
	}
	if section == "peer" && peer != nil && peer.PublicKey != "" {
		if peer.Name == "" {
			peer.Name = peerComment
		}
		parsed.Peers = append(parsed.Peers, *peer)
	}
	if err := scanner.Err(); err != nil {
		return ParsedInterface{}, fmt.Errorf("read awg config: %w", err)
	}
	return parsed, nil
}

func assignInterfaceField(parsed *ParsedInterface, key, value string) {
	switch strings.ToLower(key) {
	case "privatekey":
		parsed.PrivateKey = value
	case "listenport":
		parsed.ListenPort, _ = strconv.Atoi(value)
	case "address":
		parsed.Address = value
	case "mtu":
		parsed.MTU, _ = strconv.Atoi(value)
	case "dns":
		parsed.DNS = value
	case "jc":
		parsed.Jc, _ = strconv.Atoi(value)
	case "jmin":
		parsed.Jmin, _ = strconv.Atoi(value)
	case "jmax":
		parsed.Jmax, _ = strconv.Atoi(value)
	case "s1":
		parsed.S1, _ = strconv.Atoi(value)
	case "s2":
		parsed.S2, _ = strconv.Atoi(value)
	case "s3":
		parsed.S3, _ = strconv.Atoi(value)
	case "s4":
		parsed.S4, _ = strconv.Atoi(value)
	case "h1":
		parsed.H1 = value
	case "h2":
		parsed.H2 = value
	case "h3":
		parsed.H3 = value
	case "h4":
		parsed.H4 = value
	case "i1":
		parsed.I1 = value
	case "i2":
		parsed.I2 = value
	case "i3":
		parsed.I3 = value
	case "i4":
		parsed.I4 = value
	case "i5":
		parsed.I5 = value
	case "postup":
		parsed.PostUp = value
	case "postdown":
		parsed.PostDown = value
	}
}

func assignPeerField(peer *ParsedPeer, key, value string) {
	switch strings.ToLower(key) {
	case "publickey":
		peer.PublicKey = value
	case "presharedkey":
		peer.PresharedKey = value
	case "allowedips":
		peer.AllowedIPs = splitAllowedIPs(value)
	case "persistentkeepalive":
		peer.KeepAlive, _ = strconv.Atoi(value)
	}
}
