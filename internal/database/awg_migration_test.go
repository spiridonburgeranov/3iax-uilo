package database

import (
	"encoding/json"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func createAWGInbound(t *testing.T, remark string, port int, peers []any) *model.Inbound {
	t.Helper()
	settings, err := json.Marshal(map[string]any{
		"secretKey": "c2VjcmV0LWtleS1iYXNlNjQtMzJieXRlcy1wbGFjZWg=",
		"address":   "10.66.66.1/24",
		"mtu":       1420,
		"peers":     peers,
	})
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	in := &model.Inbound{
		UserId:   1,
		Remark:   remark,
		Port:     port,
		Protocol: model.AmneziaWG,
		Settings: string(settings),
		Tag:      remark,
	}
	if err := db.Create(in).Error; err != nil {
		t.Fatalf("create awg inbound: %v", err)
	}
	return in
}

func TestSeedAmneziawgPeersToClientsCreatesClients(t *testing.T) {
	initWGMigrationDB(t)
	in := createAWGInbound(t, "awg-server", 51821, []any{
		wgPeer("peer-one", "priv1", "pub1", "10.66.66.2/32", 25),
	})
	clearAWGSeederHistory(t, "AmneziawgPeersToClients")
	if err := seedAmneziawgPeersToClients(); err != nil {
		t.Fatalf("seedAmneziawgPeersToClients: %v", err)
	}
	settings := reloadInboundSettings(t, in.Id)
	if _, hasPeers := settings["peers"]; hasPeers {
		t.Fatal("expected peers to be removed")
	}
	clients, ok := settings["clients"].([]any)
	if !ok || len(clients) != 1 {
		t.Fatalf("expected one client, got %#v", settings["clients"])
	}
	var count int64
	if err := db.Model(&model.ClientRecord{}).Count(&count).Error; err != nil {
		t.Fatalf("count clients: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one client record, got %d", count)
	}
}

func TestPurgeLegacyAwgClientsRemovesKeylessRows(t *testing.T) {
	initWGMigrationDB(t)
	if err := db.Create(&model.AwgClient{Email: "ghost@awg"}).Error; err != nil {
		t.Fatalf("create legacy awg client: %v", err)
	}
	clearAWGSeederHistory(t, "PurgeLegacyAwgClients")
	if err := purgeLegacyAwgClients(); err != nil {
		t.Fatalf("purgeLegacyAwgClients: %v", err)
	}
	var count int64
	if err := db.Model(&model.AwgClient{}).Count(&count).Error; err != nil {
		t.Fatalf("count awg clients: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected keyless legacy awg client to be purged, got %d", count)
	}
}

func clearAWGSeederHistory(t *testing.T, name string) {
	t.Helper()
	if err := db.Where("seeder_name = ?", name).Delete(&model.HistoryOfSeeders{}).Error; err != nil {
		t.Fatalf("clear history: %v", err)
	}
}
