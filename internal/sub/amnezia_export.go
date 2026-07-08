package sub

import (
	"encoding/json"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

func init() {
	service.RegisterAmneziaXrayConfigExporter(exportClientXrayConfig)
}

func exportClientXrayConfig(
	remarkTemplate string,
	mux string,
	rules string,
	finalMask string,
	host string,
	inbound *model.Inbound,
	client model.Client,
) ([]byte, string, error) {
	if inbound == nil {
		return nil, "", common.NewError("inbound is required")
	}
	subSvc := NewSubService(remarkTemplate).ForRequest(host)
	jsonSvc := NewSubJsonService(mux, rules, finalMask, subSvc)
	subSvc.projectThroughFallbackMaster(inbound)
	configs := jsonSvc.getConfig(subSvc, inbound, client, host)
	if len(configs) == 0 {
		return nil, "", common.NewError("no xray client config generated")
	}
	config := configs[0]
	hostName := extractXrayConfigHostName(config)
	return config, hostName, nil
}

func extractXrayConfigHostName(config []byte) string {
	var doc map[string]any
	if err := json.Unmarshal(config, &doc); err != nil {
		return ""
	}
	outbounds, ok := doc["outbounds"].([]any)
	if !ok || len(outbounds) == 0 {
		return ""
	}
	first, ok := outbounds[0].(map[string]any)
	if !ok {
		return ""
	}
	settings, ok := first["settings"].(map[string]any)
	if !ok {
		return ""
	}
	switch first["protocol"] {
	case "vmess", "vless":
		if servers, ok := settings["vnext"].([]any); ok && len(servers) > 0 {
			if server, ok := servers[0].(map[string]any); ok {
				if address, ok := server["address"].(string); ok {
					return address
				}
			}
		}
	case "trojan", "shadowsocks", "hysteria":
		if servers, ok := settings["servers"].([]any); ok && len(servers) > 0 {
			if server, ok := servers[0].(map[string]any); ok {
				if address, ok := server["address"].(string); ok {
					return address
				}
			}
		}
	}
	return ""
}
