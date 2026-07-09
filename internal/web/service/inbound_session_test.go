package service

import "testing"

func TestResolveClientSessionTags(t *testing.T) {
	t.Helper()
	emailTags := map[string][]string{
		"alice": {"in-a", "in-b"},
		"bob":   {"in-a"},
	}
	tagDelta := map[string]int64{
		"in-a": 100,
		"in-b": 5,
	}
	pollActive := map[string]struct{}{"in-a": {}}
	got := resolveClientSessionTags(emailTags, tagDelta, pollActive, []string{"alice", "bob"}, nil)
	if got["alice"] != "in-a" {
		t.Fatalf("alice session tag = %q, want in-a", got["alice"])
	}
	if got["bob"] != "in-a" {
		t.Fatalf("bob session tag = %q, want in-a", got["bob"])
	}
}

func TestResolveClientSessionTagsStickyPrevious(t *testing.T) {
	t.Helper()
	emailTags := map[string][]string{"alice": {"in-a", "in-b"}}
	pollActive := map[string]struct{}{"in-a": {}, "in-b": {}}
	got := resolveClientSessionTags(
		emailTags,
		map[string]int64{},
		pollActive,
		[]string{"alice"},
		map[string]string{"alice": "in-b"},
	)
	if got["alice"] != "in-b" {
		t.Fatalf("sticky session tag = %q, want in-b", got["alice"])
	}
}
