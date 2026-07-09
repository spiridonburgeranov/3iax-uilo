package awg

import (
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	wgutil "github.com/mhsanaei/3x-ui/v3/internal/util/wireguard"
)

const OnlineHandshakeSeconds = 180

type ClientConfigInput struct {
	PrivateKey        string
	PublicKey         string
	AllowedIPs        []string
	PreSharedKey      string
	KeepAlive         int
	Jc                int
	Jmin              int
	Jmax              int
	I1                string
	I2                string
	I3                string
	I4                string
	I5                string
	ClientAllowedIPs  string
}

const defaultConfigDir = "/etc/amnezia/amneziawg"

type peer struct {
	Email      string
	PublicKey  string
	Preshared  string
	AllowedIPs []string
	KeepAlive  int
}

type inboundSettings struct {
	SecretKey         string       `json:"secretKey"`
	MTU               int          `json:"mtu"`
	DNS               string       `json:"dns"`
	Address           string       `json:"address"`
	AwgInterface      string       `json:"awgInterface"`
	Peers             []peerConfig `json:"peers"`
	Clients           []model.Client
	Jc                int    `json:"jc"`
	Jmin              int    `json:"jmin"`
	Jmax              int    `json:"jmax"`
	S1                int    `json:"s1"`
	S2                int    `json:"s2"`
	S3                int    `json:"s3"`
	S4                int    `json:"s4"`
	H1                string `json:"h1"`
	H2                string `json:"h2"`
	H3                string `json:"h3"`
	H4                string `json:"h4"`
	I1                string `json:"i1"`
	I2                string `json:"i2"`
	I3                string `json:"i3"`
	I4                string `json:"i4"`
	I5                string `json:"i5"`
	ExternalInterface string `json:"externalInterface"`
	PostUp            string `json:"postUp"`
	PostDown          string `json:"postDown"`
}

type peerConfig struct {
	Email      string   `json:"email"`
	PublicKey  string   `json:"publicKey"`
	PrivateKey string   `json:"privateKey"`
	Preshared  string   `json:"preSharedKey"`
	AllowedIPs []string `json:"allowedIPs"`
	KeepAlive  int      `json:"keepAlive"`
}

type PeerRuntime struct {
	InboundID       int      `json:"inboundId"`
	InboundRemark   string   `json:"inboundRemark"`
	InterfaceName   string   `json:"interfaceName"`
	Email           string   `json:"email"`
	PublicKey       string   `json:"publicKey"`
	Endpoint        string   `json:"endpoint"`
	AllowedIPs      []string `json:"allowedIPs"`
	LatestHandshake int64    `json:"latestHandshake"`
	TransferRx      uint64   `json:"transferRx"`
	TransferTx      uint64   `json:"transferTx"`
	KeepAlive       int      `json:"keepAlive"`
	Online          bool     `json:"online"`
}

type peerDumpRow struct {
	InterfaceName   string
	PublicKey       string
	Endpoint        string
	AllowedIPs      []string
	LatestHandshake int64
	TransferRx      uint64
	TransferTx      uint64
	KeepAlive       int
}

func IsInstalled() bool {
	_, err1 := exec.LookPath("awg")
	_, err2 := exec.LookPath("awg-quick")
	return err1 == nil && err2 == nil
}

func Version() string {
	out, err := exec.Command("awg", "--version").CombinedOutput()
	if err != nil {
		out, err = exec.Command("awg", "version").CombinedOutput()
	}
	if err != nil {
		return "unknown"
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) == 0 {
		return "unknown"
	}
	return parts[len(parts)-1]
}

func IsInboundUp(inbound *model.Inbound) bool {
	if inbound == nil {
		return false
	}
	return isUp(InterfaceName(inbound))
}

func RuntimePeers(inbound *model.Inbound) ([]PeerRuntime, error) {
	if inbound == nil {
		return nil, nil
	}
	var parsed inboundSettings
	if err := json.Unmarshal([]byte(inbound.Settings), &parsed); err != nil {
		return nil, common.NewError("invalid amneziawg inbound settings:", err)
	}
	configPeers, err := collectPeers(&parsed)
	if err != nil {
		return nil, err
	}
	emailByKey := make(map[string]string, len(configPeers))
	for _, p := range configPeers {
		emailByKey[p.PublicKey] = p.Email
	}
	iface := InterfaceName(inbound)
	rows, err := dumpPeers(iface)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	out := make([]PeerRuntime, 0, len(rows))
	for _, row := range rows {
		online := row.LatestHandshake > 0 && now-row.LatestHandshake <= OnlineHandshakeSeconds
		out = append(out, PeerRuntime{
			InboundID:       inbound.Id,
			InboundRemark:   inbound.Remark,
			InterfaceName:   iface,
			Email:           emailByKey[row.PublicKey],
			PublicKey:       row.PublicKey,
			Endpoint:        row.Endpoint,
			AllowedIPs:      row.AllowedIPs,
			LatestHandshake: row.LatestHandshake,
			TransferRx:      row.TransferRx,
			TransferTx:      row.TransferTx,
			KeepAlive:       row.KeepAlive,
			Online:          online,
		})
	}
	return out, nil
}

func RuntimeAllPeers() ([]PeerRuntime, error) {
	rows, err := dumpAllPeers()
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	out := make([]PeerRuntime, 0, len(rows))
	for _, row := range rows {
		online := row.LatestHandshake > 0 && now-row.LatestHandshake <= OnlineHandshakeSeconds
		out = append(out, PeerRuntime{
			InterfaceName:   row.InterfaceName,
			PublicKey:       row.PublicKey,
			Endpoint:        row.Endpoint,
			AllowedIPs:      row.AllowedIPs,
			LatestHandshake: row.LatestHandshake,
			TransferRx:      row.TransferRx,
			TransferTx:      row.TransferTx,
			KeepAlive:       row.KeepAlive,
			Online:          online,
		})
	}
	return out, nil
}

func RuntimePeersFromInterface(interfaceName string) ([]PeerRuntime, error) {
	rows, err := dumpPeers(interfaceName)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	out := make([]PeerRuntime, 0, len(rows))
	for _, row := range rows {
		online := row.LatestHandshake > 0 && now-row.LatestHandshake <= OnlineHandshakeSeconds
		out = append(out, PeerRuntime{
			InterfaceName:   row.InterfaceName,
			PublicKey:       row.PublicKey,
			Endpoint:        row.Endpoint,
			AllowedIPs:      row.AllowedIPs,
			LatestHandshake: row.LatestHandshake,
			TransferRx:      row.TransferRx,
			TransferTx:      row.TransferTx,
			KeepAlive:       row.KeepAlive,
			Online:          online,
		})
	}
	return out, nil
}

func ApplyInbound(inbound *model.Inbound) error {
	if inbound == nil {
		return nil
	}
	if !IsInstalled() {
		return common.NewError("amneziawg runtime is not installed (missing awg/awg-quick)")
	}
	iface := InterfaceName(inbound)
	cfg, err := buildConfig(inbound)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ConfigDir(), 0o700); err != nil {
		return fmt.Errorf("create awg config directory: %w", err)
	}
	path := filepath.Join(ConfigDir(), iface+".conf")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		return fmt.Errorf("write awg config: %w", err)
	}
	if isUp(iface) {
		if err := sync(iface, path); err != nil {
			return err
		}
		return nil
	}
	return up(path)
}

func DisableInbound(inbound *model.Inbound) error {
	if inbound == nil {
		return nil
	}
	if !IsInstalled() {
		return nil
	}
	iface := InterfaceName(inbound)
	path := filepath.Join(ConfigDir(), iface+".conf")
	_ = down(path)
	return nil
}

func RemoveConfig(inbound *model.Inbound) error {
	if inbound == nil {
		return nil
	}
	path := filepath.Join(ConfigDir(), InterfaceName(inbound)+".conf")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func InterfaceName(inbound *model.Inbound) string {
	var parsed inboundSettings
	if inbound != nil && strings.TrimSpace(inbound.Settings) != "" {
		_ = json.Unmarshal([]byte(inbound.Settings), &parsed)
		if name := strings.TrimSpace(parsed.AwgInterface); name != "" {
			return name
		}
	}
	if inbound.Tag != "" {
		re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
		tag := strings.ToLower(re.ReplaceAllString(inbound.Tag, "_"))
		if tag != "" {
			if len(tag) > 11 {
				tag = tag[:11]
			}
			return "awg_" + tag
		}
	}
	if inbound.Id > 0 {
		return "awg_" + strconv.Itoa(inbound.Id)
	}
	return "awg_panel"
}

func buildConfig(inbound *model.Inbound) (string, error) {
	var parsed inboundSettings
	if err := json.Unmarshal([]byte(inbound.Settings), &parsed); err != nil {
		return "", common.NewError("invalid amneziawg inbound settings:", err)
	}
	if strings.TrimSpace(parsed.SecretKey) == "" {
		return "", common.NewError("amneziawg secretKey is required")
	}
	serverAddr := strings.TrimSpace(parsed.Address)
	if serverAddr == "" {
		serverAddr = "10.66.66.1/24"
	}
	lines := []string{"[Interface]"}
	lines = append(lines, "PrivateKey = "+parsed.SecretKey)
	lines = append(lines, "Address = "+serverAddr)
	lines = append(lines, fmt.Sprintf("ListenPort = %d", inbound.Port))
	if parsed.MTU > 0 {
		lines = append(lines, fmt.Sprintf("MTU = %d", parsed.MTU))
	}
	appendObfuscationLines(&lines, &parsed)
	if postUp := buildPostUp(inbound, &parsed, serverAddr); postUp != "" {
		lines = append(lines, "PostUp = "+postUp)
	}
	if postDown := buildPostDown(inbound, &parsed, serverAddr); postDown != "" {
		lines = append(lines, "PostDown = "+postDown)
	}
	peers, err := collectPeers(&parsed)
	if err != nil {
		return "", err
	}
	for _, p := range peers {
		lines = append(lines, "", "[Peer]", "# "+p.Email, "PublicKey = "+p.PublicKey)
		if p.Preshared != "" {
			lines = append(lines, "PresharedKey = "+p.Preshared)
		}
		if len(p.AllowedIPs) == 0 {
			lines = append(lines, "AllowedIPs = 0.0.0.0/0, ::/0")
		} else {
			lines = append(lines, "AllowedIPs = "+strings.Join(p.AllowedIPs, ", "))
		}
		if p.KeepAlive > 0 {
			lines = append(lines, fmt.Sprintf("PersistentKeepalive = %d", p.KeepAlive))
		}
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func appendObfuscationLines(lines *[]string, parsed *inboundSettings) {
	*lines = append(*lines, fmt.Sprintf("Jc = %d", parsed.Jc))
	*lines = append(*lines, fmt.Sprintf("Jmin = %d", parsed.Jmin))
	*lines = append(*lines, fmt.Sprintf("Jmax = %d", parsed.Jmax))
	*lines = append(*lines, fmt.Sprintf("S1 = %d", parsed.S1))
	*lines = append(*lines, fmt.Sprintf("S2 = %d", parsed.S2))
	*lines = append(*lines, fmt.Sprintf("S3 = %d", parsed.S3))
	*lines = append(*lines, fmt.Sprintf("S4 = %d", parsed.S4))
	if strings.TrimSpace(parsed.H1) != "" {
		*lines = append(*lines, "H1 = "+strings.TrimSpace(parsed.H1))
	}
	if strings.TrimSpace(parsed.H2) != "" {
		*lines = append(*lines, "H2 = "+strings.TrimSpace(parsed.H2))
	}
	if strings.TrimSpace(parsed.H3) != "" {
		*lines = append(*lines, "H3 = "+strings.TrimSpace(parsed.H3))
	}
	if strings.TrimSpace(parsed.H4) != "" {
		*lines = append(*lines, "H4 = "+strings.TrimSpace(parsed.H4))
	}
	for idx, value := range []string{parsed.I1, parsed.I2, parsed.I3, parsed.I4, parsed.I5} {
		value = strings.TrimSpace(value)
		if value != "" {
			*lines = append(*lines, fmt.Sprintf("I%d = %s", idx+1, value))
		}
	}
}

func ParseInboundSettings(settings string) (inboundSettings, error) {
	var parsed inboundSettings
	if strings.TrimSpace(settings) == "" {
		return parsed, common.NewError("empty amneziawg settings")
	}
	if err := json.Unmarshal([]byte(settings), &parsed); err != nil {
		return parsed, common.NewError("invalid amneziawg inbound settings:", err)
	}
	return parsed, nil
}

func GenerateClientConfig(inbound *model.Inbound, client ClientConfigInput, endpoint string) (string, error) {
	if inbound == nil {
		return "", common.NewError("inbound is required")
	}
	parsed, err := ParseInboundSettings(inbound.Settings)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.SecretKey) == "" {
		return "", common.NewError("amneziawg secretKey is required")
	}
	serverPub, err := wgutil.PublicKeyFromPrivate(strings.TrimSpace(parsed.SecretKey))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(client.PrivateKey) == "" {
		return "", common.NewError("client private key is required")
	}
	address := strings.Join(client.AllowedIPs, ", ")
	if address == "" {
		address = "10.66.66.2/32"
	}
	lines := []string{"[Interface]"}
	lines = append(lines, "PrivateKey = "+strings.TrimSpace(client.PrivateKey))
	lines = append(lines, "Address = "+address)
	dns := strings.TrimSpace(parsed.DNS)
	if dns == "" {
		dns = "1.1.1.1,2606:4700:4700::1111"
	}
	lines = append(lines, "DNS = "+dns)
	if parsed.MTU > 0 {
		lines = append(lines, fmt.Sprintf("MTU = %d", parsed.MTU))
	}
	jc := client.Jc
	if jc <= 0 {
		jc = parsed.Jc
	}
	jmin := client.Jmin
	if jmin <= 0 {
		jmin = parsed.Jmin
	}
	jmax := client.Jmax
	if jmax <= 0 {
		jmax = parsed.Jmax
	}
	lines = append(lines, fmt.Sprintf("Jc = %d", jc))
	lines = append(lines, fmt.Sprintf("Jmin = %d", jmin))
	lines = append(lines, fmt.Sprintf("Jmax = %d", jmax))
	lines = append(lines, fmt.Sprintf("S1 = %d", parsed.S1))
	lines = append(lines, fmt.Sprintf("S2 = %d", parsed.S2))
	lines = append(lines, fmt.Sprintf("S3 = %d", parsed.S3))
	lines = append(lines, fmt.Sprintf("S4 = %d", parsed.S4))
	if strings.TrimSpace(parsed.H1) != "" {
		lines = append(lines, "H1 = "+strings.TrimSpace(parsed.H1))
	}
	if strings.TrimSpace(parsed.H2) != "" {
		lines = append(lines, "H2 = "+strings.TrimSpace(parsed.H2))
	}
	if strings.TrimSpace(parsed.H3) != "" {
		lines = append(lines, "H3 = "+strings.TrimSpace(parsed.H3))
	}
	if strings.TrimSpace(parsed.H4) != "" {
		lines = append(lines, "H4 = "+strings.TrimSpace(parsed.H4))
	}
	for idx, fallback := range []string{parsed.I1, parsed.I2, parsed.I3, parsed.I4, parsed.I5} {
		value := fallback
		switch idx {
		case 0:
			if strings.TrimSpace(client.I1) != "" {
				value = client.I1
			}
		case 1:
			if strings.TrimSpace(client.I2) != "" {
				value = client.I2
			}
		case 2:
			if strings.TrimSpace(client.I3) != "" {
				value = client.I3
			}
		case 3:
			if strings.TrimSpace(client.I4) != "" {
				value = client.I4
			}
		case 4:
			if strings.TrimSpace(client.I5) != "" {
				value = client.I5
			}
		}
		value = strings.TrimSpace(value)
		if value != "" {
			lines = append(lines, fmt.Sprintf("I%d = %s", idx+1, value))
		}
	}
	lines = append(lines, "", "[Peer]", "PublicKey = "+serverPub)
	if strings.TrimSpace(client.PreSharedKey) != "" {
		lines = append(lines, "PresharedKey = "+strings.TrimSpace(client.PreSharedKey))
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint != "" {
		if !strings.Contains(endpoint, ":") && inbound.Port > 0 {
			endpoint = fmt.Sprintf("%s:%d", endpoint, inbound.Port)
		}
		lines = append(lines, "Endpoint = "+endpoint)
	}
	allowed := strings.TrimSpace(client.ClientAllowedIPs)
	if allowed == "" {
		allowed = "0.0.0.0/0, ::/0"
	}
	lines = append(lines, "AllowedIPs = "+allowed)
	keepAlive := client.KeepAlive
	if keepAlive <= 0 {
		keepAlive = 25
	}
	if keepAlive > 0 {
		lines = append(lines, fmt.Sprintf("PersistentKeepalive = %d", keepAlive))
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func buildPostUp(inbound *model.Inbound, parsed *inboundSettings, serverAddr string) string {
	if strings.TrimSpace(parsed.PostUp) != "" {
		return strings.TrimSpace(parsed.PostUp)
	}
	iface := strings.TrimSpace(parsed.ExternalInterface)
	if iface == "" {
		iface = detectDefaultInterface()
	}
	prefix := serverIPv4Prefix(serverAddr)
	if prefix == "" {
		return "sysctl -w net.ipv4.ip_forward=1"
	}
	name := InterfaceName(inbound)
	return strings.Join([]string{
		fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s -o %s -j MASQUERADE", prefix, iface),
		fmt.Sprintf("iptables -A FORWARD -i %s -j ACCEPT", name),
		fmt.Sprintf("iptables -A FORWARD -o %s -j ACCEPT", name),
		"sysctl -w net.ipv4.ip_forward=1",
	}, "; ")
}

func buildPostDown(inbound *model.Inbound, parsed *inboundSettings, serverAddr string) string {
	if strings.TrimSpace(parsed.PostDown) != "" {
		return strings.TrimSpace(parsed.PostDown)
	}
	iface := strings.TrimSpace(parsed.ExternalInterface)
	if iface == "" {
		iface = detectDefaultInterface()
	}
	prefix := serverIPv4Prefix(serverAddr)
	if prefix == "" {
		return ""
	}
	name := InterfaceName(inbound)
	return strings.Join([]string{
		fmt.Sprintf("iptables -t nat -D POSTROUTING -s %s -o %s -j MASQUERADE", prefix, iface),
		fmt.Sprintf("iptables -D FORWARD -i %s -j ACCEPT", name),
		fmt.Sprintf("iptables -D FORWARD -o %s -j ACCEPT", name),
	}, "; ")
}

func serverIPv4Prefix(serverAddr string) string {
	for _, part := range strings.Split(serverAddr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		p, err := netip.ParsePrefix(part)
		if err != nil || !p.Addr().Is4() {
			continue
		}
		return p.Masked().String()
	}
	return ""
}

func detectDefaultInterface() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "eth0"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		name := iface.Name
		if strings.HasPrefix(name, "awg") || strings.HasPrefix(name, "wg") ||
			strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "veth") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && ipNet.IP.To4() != nil && !ipNet.IP.IsLinkLocalUnicast() {
				return name
			}
		}
	}
	return "eth0"
}

func collectPeers(parsed *inboundSettings) ([]peer, error) {
	out := make([]peer, 0, len(parsed.Clients)+len(parsed.Peers))
	used := make([]string, 0, len(parsed.Clients))
	for i := range parsed.Clients {
		c := parsed.Clients[i]
		if !c.Enable || strings.TrimSpace(c.Email) == "" {
			continue
		}
		publicKey := strings.TrimSpace(c.PublicKey)
		if publicKey == "" && strings.TrimSpace(c.PrivateKey) != "" {
			derived, err := wgutil.PublicKeyFromPrivate(strings.TrimSpace(c.PrivateKey))
			if err == nil {
				publicKey = derived
			}
		}
		if publicKey == "" {
			logger.Warning("skip amneziawg client without publicKey:", c.Email)
			continue
		}
		allowed := normalizeAllowedIPs(c.AllowedIPs)
		if len(allowed) == 0 {
			next, err := allocateAddress(used, serverIPv4Prefix(parsed.Address))
			if err != nil {
				return nil, err
			}
			allowed = []string{next}
		}
		used = append(used, allowed...)
		out = append(out, peer{
			Email:      c.Email,
			PublicKey:  publicKey,
			Preshared:  c.PreSharedKey,
			AllowedIPs: allowed,
			KeepAlive:  c.KeepAlive,
		})
	}
	for i := range parsed.Peers {
		p := parsed.Peers[i]
		if strings.TrimSpace(p.PublicKey) == "" {
			continue
		}
		out = append(out, peer{
			Email:      p.Email,
			PublicKey:  strings.TrimSpace(p.PublicKey),
			Preshared:  strings.TrimSpace(p.Preshared),
			AllowedIPs: normalizeAllowedIPs(p.AllowedIPs),
			KeepAlive:  p.KeepAlive,
		})
	}
	return out, nil
}

func normalizeAllowedIPs(values []string) []string {
	out := make([]string, 0, len(values))
	for _, raw := range values {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if p, err := netip.ParsePrefix(v); err == nil {
			out = append(out, p.String())
			continue
		}
		if a, err := netip.ParseAddr(v); err == nil {
			out = append(out, netip.PrefixFrom(a, a.BitLen()).String())
		}
	}
	return out
}

func allocateAddress(used []string, base string) (string, error) {
	if strings.TrimSpace(base) == "" {
		base = "10.66.66.0/24"
	}
	prefix, err := netip.ParsePrefix(base)
	if err != nil {
		return "", err
	}
	taken := map[netip.Addr]struct{}{}
	for _, entry := range used {
		if p, e := netip.ParsePrefix(entry); e == nil {
			taken[p.Addr()] = struct{}{}
		}
	}
	addr := prefix.Addr().Next().Next()
	for prefix.Contains(addr) {
		if _, ok := taken[addr]; !ok {
			return addr.String() + "/32", nil
		}
		addr = addr.Next()
	}
	return "", common.NewError("amneziawg address pool exhausted")
}

func isUp(interfaceName string) bool {
	cmd := exec.Command("awg", "show", interfaceName)
	return cmd.Run() == nil
}

func dumpPeers(interfaceName string) ([]peerDumpRow, error) {
	cmd := exec.Command("awg", "show", interfaceName, "dump")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("awg show dump failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) <= 1 {
		return nil, nil
	}
	out := make([]peerDumpRow, 0, len(lines)-1)
	for _, line := range lines[1:] {
		fields := strings.Split(line, "\t")
		if len(fields) < 8 {
			continue
		}
		handshake, _ := strconv.ParseInt(fields[4], 10, 64)
		rx, _ := strconv.ParseUint(fields[5], 10, 64)
		tx, _ := strconv.ParseUint(fields[6], 10, 64)
		keepAlive, _ := strconv.Atoi(fields[7])
		out = append(out, peerDumpRow{
			InterfaceName:   interfaceName,
			PublicKey:       fields[0],
			Endpoint:        fields[2],
			AllowedIPs:      splitAllowedIPs(fields[3]),
			LatestHandshake: handshake,
			TransferRx:      rx,
			TransferTx:      tx,
			KeepAlive:       keepAlive,
		})
	}
	return out, nil
}

func dumpAllPeers() ([]peerDumpRow, error) {
	cmd := exec.Command("awg", "show", "all", "dump")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("awg show all dump failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) <= 1 {
		return nil, nil
	}
	out := make([]peerDumpRow, 0, len(lines)-1)
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) < 9 {
			continue
		}
		handshake, _ := strconv.ParseInt(fields[5], 10, 64)
		rx, _ := strconv.ParseUint(fields[6], 10, 64)
		tx, _ := strconv.ParseUint(fields[7], 10, 64)
		keepAlive, _ := strconv.Atoi(fields[8])
		out = append(out, peerDumpRow{
			InterfaceName:   fields[0],
			PublicKey:       fields[1],
			Endpoint:        fields[3],
			AllowedIPs:      splitAllowedIPs(fields[4]),
			LatestHandshake: handshake,
			TransferRx:      rx,
			TransferTx:      tx,
			KeepAlive:       keepAlive,
		})
	}
	return out, nil
}

func splitAllowedIPs(value string) []string {
	if value == "" || value == "(none)" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func up(configPath string) error {
	cmd := exec.Command("awg-quick", "up", configPath)
	withAwgGoEnv(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("awg-quick up failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func down(configPath string) error {
	cmd := exec.Command("awg-quick", "down", configPath)
	withAwgGoEnv(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("awg-quick down failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func sync(interfaceName, configPath string) error {
	stripCmd := exec.Command("awg-quick", "strip", configPath)
	withAwgGoEnv(stripCmd)
	stripped, err := stripCmd.Output()
	if err != nil {
		return up(configPath)
	}
	cmd := exec.Command("awg", "syncconf", interfaceName, "/dev/stdin")
	cmd.Stdin = strings.NewReader(string(stripped))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("awg syncconf failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func withAwgGoEnv(cmd *exec.Cmd) {
	if _, err := exec.LookPath("amneziawg-go"); err == nil {
		cmd.Env = append(os.Environ(), "WG_QUICK_USERSPACE_IMPLEMENTATION=amneziawg-go")
	}
}
