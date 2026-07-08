package service

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

var (
	panelIPv4CacheMu  sync.Mutex
	panelIPv4Cache    string
	panelIPv4CachedAt time.Time
)

const panelIPv4CacheTTL = 5 * time.Minute

func (s *InboundService) ResolveShareEndpoint(inbound *model.Inbound, requestHost, override string) string {
	if inbound == nil || inbound.Port <= 0 {
		return strings.TrimSpace(override)
	}
	if host := advertisableEndpointHost(override); host != "" {
		return joinShareHostPort(host, endpointPort(override, inbound.Port))
	}
	host := s.resolveInboundShareHost(inbound, requestHost)
	if host == "" {
		return ""
	}
	return joinShareHostPort(host, inbound.Port)
}

func (s *InboundService) resolveInboundShareHost(inbound *model.Inbound, requestHost string) string {
	nodeAddr := ""
	if inbound.NodeID != nil {
		var node model.Node
		if err := database.GetDB().First(&node, *inbound.NodeID).Error; err == nil {
			nodeAddr = strings.TrimSpace(node.Address)
		}
	}
	if !isAdvertisableShareHost(nodeAddr) {
		nodeAddr = ""
	}
	listenAddr := shareHostFromListen(inbound.Listen)
	customAddr := ""
	if isAdvertisableShareHost(inbound.ShareAddr) {
		customAddr = strings.TrimSpace(inbound.ShareAddr)
	}
	candidates := []string{nodeAddr, listenAddr}
	switch strings.TrimSpace(inbound.ShareAddrStrategy) {
	case "listen":
		candidates = []string{listenAddr, nodeAddr}
	case "custom":
		candidates = []string{customAddr, nodeAddr, listenAddr}
	}
	for _, candidate := range candidates {
		if candidate != "" {
			return candidate
		}
	}
	if host := configuredPublicHost(); host != "" {
		return host
	}
	if ip := panelPublicIPv4(); ip != "" {
		return ip
	}
	requestHost = extractRequestHostname(requestHost)
	if isAdvertisableShareHost(requestHost) {
		return requestHost
	}
	return ""
}

func advertisableEndpointHost(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	host := endpoint
	if strings.Contains(endpoint, ":") {
		parsedHost, _, err := net.SplitHostPort(endpoint)
		if err != nil {
			return ""
		}
		host = parsedHost
	}
	if !isAdvertisableShareHost(host) {
		return ""
	}
	return strings.Trim(host, "[]")
}

func endpointPort(endpoint string, inboundPort int) int {
	endpoint = strings.TrimSpace(endpoint)
	if strings.Contains(endpoint, ":") {
		if _, portText, err := net.SplitHostPort(endpoint); err == nil {
			var port int
			if _, scanErr := fmt.Sscanf(portText, "%d", &port); scanErr == nil && port > 0 {
				return port
			}
		}
	}
	return inboundPort
}

func joinShareHostPort(host string, port int) string {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if host == "" || port <= 0 {
		return ""
	}
	if strings.Contains(host, ":") {
		return fmt.Sprintf("[%s]:%d", host, port)
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func shareHostFromListen(listen string) string {
	listen = strings.TrimSpace(listen)
	if listen == "" || listen[0] == '@' || listen[0] == '/' {
		return ""
	}
	if !isAdvertisableShareHost(listen) {
		return ""
	}
	return strings.Trim(listen, "[]")
}

func isAdvertisableShareHost(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback() && !ip.IsUnspecified()
	}
	if isWireguardInterfaceName(host) {
		return false
	}
	return !strings.Contains(host, "://") && !strings.ContainsAny(host, "/?#@")
}

func isWireguardInterfaceName(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	switch host {
	case "awg", "wg":
		return true
	}
	if strings.HasPrefix(host, "inbound-awg") {
		return true
	}
	if len(host) >= 4 && strings.HasPrefix(host, "awg") {
		rest := host[3:]
		if rest == "" {
			return true
		}
		for _, r := range rest {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	if len(host) >= 3 && strings.HasPrefix(host, "wg") {
		rest := host[2:]
		if rest == "" {
			return true
		}
		for _, r := range rest {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	return false
}

func configuredPublicHost() string {
	setting := SettingService{}
	if domain, err := setting.GetSubDomain(); err == nil {
		domain = strings.TrimSpace(domain)
		if domain != "" {
			return domain
		}
	}
	if domain, err := setting.GetWebDomain(); err == nil {
		return strings.TrimSpace(domain)
	}
	return ""
}

func panelPublicIPv4() string {
	panelIPv4CacheMu.Lock()
	defer panelIPv4CacheMu.Unlock()
	if panelIPv4Cache != "" && time.Since(panelIPv4CachedAt) < panelIPv4CacheTTL {
		return panelIPv4Cache
	}
	server := &ServerService{}
	server.resolvePublicIPs()
	ip := strings.TrimSpace(server.cachedIPv4)
	if ip != "" && ip != "N/A" {
		panelIPv4Cache = ip
		panelIPv4CachedAt = time.Now()
		return ip
	}
	return panelIPv4Cache
}

func extractRequestHostname(requestHost string) string {
	requestHost = strings.TrimSpace(requestHost)
	if requestHost == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(requestHost); err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(requestHost, "[]")
}
