package awg

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	amneziavpn "github.com/mhsanaei/3x-ui/v3/internal/amnezia/vpnuri"
	wgutil "github.com/mhsanaei/3x-ui/v3/internal/util/wireguard"
)

type vpnURIInner struct {
	H1                   string   `json:"H1"`
	H2                   string   `json:"H2"`
	H3                   string   `json:"H3"`
	H4                   string   `json:"H4"`
	Jc                   string   `json:"Jc"`
	Jmin                 string   `json:"Jmin"`
	Jmax                 string   `json:"Jmax"`
	S1                   string   `json:"S1"`
	S2                   string   `json:"S2"`
	S3                   string   `json:"S3"`
	S4                   string   `json:"S4"`
	I1                   string   `json:"I1,omitempty"`
	I2                   string   `json:"I2,omitempty"`
	I3                   string   `json:"I3,omitempty"`
	I4                   string   `json:"I4,omitempty"`
	I5                   string   `json:"I5,omitempty"`
	AllowedIPs           []string `json:"allowed_ips"`
	ClientIP             string   `json:"client_ip"`
	ClientIPv6           string   `json:"client_ipv6"`
	ClientPrivateKey     string   `json:"client_priv_key"`
	PskKey               string   `json:"psk_key,omitempty"`
	Config               string   `json:"config"`
	HostName             string   `json:"hostName"`
	MTU                  string   `json:"mtu"`
	PersistentKeepAlive  string   `json:"persistent_keep_alive"`
	Port                 int      `json:"port"`
	ServerPublicKey      string   `json:"server_pub_key"`
}

type vpnURIContainer struct {
	Awg struct {
		IsThirdPartyConfig bool   `json:"isThirdPartyConfig"`
		LastConfig         string `json:"last_config"`
		Port               string `json:"port"`
		ProtocolVersion    string `json:"protocol_version"`
		TransportProto     string `json:"transport_proto"`
	} `json:"awg"`
	Container string `json:"container"`
}

type vpnURIOuter struct {
	Containers       []vpnURIContainer `json:"containers"`
	DefaultContainer string            `json:"defaultContainer"`
	Description      string            `json:"description"`
	DNS1             string            `json:"dns1"`
	DNS2             string            `json:"dns2"`
	HostName         string            `json:"hostName"`
}

func GenerateVpnURI(inbound *model.Inbound, client ClientConfigInput, endpoint string) (string, error) {
	conf, err := GenerateClientConfig(inbound, client, endpoint)
	if err != nil {
		return "", err
	}
	parsed, err := ParseInboundSettings(inbound.Settings)
	if err != nil {
		return "", err
	}
	serverPub, err := wgutil.PublicKeyFromPrivate(strings.TrimSpace(parsed.SecretKey))
	if err != nil {
		return "", err
	}
	host, port, err := splitEndpointHostPort(endpoint, inbound.Port)
	if err != nil {
		return "", err
	}
	clientIPv4, clientIPv6 := splitClientAddresses(client.AllowedIPs)
	dns1, dns2 := splitDNS(parsed.DNS)
	allowedPeerIPs := splitAllowedPeerIPs(client.ClientAllowedIPs)
	mtu := parsed.MTU
	if mtu <= 0 {
		mtu = 1420
	}
	keepAlive := client.KeepAlive
	if keepAlive <= 0 {
		keepAlive = 25
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
	inner := vpnURIInner{
		H1:                  strings.TrimSpace(parsed.H1),
		H2:                  strings.TrimSpace(parsed.H2),
		H3:                  strings.TrimSpace(parsed.H3),
		H4:                  strings.TrimSpace(parsed.H4),
		Jc:                  strconv.Itoa(jc),
		Jmin:                strconv.Itoa(jmin),
		Jmax:                strconv.Itoa(jmax),
		S1:                  strconv.Itoa(parsed.S1),
		S2:                  strconv.Itoa(parsed.S2),
		S3:                  strconv.Itoa(parsed.S3),
		S4:                  strconv.Itoa(parsed.S4),
		I1:                  strings.TrimSpace(firstNonEmpty(client.I1, parsed.I1)),
		I2:                  strings.TrimSpace(firstNonEmpty(client.I2, parsed.I2)),
		I3:                  strings.TrimSpace(firstNonEmpty(client.I3, parsed.I3)),
		I4:                  strings.TrimSpace(firstNonEmpty(client.I4, parsed.I4)),
		I5:                  strings.TrimSpace(firstNonEmpty(client.I5, parsed.I5)),
		AllowedIPs:          allowedPeerIPs,
		ClientIP:            clientIPv4,
		ClientIPv6:          clientIPv6,
		ClientPrivateKey:    strings.TrimSpace(client.PrivateKey),
		Config:              strings.TrimRight(conf, "\n"),
		HostName:            host,
		MTU:                 strconv.Itoa(mtu),
		PersistentKeepAlive: strconv.Itoa(keepAlive),
		Port:                port,
		ServerPublicKey:     serverPub,
	}
	if psk := strings.TrimSpace(client.PreSharedKey); psk != "" {
		inner.PskKey = psk
	}
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return "", common.NewError("marshal awg vpn uri inner json:", err)
	}
	container := vpnURIContainer{Container: "amnezia-awg"}
	container.Awg.IsThirdPartyConfig = true
	container.Awg.LastConfig = string(innerJSON)
	container.Awg.Port = strconv.Itoa(port)
	container.Awg.ProtocolVersion = "2"
	container.Awg.TransportProto = "udp"
	outer := vpnURIOuter{
		Containers:       []vpnURIContainer{container},
		DefaultContainer: "amnezia-awg",
		Description:      "AWG Server",
		DNS1:             dns1,
		DNS2:             dns2,
		HostName:         host,
	}
	return amneziavpn.Encode(outer)
}

func splitEndpointHostPort(endpoint string, inboundPort int) (string, int, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", 0, common.NewError("endpoint is required for amnezia vpn uri")
	}
	if !strings.Contains(endpoint, ":") {
		if inboundPort <= 0 {
			return "", 0, common.NewError("invalid endpoint port")
		}
		return endpoint, inboundPort, nil
	}
	host, portText, err := net.SplitHostPort(endpoint)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 {
		return "", 0, common.NewError("invalid endpoint port")
	}
	if strings.Contains(host, ":") {
		host = strings.Trim(host, "[]")
	}
	return host, port, nil
}

func splitClientAddresses(allowed []string) (ipv4 string, ipv6 string) {
	for _, item := range allowed {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			item = strings.SplitN(item, "/", 2)[0]
		}
		if strings.Contains(item, ":") {
			if ipv6 == "" {
				ipv6 = item
			}
			continue
		}
		if ipv4 == "" {
			ipv4 = item
		}
	}
	return ipv4, ipv6
}

func splitAllowedPeerIPs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{"0.0.0.0/0", "::/0"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{"0.0.0.0/0", "::/0"}
	}
	return out
}

func splitDNS(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "1.1.1.1", "2606:4700:4700::1111"
	}
	parts := strings.Split(raw, ",")
	dns1 := strings.TrimSpace(parts[0])
	dns2 := dns1
	if len(parts) > 1 {
		dns2 = strings.TrimSpace(parts[1])
	}
	if dns1 == "" {
		dns1 = "1.1.1.1"
	}
	if dns2 == "" {
		dns2 = dns1
	}
	return dns1, dns2
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
