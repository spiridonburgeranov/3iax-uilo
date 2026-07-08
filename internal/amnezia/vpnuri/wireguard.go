package vpnuri

import (
	"encoding/json"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
)

var wireguardDNSPattern = regexp.MustCompile(
	`DNS\s*=\s*(\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b).*(?:\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b|(?:[0-9a-fA-F:]+:+[0-9a-fA-F:]+))`,
)

func FromWireGuardConfig(conf string, description string) (string, error) {
	conf = strings.TrimSpace(conf)
	if conf == "" {
		return "", common.NewError("wireguard config is required")
	}
	configMap := parseWireGuardLines(conf)
	endpoint := strings.TrimSpace(configMap["Endpoint"])
	if endpoint == "" {
		return "", common.NewError("wireguard endpoint is required")
	}
	hostName, port, err := splitWireGuardEndpoint(endpoint)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(configMap["PrivateKey"]) == "" ||
		strings.TrimSpace(configMap["Address"]) == "" ||
		strings.TrimSpace(configMap["PublicKey"]) == "" {
		return "", common.NewError("wireguard config missing private key, address, or peer public key")
	}

	lastConfig := map[string]any{
		"config": conf,
	}
	lastConfig["hostName"] = hostName
	lastConfig["port"] = port
	lastConfig["client_priv_key"] = strings.TrimSpace(configMap["PrivateKey"])
	lastConfig["client_ip"] = strings.TrimSpace(configMap["Address"])
	lastConfig["server_pub_key"] = strings.TrimSpace(configMap["PublicKey"])
	if psk := firstNonEmpty(configMap["PresharedKey"], configMap["PreSharedKey"]); psk != "" {
		lastConfig["psk_key"] = psk
	}
	if mtu := strings.TrimSpace(configMap["MTU"]); mtu != "" {
		lastConfig["mtu"] = mtu
	} else {
		lastConfig["mtu"] = "1280"
	}
	if keepAlive := strings.TrimSpace(configMap["PersistentKeepalive"]); keepAlive != "" {
		lastConfig["persistent_keep_alive"] = keepAlive
	}
	allowed := strings.TrimSpace(configMap["AllowedIPs"])
	if allowed == "" {
		lastConfig["allowed_ips"] = []string{"0.0.0.0/0", "::/0"}
	} else {
		parts := strings.Split(allowed, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		lastConfig["allowed_ips"] = out
	}

	protocolName, protocolVersion := detectWireGuardProtocol(configMap)
	lastConfigJSON, err := json.Marshal(lastConfig)
	if err != nil {
		return "", err
	}
	protocolConfig := map[string]any{
		"last_config":        string(lastConfigJSON),
		"isThirdPartyConfig": true,
		"port":               strconv.Itoa(port),
		"transport_proto":    "udp",
	}
	if protocolName == "awg" && protocolVersion != "" {
		protocolConfig["protocolVersion"] = protocolVersion
	}
	containerName := "amnezia-" + protocolName
	containers := []map[string]any{{
		"container": containerName,
		protocolName: protocolConfig,
	}}
	outer := map[string]any{
		"containers":       containers,
		"defaultContainer": containerName,
		"description":      description,
		"hostName":         hostName,
	}
	if dns1, dns2, ok := extractWireGuardDNS(conf); ok {
		outer["dns1"] = dns1
		outer["dns2"] = dns2
	}
	return Encode(outer)
}

func parseWireGuardLines(conf string) map[string]string {
	configMap := make(map[string]string)
	for _, line := range strings.Split(conf, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		configMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return configMap
}

func splitWireGuardEndpoint(endpoint string) (string, int, error) {
	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return "", 0, err
		}
		host := parsed.Hostname()
		portText := parsed.Port()
		if host == "" {
			return "", 0, common.NewError("invalid wireguard endpoint")
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port <= 0 {
			port = 51820
		}
		return host, port, nil
	}
	host, portText, err := net.SplitHostPort(endpoint)
	if err != nil {
		if !strings.Contains(endpoint, ":") {
			return endpoint, 51820, nil
		}
		return "", 0, err
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 {
		return "", 0, common.NewError("invalid wireguard endpoint port")
	}
	if strings.Contains(host, ":") {
		host = strings.Trim(host, "[]")
	}
	return host, port, nil
}

func detectWireGuardProtocol(configMap map[string]string) (string, string) {
	required := []string{
		"Jc", "Jmin", "Jmax", "S1", "S2", "S3", "S4",
		"H1", "H2", "H3", "H4",
	}
	for _, key := range required {
		if strings.TrimSpace(configMap[key]) == "" {
			return "wireguard", ""
		}
	}
	optional := []string{
		"I1", "I2", "I3", "I4", "I5",
		"CookieReplyPacketJunkSize", "TransportPacketJunkSize",
	}
	hasOptional := false
	for _, key := range optional {
		if strings.TrimSpace(configMap[key]) != "" {
			hasOptional = true
			break
		}
	}
	if !hasOptional {
		return "wireguard", ""
	}
	if strings.TrimSpace(configMap["CookieReplyPacketJunkSize"]) != "" &&
		strings.TrimSpace(configMap["TransportPacketJunkSize"]) != "" {
		return "awg", "2"
	}
	return "awg", "1.5"
}

func extractWireGuardDNS(conf string) (string, string, bool) {
	match := wireguardDNSPattern.FindStringSubmatch(conf)
	if len(match) < 2 {
		return "", "", false
	}
	dns1 := strings.TrimSpace(match[1])
	dns2 := dns1
	if len(match) > 2 {
		dns2 = strings.TrimSpace(match[2])
	}
	if dns1 == "" {
		return "", "", false
	}
	if dns2 == "" {
		dns2 = dns1
	}
	return dns1, dns2, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
