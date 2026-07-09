package service

import (
	"encoding/json"
	"net/netip"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	wgutil "github.com/mhsanaei/3x-ui/v3/internal/util/wireguard"
)

const defaultWireguardBase = "10.0.0.0/24"
const defaultAmneziaWGAddress = "10.66.66.1/24"
const defaultAmneziaWGBase = "10.66.66.0/24"

func keepAliveStr(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	return strconv.Itoa(seconds)
}

func wireguardHostAddr(s string) netip.Addr {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Addr{}
	}
	if p, err := netip.ParsePrefix(s); err == nil {
		return p.Addr()
	}
	if a, err := netip.ParseAddr(s); err == nil {
		return a
	}
	return netip.Addr{}
}

func wireguardAllocationBase(used []string, fallback string) string {
	for _, u := range used {
		a := wireguardHostAddr(u)
		if !a.IsValid() || !a.Is4() || a.IsUnspecified() {
			continue
		}
		if p, err := a.Prefix(24); err == nil {
			return p.String()
		}
	}
	return fallback
}

func wireguardSettingsAddresses(settings string, fallback string) []string {
	var raw map[string]any
	if err := json.Unmarshal([]byte(settings), &raw); err != nil {
		if fallback == "" {
			return nil
		}
		return []string{fallback}
	}
	value, ok := raw["address"]
	if !ok {
		if fallback == "" {
			return nil
		}
		return []string{fallback}
	}
	switch v := value.(type) {
	case string:
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) > 0 {
			return out
		}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					out = append(out, trimmed)
				}
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if fallback == "" {
		return nil
	}
	return []string{fallback}
}

func wireguardBaseFromAddresses(addresses []string, fallback string) string {
	for _, address := range addresses {
		if p, err := netip.ParsePrefix(strings.TrimSpace(address)); err == nil && p.Addr().Is4() {
			return p.Masked().String()
		}
	}
	return fallback
}

// allocateWireguardAddress returns the first free /32 host address in base that
// is not already present in used. The server holds the first host (.1), so
// allocation starts at the second host (.2).
func allocateWireguardAddress(used []string, base string) (string, error) {
	if base == "" {
		base = defaultWireguardBase
	}
	prefix, err := netip.ParsePrefix(base)
	if err != nil {
		return "", err
	}
	taken := make(map[netip.Addr]struct{}, len(used))
	for _, u := range used {
		if a := wireguardHostAddr(u); a.IsValid() {
			taken[a] = struct{}{}
		}
	}
	addr := prefix.Masked().Addr().Next().Next()
	for prefix.Contains(addr) {
		if _, ok := taken[addr]; !ok {
			return addr.String() + "/32", nil
		}
		addr = addr.Next()
	}
	return "", common.NewError("wireguard: no free address available in", base)
}

// normalizeWireguardAllowedIPs validates user-supplied allowedIPs entries and
// canonicalizes them: bare addresses become single-host prefixes, duplicates drop.
func normalizeWireguardAllowedIPs(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		p, err := netip.ParsePrefix(v)
		if err != nil {
			a, aErr := netip.ParseAddr(v)
			if aErr != nil {
				return nil, common.NewError("wireguard: invalid allowedIPs entry:", v)
			}
			p = netip.PrefixFrom(a, a.BitLen())
		}
		norm := p.String()
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	return out, nil
}

func wireguardAllowedIPsCollision(entries, used []string) string {
	taken := make(map[string]struct{}, len(used))
	for _, u := range used {
		taken[strings.TrimSpace(u)] = struct{}{}
	}
	for _, e := range entries {
		if _, ok := taken[e]; ok {
			return e
		}
	}
	return ""
}

// defaultWireguardClients fills in blank WireGuard credentials for newly added
// clients: a generated keypair when none was provided, a derived public key when
// only a private key was given, and a unique tunnel address allocated from the
// inbound's subnet. It mutates both the typed clients and the parallel raw client
// maps that get persisted into the inbound settings. Existing values are never
// overwritten, so editing a client never rotates its keys.
func defaultWireguardClients(existing, clients []model.Client, interfaceClients []any) error {
	return defaultWireguardClientsWithBase(existing, clients, interfaceClients, defaultWireguardBase, nil)
}

func defaultTunnelClients(protocol model.Protocol, settings string, existing, clients []model.Client, interfaceClients []any) error {
	if protocol != model.AmneziaWG {
		return defaultWireguardClients(existing, clients, interfaceClients)
	}
	addresses := wireguardSettingsAddresses(settings, defaultAmneziaWGAddress)
	base := wireguardBaseFromAddresses(addresses, defaultAmneziaWGBase)
	return defaultWireguardClientsWithBase(existing, clients, interfaceClients, base, addresses)
}

func defaultWireguardClientsWithBase(existing, clients []model.Client, interfaceClients []any, fallbackBase string, reserved []string) error {
	used := make([]string, 0)
	used = append(used, reserved...)
	for i := range existing {
		used = append(used, existing[i].AllowedIPs...)
	}
	base := wireguardAllocationBase(used, fallbackBase)
	for i := range clients {
		c := &clients[i]
		if c.PrivateKey == "" && c.PublicKey == "" {
			priv, pub, err := wgutil.GenerateWireguardKeypair()
			if err != nil {
				return err
			}
			c.PrivateKey = priv
			c.PublicKey = pub
		} else if c.PublicKey == "" && c.PrivateKey != "" {
			pub, err := wgutil.PublicKeyFromPrivate(c.PrivateKey)
			if err != nil {
				return err
			}
			c.PublicKey = pub
		}
		if len(c.AllowedIPs) == 0 {
			addr, err := allocateWireguardAddress(used, base)
			if err != nil {
				return err
			}
			c.AllowedIPs = []string{addr}
		} else {
			normalized, err := normalizeWireguardAllowedIPs(c.AllowedIPs)
			if err != nil {
				return err
			}
			if len(normalized) == 0 {
				return common.NewError("wireguard: allowedIPs has no usable entry")
			}
			if hit := wireguardAllowedIPsCollision(normalized, used); hit != "" {
				return common.NewError("wireguard: allowedIPs entry already used by another client:", hit)
			}
			c.AllowedIPs = normalized
		}
		used = append(used, c.AllowedIPs...)

		if i < len(interfaceClients) {
			if m, ok := interfaceClients[i].(map[string]any); ok {
				m["privateKey"] = c.PrivateKey
				m["publicKey"] = c.PublicKey
				m["allowedIPs"] = c.AllowedIPs
				if c.PreSharedKey != "" {
					m["preSharedKey"] = c.PreSharedKey
				}
				if c.KeepAlive > 0 {
					m["keepAlive"] = c.KeepAlive
				}
				interfaceClients[i] = m
			}
		}
	}
	return nil
}

func tunnelPeerPublicKey(peer map[string]any) string {
	for _, key := range []string{"publicKey", "public_key", "pubKey"} {
		if v, ok := peer[key].(string); ok {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func tunnelPeerMatchesDelete(peer map[string]any, email, publicKey string) bool {
	if email != "" {
		if e, ok := peer["email"].(string); ok && strings.TrimSpace(e) == email {
			return true
		}
	}
	if publicKey != "" && tunnelPeerPublicKey(peer) == publicKey {
		return true
	}
	return false
}

func removeTunnelPeerEntries(settings map[string]any, email, publicKey string) bool {
	peers, ok := settings["peers"].([]any)
	if !ok || len(peers) == 0 {
		return false
	}
	kept := make([]any, 0, len(peers))
	removed := false
	for _, raw := range peers {
		peer, ok := raw.(map[string]any)
		if !ok {
			kept = append(kept, raw)
			continue
		}
		if tunnelPeerMatchesDelete(peer, email, publicKey) {
			removed = true
			continue
		}
		kept = append(kept, raw)
	}
	if !removed {
		return false
	}
	if len(kept) == 0 {
		delete(settings, "peers")
	} else {
		settings["peers"] = kept
	}
	return true
}

func tunnelClientPublicKey(client *model.ClientRecord) string {
	if client == nil {
		return ""
	}
	return strings.TrimSpace(client.PublicKey)
}
