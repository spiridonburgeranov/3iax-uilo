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

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	wgutil "github.com/mhsanaei/3x-ui/v3/internal/util/wireguard"
)

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
	Peers             []peerConfig `json:"peers"`
	Clients           []model.Client
	Jc                int    `json:"jc"`
	Jmin              int    `json:"jmin"`
	Jmax              int    `json:"jmax"`
	S1                int    `json:"s1"`
	S2                int    `json:"s2"`
	H1                int    `json:"h1"`
	H2                int    `json:"h2"`
	H3                int    `json:"h3"`
	H4                int    `json:"h4"`
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

func IsInstalled() bool {
	_, err1 := exec.LookPath("awg")
	_, err2 := exec.LookPath("awg-quick")
	return err1 == nil && err2 == nil
}

func ApplyInbound(inbound *model.Inbound) error {
	if inbound == nil {
		return nil
	}
	if !IsInstalled() {
		return common.NewError("amneziawg runtime is not installed (missing awg/awg-quick)")
	}
	iface := interfaceName(inbound)
	cfg, err := buildConfig(inbound)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(defaultConfigDir, 0o700); err != nil {
		return fmt.Errorf("create awg config directory: %w", err)
	}
	path := filepath.Join(defaultConfigDir, iface+".conf")
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
	iface := interfaceName(inbound)
	path := filepath.Join(defaultConfigDir, iface+".conf")
	_ = down(path)
	return nil
}

func RemoveConfig(inbound *model.Inbound) error {
	if inbound == nil {
		return nil
	}
	path := filepath.Join(defaultConfigDir, interfaceName(inbound)+".conf")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func interfaceName(inbound *model.Inbound) string {
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
	if strings.TrimSpace(parsed.DNS) != "" {
		lines = append(lines, "DNS = "+strings.TrimSpace(parsed.DNS))
	}
	if parsed.Jc > 0 {
		lines = append(lines, fmt.Sprintf("Jc = %d", parsed.Jc))
	}
	if parsed.Jmin > 0 {
		lines = append(lines, fmt.Sprintf("Jmin = %d", parsed.Jmin))
	}
	if parsed.Jmax > 0 {
		lines = append(lines, fmt.Sprintf("Jmax = %d", parsed.Jmax))
	}
	if parsed.S1 > 0 {
		lines = append(lines, fmt.Sprintf("S1 = %d", parsed.S1))
	}
	if parsed.S2 > 0 {
		lines = append(lines, fmt.Sprintf("S2 = %d", parsed.S2))
	}
	if parsed.H1 > 0 {
		lines = append(lines, fmt.Sprintf("H1 = %d", parsed.H1))
	}
	if parsed.H2 > 0 {
		lines = append(lines, fmt.Sprintf("H2 = %d", parsed.H2))
	}
	if parsed.H3 > 0 {
		lines = append(lines, fmt.Sprintf("H3 = %d", parsed.H3))
	}
	if parsed.H4 > 0 {
		lines = append(lines, fmt.Sprintf("H4 = %d", parsed.H4))
	}
	if postUp := buildPostUp(&parsed, serverAddr); postUp != "" {
		lines = append(lines, "PostUp = "+postUp)
	}
	if postDown := buildPostDown(&parsed, serverAddr); postDown != "" {
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

func buildPostUp(parsed *inboundSettings, serverAddr string) string {
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
	return strings.Join([]string{
		"sysctl -w net.ipv4.ip_forward=1",
		fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s -o %s -j MASQUERADE", prefix, iface),
	}, "; ")
}

func buildPostDown(parsed *inboundSettings, serverAddr string) string {
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
	return fmt.Sprintf("iptables -t nat -D POSTROUTING -s %s -o %s -j MASQUERADE", prefix, iface)
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
			return nil, common.NewError("amneziawg client requires publicKey:", c.Email)
		}
		allowed := normalizeAllowedIPs(c.AllowedIPs)
		if len(allowed) == 0 {
			next, err := allocateAddress(used)
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

func allocateAddress(used []string) (string, error) {
	prefix, err := netip.ParsePrefix("10.66.66.0/24")
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

func up(configPath string) error {
	cmd := exec.Command("awg-quick", "up", configPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("awg-quick up failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func down(configPath string) error {
	cmd := exec.Command("awg-quick", "down", configPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("awg-quick down failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func sync(interfaceName, configPath string) error {
	stripCmd := exec.Command("awg-quick", "strip", configPath)
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
