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
	iface := pickInterfaceName(port, snapshot.interfaceNames)
	serverAddr, pool := pickSubnet(snapshot.subnetBases)
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		return ProvisionPlan{}, err
	}
	obf := randomObfuscation()
	return ProvisionPlan{
		InterfaceName: iface,
		ListenPort:    port,
		ServerAddress: serverAddr,
		SubnetPool:    pool,
		PrivateKey:    priv,
		PublicKey:     pub,
		MTU:           1420,
		DNS:           "1.1.1.1,2606:4700:4700::1111",
		Jc:            obf.jc,
		Jmin:          obf.jmin,
		Jmax:          obf.jmax,
		S1:            obf.s1,
		S2:            obf.s2,
		S3:            obf.s3,
		S4:            obf.s4,
		H1:            obf.h1,
		H2:            obf.h2,
		H3:            obf.h3,
		H4:            obf.h4,
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
	return pickInterfaceName(port, used)
}

func pickInterfaceName(port int, used map[string]struct{}) string {
	if _, taken := used["awg0"]; !taken {
		return "awg0"
	}
	name := fmt.Sprintf("awg_in_%d_ud", port)
	if _, taken := used[name]; !taken {
		return name
	}
	for idx := 2; idx < 1000; idx++ {
		candidate := fmt.Sprintf("%s_%d", name, idx)
		if _, taken := used[candidate]; !taken {
			return candidate
		}
	}
	return name
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

type obfuscationSeed struct {
	jc, jmin, jmax       int
	s1, s2, s3, s4       int
	h1, h2, h3, h4       string
}

func randomObfuscation() obfuscationSeed {
	jc := 3 + random.Num(6)
	jmin := 50 + random.Num(80)
	jmax := jmin + 100 + random.Num(200)
	if jmax > 1024 {
		jmax = 1024
	}
	return obfuscationSeed{
		jc:   jc,
		jmin: jmin,
		jmax: jmax,
		s1:   10 + random.Num(20),
		s2:   20 + random.Num(20),
		s3:   30 + random.Num(20),
		s4:   10 + random.Num(20),
		h1:   randomDigits(1 + random.Num(3)),
		h2:   randomDigits(1 + random.Num(3)),
		h3:   randomDigits(1 + random.Num(3)),
		h4:   randomDigits(1 + random.Num(3)),
	}
}

func randomDigits(length int) string {
	if length <= 0 {
		length = 1
	}
	out := make([]byte, length)
	for i := range out {
		out[i] = byte('0' + random.Num(10))
	}
	return string(out)
}

func SubnetBaseFromAddress(address string) string {
	return subnetBase(address)
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
