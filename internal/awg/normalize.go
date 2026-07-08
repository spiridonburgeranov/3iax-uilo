package awg

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const defaultProvisionDNS = "1.1.1.1,2606:4700:4700::1111"

func NeedsObfuscationProvision(parsed inboundSettings) bool {
	if parsed.Jc <= 0 || parsed.Jmin <= 0 || parsed.Jmax <= 0 {
		return true
	}
	if parsed.S1 <= 0 && parsed.S2 <= 0 && parsed.S3 <= 0 && parsed.S4 <= 0 {
		return true
	}
	if isWeakHeader(parsed.H1) || isWeakHeader(parsed.H2) || isWeakHeader(parsed.H3) || isWeakHeader(parsed.H4) {
		return true
	}
	for _, value := range []string{parsed.I1, parsed.I2, parsed.I3, parsed.I4, parsed.I5} {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}

func isWeakHeader(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if strings.Contains(value, "-") {
		return false
	}
	single, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	return single >= 1 && single <= 4
}

func NormalizeInboundSettings(settingsJSON string, inboundPort int, snapshot ResourceSnapshot) (string, int, error) {
	var settings map[string]any
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return "", inboundPort, err
	}
	if settings == nil {
		settings = map[string]any{}
	}
	parsed, err := ParseInboundSettings(settingsJSON)
	if err != nil {
		return "", inboundPort, err
	}

	plan, planErr := BuildProvisionPlan(snapshot)
	usePlan := planErr == nil

	if strings.TrimSpace(parsed.DNS) == "" {
		if usePlan {
			settings["dns"] = plan.DNS
		} else {
			settings["dns"] = defaultProvisionDNS
		}
	}
	if parsed.MTU <= 0 {
		if usePlan {
			settings["mtu"] = plan.MTU
		} else {
			settings["mtu"] = 1420
		}
	}
	if strings.TrimSpace(parsed.Address) == "" && usePlan {
		settings["address"] = plan.ServerAddress
	}
	if strings.TrimSpace(parsed.AwgInterface) == "" && usePlan {
		settings["awgInterface"] = plan.InterfaceName
	}
	port := inboundPort
	if port <= 0 && usePlan {
		port = plan.ListenPort
	}
	if strings.TrimSpace(parsed.SecretKey) == "" && usePlan {
		settings["secretKey"] = plan.PrivateKey
	}

	if NeedsObfuscationProvision(parsed) {
		var obf ObfuscationParams
		if usePlan {
			obf = ObfuscationParams{
				Jc: plan.Jc, Jmin: plan.Jmin, Jmax: plan.Jmax,
				S1: plan.S1, S2: plan.S2, S3: plan.S3, S4: plan.S4,
				H1: plan.H1, H2: plan.H2, H3: plan.H3, H4: plan.H4,
				I1: plan.I1, I2: plan.I2, I3: plan.I3, I4: plan.I4, I5: plan.I5,
			}
		} else {
			obf = GenerateObfuscationParams()
		}
		applyObfuscation(settings, obf)
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", port, err
	}
	return string(out), port, nil
}

func applyObfuscation(settings map[string]any, obf ObfuscationParams) {
	settings["jc"] = obf.Jc
	settings["jmin"] = obf.Jmin
	settings["jmax"] = obf.Jmax
	settings["s1"] = obf.S1
	settings["s2"] = obf.S2
	settings["s3"] = obf.S3
	settings["s4"] = obf.S4
	settings["h1"] = obf.H1
	settings["h2"] = obf.H2
	settings["h3"] = obf.H3
	settings["h4"] = obf.H4
	settings["i1"] = obf.I1
	settings["i2"] = obf.I2
	settings["i3"] = obf.I3
	settings["i4"] = obf.I4
	settings["i5"] = obf.I5
}

func PlanToSettingsMap(plan ProvisionPlan) map[string]any {
	return map[string]any{
		"secretKey":    plan.PrivateKey,
		"address":      plan.ServerAddress,
		"dns":          plan.DNS,
		"awgInterface": plan.InterfaceName,
		"mtu":          plan.MTU,
		"jc":           plan.Jc,
		"jmin":         plan.Jmin,
		"jmax":         plan.Jmax,
		"s1":           plan.S1,
		"s2":           plan.S2,
		"s3":           plan.S3,
		"s4":           plan.S4,
		"h1":           plan.H1,
		"h2":           plan.H2,
		"h3":           plan.H3,
		"h4":           plan.H4,
		"i1":           plan.I1,
		"i2":           plan.I2,
		"i3":           plan.I3,
		"i4":           plan.I4,
		"i5":           plan.I5,
		"postUp":       "",
		"postDown":     "",
		"clients":      []any{},
		"peers":        []any{},
	}
}

func FormatInterfaceTag(iface string) string {
	iface = strings.TrimSpace(iface)
	if iface == "" {
		return "inbound-awg"
	}
	return fmt.Sprintf("inbound-%s", iface)
}
