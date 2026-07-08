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
		if entry.ConfigPath == "" {
			entry.ConfigPath = filepath.Join(ConfigDir(), name+".conf")
			if parsed, perr := ParseConfigFile(entry.ConfigPath); perr == nil {
				parsed.Name = name
				parsed.ConfigPath = entry.ConfigPath
				parsed.Running = true
				entry = parsed
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
	dir := ConfigDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read awg config dir: %w", err)
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".conf") {
			out = append(out, filepath.Join(dir, name))
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
