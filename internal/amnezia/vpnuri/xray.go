package vpnuri

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
)

var xrayAddressPattern = regexp.MustCompile(`"address"\s*:\s*"([^"]+)"`)

func FromXrayConfig(configJSON []byte, description, hostName, container string) (string, error) {
	configJSON = bytesTrimSpace(configJSON)
	if len(configJSON) == 0 {
		return "", common.NewError("xray config is required")
	}
	if !json.Valid(configJSON) {
		return "", common.NewError("invalid xray config json")
	}
	if container == "" {
		container = "amnezia-xray"
	}
	protoKey := "xray"
	if container == "amnezia-ssxray" {
		protoKey = "ssxray"
	}
	if hostName == "" {
		hostName = extractXrayHostName(string(configJSON))
	}
	lastConfig := map[string]any{
		"config":             string(configJSON),
		"last_config":        string(configJSON),
		"isThirdPartyConfig": true,
	}
	containers := []map[string]any{{
		"container": container,
		protoKey:    lastConfig,
	}}
	outer := map[string]any{
		"containers":       containers,
		"defaultContainer": container,
		"description":      description,
		"hostName":         hostName,
	}
	return Encode(outer)
}

func extractXrayHostName(config string) string {
	match := xrayAddressPattern.FindStringSubmatch(config)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func bytesTrimSpace(data []byte) []byte {
	return []byte(strings.TrimSpace(string(data)))
}
