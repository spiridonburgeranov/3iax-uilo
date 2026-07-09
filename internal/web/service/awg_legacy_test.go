package service

import (
	"testing"
)

func TestTunnelPeerMatchesDelete(t *testing.T) {
	t.Parallel()
	peer := map[string]any{
		"email":      "user@awg",
		"publicKey":  "abc123",
		"privateKey": "def456",
	}
	if !tunnelPeerMatchesDelete(peer, "user@awg", "") {
		t.Fatal("expected email match")
	}
	if !tunnelPeerMatchesDelete(peer, "", "abc123") {
		t.Fatal("expected public key match")
	}
	if tunnelPeerMatchesDelete(peer, "other@awg", "zzz") {
		t.Fatal("expected no match")
	}
}

func TestRemoveTunnelPeerEntries(t *testing.T) {
	t.Parallel()
	settings := map[string]any{
		"peers": []any{
			map[string]any{"email": "keep@awg", "publicKey": "keep"},
			map[string]any{"email": "drop@awg", "publicKey": "drop"},
		},
	}
	if !removeTunnelPeerEntries(settings, "drop@awg", "drop") {
		t.Fatal("expected peer removal")
	}
	peers, ok := settings["peers"].([]any)
	if !ok || len(peers) != 1 {
		t.Fatalf("expected one peer left, got %#v", settings["peers"])
	}
}
