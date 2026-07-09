package service

import (
    "encoding/json"
    "path/filepath"
    "testing"

    xconf "github.com/xtls/xray-core/infra/conf"
    "github.com/mhsanaei/3x-ui/v3/internal/database"
    "github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestGetXrayConfigDefaultVlessInboundLoads(t *testing.T) {
    setupSettingTestDB(t)
    db := database.GetDB()
    stream := `{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"none"}}}`
    settings := `{"decryption":"none","encryption":"none","clients":[{"id":"11111111-2222-4333-8444-555555555555","email":"t@test","enable":true,"flow":""}]}`
    in := &model.Inbound{
        Tag: "in-443-tcp", Enable: true, Port: 443, Protocol: model.VLESS,
        Settings: settings, StreamSettings: stream,
        Sniffing: `{"enabled":true,"destOverride":["http","tls","quic","fakedns"]}`,
    }
    if err := db.Create(in).Error; err != nil {
        t.Fatal(err)
    }
    svc := &XrayService{}
    cfg, err := svc.GetXrayConfig()
    if err != nil {
        t.Fatalf("GetXrayConfig: %v", err)
    }
    raw, err := json.Marshal(cfg)
    if err != nil {
        t.Fatal(err)
    }
    var root map[string]any
    if err := json.Unmarshal(raw, &root); err != nil {
        t.Fatal(err)
    }
    outbounds, _ := root["outbounds"].([]any)
    for _, ob := range outbounds {
        m, _ := ob.(map[string]any)
        tag, _ := m["tag"].(string)
        b, _ := json.Marshal(m)
        c := new(xconf.OutboundDetourConfig)
        if err := json.Unmarshal(b, c); err != nil {
            t.Fatalf("outbound %s unmarshal: %v", tag, err)
        }
        if _, err := c.Build(); err != nil {
            t.Fatalf("outbound %s build: %v\n%s", tag, err, string(b))
        }
    }
}
