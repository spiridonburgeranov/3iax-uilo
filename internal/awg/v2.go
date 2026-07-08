package awg

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	wgutil "github.com/mhsanaei/3x-ui/v3/internal/util/wireguard"
)

func GenerateKeyPair() (string, string, error) {
	return wgutil.GenerateWireguardKeypair()
}

func GeneratePresharedKey() (string, error) {
	return wgutil.GenerateWireguardPSK()
}

func WriteServerConfig(interfaceName string, config string) error {
	if strings.TrimSpace(interfaceName) == "" {
		interfaceName = "awg0"
	}
	if err := os.MkdirAll(defaultConfigDir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	path := filepath.Join(defaultConfigDir, interfaceName+".conf")
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func RemoveServerConfig(interfaceName string) {
	if strings.TrimSpace(interfaceName) == "" {
		interfaceName = "awg0"
	}
	_ = os.Remove(filepath.Join(defaultConfigDir, interfaceName+".conf"))
}

func InterfaceUp(interfaceName string) error {
	if strings.TrimSpace(interfaceName) == "" {
		interfaceName = "awg0"
	}
	return up(filepath.Join(defaultConfigDir, interfaceName+".conf"))
}

func InterfaceDown(interfaceName string) error {
	if strings.TrimSpace(interfaceName) == "" {
		interfaceName = "awg0"
	}
	return down(filepath.Join(defaultConfigDir, interfaceName+".conf"))
}

func SyncConfig(interfaceName string) error {
	if strings.TrimSpace(interfaceName) == "" {
		interfaceName = "awg0"
	}
	return sync(interfaceName, filepath.Join(defaultConfigDir, interfaceName+".conf"))
}

func IsInterfaceUp(interfaceName string) bool {
	if strings.TrimSpace(interfaceName) == "" {
		interfaceName = "awg0"
	}
	return isUp(interfaceName)
}

func GenerateServerConfig(server *model.AwgServer, clients []model.AwgClient) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString("PrivateKey = " + server.PrivateKey + "\n")
	addresses := []string{server.IPv4Address}
	if server.IPv6Enabled && strings.TrimSpace(server.IPv6Address) != "" {
		addresses = append(addresses, server.IPv6Address)
	}
	b.WriteString("Address = " + strings.Join(addresses, ", ") + "\n")
	b.WriteString(fmt.Sprintf("ListenPort = %d\n", server.ListenPort))
	if server.MTU > 0 {
		b.WriteString(fmt.Sprintf("MTU = %d\n", server.MTU))
	}
	writeAwgObfuscation(&b, server)
	postUp := strings.TrimSpace(server.PostUp)
	if postUp == "" {
		postUp = GenerateDefaultPostUp(server, clients)
	}
	postDown := strings.TrimSpace(server.PostDown)
	if postDown == "" {
		postDown = GenerateDefaultPostDown(server, clients)
	}
	if postUp != "" {
		b.WriteString("PostUp = " + postUp + "\n")
	}
	if postDown != "" {
		b.WriteString("PostDown = " + postDown + "\n")
	}
	for _, client := range clients {
		if !client.Enable {
			continue
		}
		b.WriteString("\n[Peer]\n")
		if strings.TrimSpace(client.Name) != "" {
			b.WriteString("# " + client.Name + "\n")
		}
		b.WriteString("PublicKey = " + client.PublicKey + "\n")
		if strings.TrimSpace(client.PresharedKey) != "" {
			b.WriteString("PresharedKey = " + client.PresharedKey + "\n")
		}
		b.WriteString("AllowedIPs = " + client.AllowedIPs + "\n")
	}
	return b.String()
}

func GenerateClientConfig(server *model.AwgServer, client *model.AwgClient) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString("PrivateKey = " + client.PrivateKey + "\n")
	addresses := []string{client.IPv4Address}
	if server.IPv6Enabled && strings.TrimSpace(client.IPv6Address) != "" {
		addresses = append(addresses, client.IPv6Address)
	}
	b.WriteString("Address = " + strings.Join(addresses, ", ") + "\n")
	if strings.TrimSpace(server.DNS) != "" {
		b.WriteString("DNS = " + server.DNS + "\n")
	}
	if server.MTU > 0 {
		b.WriteString(fmt.Sprintf("MTU = %d\n", server.MTU))
	}
	writeAwgObfuscation(&b, server)
	b.WriteString("\n[Peer]\n")
	b.WriteString("PublicKey = " + server.PublicKey + "\n")
	if strings.TrimSpace(client.PresharedKey) != "" {
		b.WriteString("PresharedKey = " + client.PresharedKey + "\n")
	}
	endpoint := strings.TrimSpace(server.Endpoint)
	if endpoint != "" {
		if !strings.Contains(endpoint, ":") {
			endpoint = fmt.Sprintf("%s:%d", endpoint, server.ListenPort)
		}
		b.WriteString("Endpoint = " + endpoint + "\n")
	}
	allowedIPs := strings.TrimSpace(client.ClientAllowedIPs)
	if allowedIPs == "" {
		allowedIPs = "0.0.0.0/0, ::/0"
	}
	b.WriteString("AllowedIPs = " + allowedIPs + "\n")
	if client.PersistentKeepalive > 0 {
		b.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", client.PersistentKeepalive))
	}
	return b.String()
}

func writeAwgObfuscation(b *strings.Builder, server *model.AwgServer) {
	b.WriteString(fmt.Sprintf("Jc = %d\n", nonZero(server.Jc, 4)))
	b.WriteString(fmt.Sprintf("Jmin = %d\n", nonZero(server.Jmin, 50)))
	b.WriteString(fmt.Sprintf("Jmax = %d\n", nonZero(server.Jmax, 1000)))
	b.WriteString(fmt.Sprintf("S1 = %d\n", server.S1))
	b.WriteString(fmt.Sprintf("S2 = %d\n", server.S2))
	b.WriteString(fmt.Sprintf("H1 = %d\n", nonZero(server.H1, 1)))
	b.WriteString(fmt.Sprintf("H2 = %d\n", nonZero(server.H2, 2)))
	b.WriteString(fmt.Sprintf("H3 = %d\n", nonZero(server.H3, 3)))
	b.WriteString(fmt.Sprintf("H4 = %d\n", nonZero(server.H4, 4)))
}

func nonZero(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func ipv6Iface(server *model.AwgServer) string {
	if strings.TrimSpace(server.IPv6ExternalInterface) != "" {
		return strings.TrimSpace(server.IPv6ExternalInterface)
	}
	if strings.TrimSpace(server.ExternalInterface) != "" {
		return strings.TrimSpace(server.ExternalInterface)
	}
	return detectDefaultInterface()
}

func GenerateDefaultPostUp(server *model.AwgServer, clients []model.AwgClient) string {
	iface := strings.TrimSpace(server.ExternalInterface)
	if iface == "" {
		iface = detectDefaultInterface()
	}
	name := strings.TrimSpace(server.InterfaceName)
	if name == "" {
		name = "awg0"
	}
	pool := strings.TrimSpace(server.IPv4Pool)
	if pool == "" {
		pool = "10.66.66.0/24"
	}
	parts := []string{
		fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s -o %s -j MASQUERADE", pool, iface),
		fmt.Sprintf("iptables -A FORWARD -i %s -j ACCEPT", name),
		fmt.Sprintf("iptables -A FORWARD -o %s -j ACCEPT", name),
	}
	if server.IPv6Enabled {
		iface6 := ipv6Iface(server)
		parts = append(parts,
			fmt.Sprintf("ip6tables -A FORWARD -i %s -j ACCEPT", name),
			fmt.Sprintf("ip6tables -A FORWARD -o %s -j ACCEPT", name),
			fmt.Sprintf("ip6tables -A FORWARD -i %s -o %s -j ACCEPT", iface6, name),
			"sysctl -w net.ipv6.conf.all.forwarding=1",
			fmt.Sprintf("sysctl -w net.ipv6.conf.%s.proxy_ndp=1", iface6),
		)
		for _, client := range clients {
			if client.Enable && strings.TrimSpace(client.IPv6Address) != "" {
				parts = append(parts, fmt.Sprintf("ip -6 neigh add proxy %s dev %s", stripMask(client.IPv6Address), iface6))
			}
		}
	}
	parts = append(parts, "sysctl -w net.ipv4.ip_forward=1")
	return strings.Join(parts, "; ")
}

func GenerateDefaultPostDown(server *model.AwgServer, clients []model.AwgClient) string {
	iface := strings.TrimSpace(server.ExternalInterface)
	if iface == "" {
		iface = detectDefaultInterface()
	}
	name := strings.TrimSpace(server.InterfaceName)
	if name == "" {
		name = "awg0"
	}
	pool := strings.TrimSpace(server.IPv4Pool)
	if pool == "" {
		pool = "10.66.66.0/24"
	}
	parts := []string{
		fmt.Sprintf("iptables -t nat -D POSTROUTING -s %s -o %s -j MASQUERADE", pool, iface),
		fmt.Sprintf("iptables -D FORWARD -i %s -j ACCEPT", name),
		fmt.Sprintf("iptables -D FORWARD -o %s -j ACCEPT", name),
	}
	if server.IPv6Enabled {
		iface6 := ipv6Iface(server)
		parts = append(parts,
			fmt.Sprintf("ip6tables -D FORWARD -i %s -j ACCEPT", name),
			fmt.Sprintf("ip6tables -D FORWARD -o %s -j ACCEPT", name),
			fmt.Sprintf("ip6tables -D FORWARD -i %s -o %s -j ACCEPT", iface6, name),
		)
		for _, client := range clients {
			if client.Enable && strings.TrimSpace(client.IPv6Address) != "" {
				parts = append(parts, fmt.Sprintf("ip -6 neigh del proxy %s dev %s", stripMask(client.IPv6Address), iface6))
			}
		}
	}
	return strings.Join(parts, "; ")
}

func stripMask(value string) string {
	return strings.TrimSpace(strings.Split(value, "/")[0])
}

func AllocateIPv4(pool string, serverAddr string, usedIPs []string) (string, error) {
	if strings.TrimSpace(pool) == "" {
		pool = "10.66.66.0/24"
	}
	prefix, err := netip.ParsePrefix(pool)
	if err != nil {
		return "", err
	}
	taken := make(map[netip.Addr]struct{}, len(usedIPs)+1)
	if addr := firstAddr(serverAddr); addr.IsValid() {
		taken[addr] = struct{}{}
	}
	for _, used := range usedIPs {
		if addr := firstAddr(used); addr.IsValid() {
			taken[addr] = struct{}{}
		}
	}
	addr := prefix.Masked().Addr().Next()
	for prefix.Contains(addr) {
		if _, ok := taken[addr]; !ok {
			return addr.String() + "/32", nil
		}
		addr = addr.Next()
	}
	return "", common.NewError("amneziawg: IPv4 pool exhausted")
}

func AllocateIPv6(pool string, serverAddr string, usedIPs []string) (string, error) {
	prefix, err := netip.ParsePrefix(pool)
	if err != nil {
		return "", err
	}
	taken := make(map[netip.Addr]struct{}, len(usedIPs)+1)
	if addr := firstAddr(serverAddr); addr.IsValid() {
		taken[addr] = struct{}{}
	}
	for _, used := range usedIPs {
		if addr := firstAddr(used); addr.IsValid() {
			taken[addr] = struct{}{}
		}
	}
	addr := prefix.Masked().Addr().Next()
	for prefix.Contains(addr) {
		if _, ok := taken[addr]; !ok {
			return addr.String() + "/128", nil
		}
		addr = addr.Next()
	}
	return "", common.NewError("amneziawg: IPv6 pool exhausted")
}

func firstAddr(value string) netip.Addr {
	value = strings.TrimSpace(value)
	if value == "" {
		return netip.Addr{}
	}
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Addr()
	}
	if addr, err := netip.ParseAddr(value); err == nil {
		return addr
	}
	return netip.Addr{}
}
