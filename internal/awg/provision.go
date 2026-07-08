package awg

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/util/random"
)

const (
	minAmneziaListenPort     = 10000
	maxAmneziaListenPort     = 65535
	randomPortSearchAttempts = 256
	defaultAmneziaServerAddr = "10.66.66.1/24"
)

type ProvisionPlan struct {
	InterfaceName string
	ListenPort    int
	ServerAddress string
	SubnetPool    string
	PrivateKey    string
	PublicKey     string
	MTU           int
	DNS           string
	Jc            int
	Jmin          int
	Jmax          int
	S1            int
	S2            int
	S3            int
	S4            int
	H1            string
	H2            string
	H3            string
	H4            string
	I1            string
	I2            string
	I3            string
	I4            string
	I5            string
}

type ResourceSnapshot struct {
	interfaceNames map[string]struct{}
	ports          map[int]struct{}
	subnetBases    map[string]struct{}
}

func BuildProvisionPlan(snapshot ResourceSnapshot) (ProvisionPlan, error) {
	port, err := pickListenPort(snapshot.ports)
	if err != nil {
		return ProvisionPlan{}, err
	}
	iface := pickInterfaceName(snapshot.interfaceNames)
	serverAddr, pool := pickSubnet(snapshot.subnetBases)
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		return ProvisionPlan{}, err
	}
	obf := GenerateObfuscationParams()
	return ProvisionPlan{
		InterfaceName: iface,
		ListenPort:    port,
		ServerAddress: serverAddr,
		SubnetPool:    pool,
		PrivateKey:    priv,
		PublicKey:     pub,
		MTU:           1420,
		DNS:           "1.1.1.1,2606:4700:4700::1111",
		Jc:            obf.Jc,
		Jmin:          obf.Jmin,
		Jmax:          obf.Jmax,
		S1:            obf.S1,
		S2:            obf.S2,
		S3:            obf.S3,
		S4:            obf.S4,
		H1:            obf.H1,
		H2:            obf.H2,
		H3:            obf.H3,
		H4:            obf.H4,
		I1:            obf.I1,
		I2:            obf.I2,
		I3:            obf.I3,
		I4:            obf.I4,
		I5:            obf.I5,
	}, nil
}

func SnapshotFromNamesPortsSubnets(names []string, ports []int, subnets []string) ResourceSnapshot {
	out := ResourceSnapshot{
		interfaceNames: map[string]struct{}{},
		ports:          map[int]struct{}{},
		subnetBases:    map[string]struct{}{},
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out.interfaceNames[name] = struct{}{}
		}
	}
	for _, port := range ports {
		if port > 0 {
			out.ports[port] = struct{}{}
		}
	}
	for _, subnet := range subnets {
		if base := subnetBase(subnet); base != "" {
			out.subnetBases[base] = struct{}{}
		}
	}
	return out
}

func InterfaceNameForPort(port int, used map[string]struct{}) string {
	_ = port
	return pickInterfaceName(used)
}

func pickInterfaceName(used map[string]struct{}) string {
	for idx := 0; idx < 10_000; idx++ {
		candidate := fmt.Sprintf("awg%d", idx)
		if _, taken := used[candidate]; !taken {
			return candidate
		}
	}
	return "awg_panel"
}

func pickListenPort(blocked map[int]struct{}) (int, error) {
	span := maxAmneziaListenPort - minAmneziaListenPort + 1
	for range randomPortSearchAttempts {
		port := minAmneziaListenPort + random.Num(span)
		if _, exists := blocked[port]; exists {
			continue
		}
		if isUDPPortAvailable(port) {
			return port, nil
		}
	}
	for port := minAmneziaListenPort; port <= maxAmneziaListenPort; port++ {
		if _, exists := blocked[port]; exists {
			continue
		}
		if isUDPPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available UDP port found in range %d-%d", minAmneziaListenPort, maxAmneziaListenPort)
}

func isUDPPortAvailable(port int) bool {
	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func pickSubnet(used map[string]struct{}) (serverAddr string, pool string) {
	if len(used) == 0 {
		return defaultAmneziaServerAddr, "10.66.66.0/24"
	}
	for third := 66; third < 256; third++ {
		base := fmt.Sprintf("10.66.%d.0/24", third)
		if _, taken := used[base]; taken {
			continue
		}
		return fmt.Sprintf("10.66.%d.1/24", third), base
	}
	for second := 67; second < 256; second++ {
		for third := 0; third < 256; third++ {
			base := fmt.Sprintf("10.%d.%d.0/24", second, third)
			if _, taken := used[base]; taken {
				continue
			}
			return fmt.Sprintf("10.%d.%d.1/24", second, third), base
		}
	}
	return defaultAmneziaServerAddr, "10.66.66.0/24"
}

func subnetBase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if prefix, err := netip.ParsePrefix(value); err == nil && prefix.Addr().Is4() {
		network := prefix.Masked()
		return network.String()
	}
	if addr, err := netip.ParseAddr(value); err == nil && addr.Is4() {
		if prefix, perr := addr.Prefix(24); perr == nil {
			return prefix.Masked().String()
		}
	}
	return ""
}

func SubnetBaseFromAddress(address string) string {
	return subnetBase(address)
}

func ParseInterfaceIndex(name string) int {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "awg") {
		return -1
	}
	suffix := strings.TrimPrefix(name, "awg")
	if suffix == "" {
		return 0
	}
	idx, err := strconv.Atoi(suffix)
	if err != nil || idx < 0 {
		return -1
	}
	return idx
}

func ParseInterfacePort(name string) int {
	name = strings.TrimSpace(name)
	const prefix = "awg_in_"
	const suffix = "_ud"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
		return 0
	}
	middle := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	if idx := strings.Index(middle, "_"); idx > 0 {
		middle = middle[:idx]
	}
	port, err := strconv.Atoi(middle)
	if err != nil || port <= 0 {
		return 0
	}
	return port
}
