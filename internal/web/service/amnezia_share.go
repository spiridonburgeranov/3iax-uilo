package service

import (
	"fmt"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/amnezia/vpnuri"
	"github.com/mhsanaei/3x-ui/v3/internal/awg"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	wgconfig "github.com/mhsanaei/3x-ui/v3/internal/wireguard"
)

type AmneziaXrayConfigExporter func(
	remarkTemplate string,
	mux string,
	rules string,
	finalMask string,
	host string,
	inbound *model.Inbound,
	client model.Client,
) ([]byte, string, error)

var registeredAmneziaXrayConfigExporter AmneziaXrayConfigExporter

func RegisterAmneziaXrayConfigExporter(exporter AmneziaXrayConfigExporter) {
	registeredAmneziaXrayConfigExporter = exporter
}

func (s *InboundService) ClientVpnURI(
	setting SettingService,
	host string,
	endpoint string,
	inbound *model.Inbound,
	client *model.Client,
) (string, error) {
	if inbound == nil || client == nil {
		return "", fmt.Errorf("inbound and client are required")
	}
	description := strings.TrimSpace(inbound.Remark)
	if description == "" {
		description = string(inbound.Protocol)
	}
	switch inbound.Protocol {
	case model.AmneziaWG:
		return awg.GenerateVpnURI(inbound, awg.ClientConfigInput{
			PrivateKey:       client.PrivateKey,
			PublicKey:        client.PublicKey,
			AllowedIPs:       client.AllowedIPs,
			PreSharedKey:     client.PreSharedKey,
			KeepAlive:        client.KeepAlive,
			ClientAllowedIPs: "0.0.0.0/0, ::/0",
		}, endpoint)
	case model.WireGuard:
		conf, err := wgconfig.GenerateClientConfig(inbound, client, endpoint)
		if err != nil {
			return "", err
		}
		return vpnuri.FromWireGuardConfig(conf, description)
	case model.VMESS, model.VLESS, model.Trojan, model.Hysteria:
		return s.clientXrayVpnURI(setting, host, inbound, *client, description, "amnezia-xray")
	case model.Shadowsocks:
		return s.clientXrayVpnURI(setting, host, inbound, *client, description, "amnezia-ssxray")
	default:
		return "", fmt.Errorf("protocol %s is not supported by Amnezia VPN import", inbound.Protocol)
	}
}

func (s *InboundService) clientXrayVpnURI(
	setting SettingService,
	host string,
	inbound *model.Inbound,
	client model.Client,
	description string,
	container string,
) (string, error) {
	remarkTemplate, _ := setting.GetRemarkTemplate()
	mux, _ := setting.GetSubJsonMux()
	rules, _ := setting.GetSubJsonRules()
	finalMask, _ := setting.GetSubJsonFinalMask()
	if registeredAmneziaXrayConfigExporter == nil {
		return "", fmt.Errorf("xray amnezia export is not available")
	}
	configJSON, hostName, err := registeredAmneziaXrayConfigExporter(
		remarkTemplate,
		mux,
		rules,
		finalMask,
		host,
		inbound,
		client,
	)
	if err != nil {
		return "", err
	}
	if hostName == "" {
		hostName = host
	}
	return vpnuri.FromXrayConfig(configJSON, description, hostName, container)
}
