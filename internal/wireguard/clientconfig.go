package wireguard

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	wgutil "github.com/mhsanaei/3x-ui/v3/internal/util/wireguard"
)

type inboundSettings struct {
	SecretKey string `json:"secretKey"`
	DNS       string `json:"dns"`
	MTU       int    `json:"mtu"`
}

func GenerateClientConfig(inbound *model.Inbound, client *model.Client, endpoint string) (string, error) {
	if inbound == nil || client == nil {
		return "", common.NewError("inbound and client are required")
	}
	if inbound.Protocol != model.WireGuard {
		return "", common.NewError("inbound is not wireguard")
	}
	var parsed inboundSettings
	if err := json.Unmarshal([]byte(inbound.Settings), &parsed); err != nil {
		return "", common.NewError("invalid wireguard inbound settings:", err)
	}
	if strings.TrimSpace(parsed.SecretKey) == "" {
		return "", common.NewError("wireguard secretKey is required")
	}
	if strings.TrimSpace(client.PrivateKey) == "" {
		return "", common.NewError("client private key is required")
	}
	serverPub, err := wgutil.PublicKeyFromPrivate(strings.TrimSpace(parsed.SecretKey))
	if err != nil {
		return "", err
	}
	address := strings.Join(client.AllowedIPs, ", ")
	if address == "" {
		address = "10.0.0.2/32"
	}
	dns := strings.TrimSpace(parsed.DNS)
	if dns == "" {
		dns = "1.1.1.1, 1.0.0.1"
	}
	lines := []string{
		"[Interface]",
		"PrivateKey = " + strings.TrimSpace(client.PrivateKey),
		"Address = " + address,
		"DNS = " + dns,
	}
	if parsed.MTU > 0 {
		lines = append(lines, fmt.Sprintf("MTU = %d", parsed.MTU))
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
	lines = append(lines, "AllowedIPs = 0.0.0.0/0, ::/0")
	if client.KeepAlive > 0 {
		lines = append(lines, fmt.Sprintf("PersistentKeepalive = %d", client.KeepAlive))
	}
	return strings.Join(lines, "\n"), nil
}
