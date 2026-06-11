package main

import (
	"strings"
	"testing"
)

func TestDeduplicateSubnets(t *testing.T) {
	input := `/ip firewall address-list remove [find list="test"];
add list=test address=1.2.3.0/24 comment=site1
add list=test address=4.5.6.0/24 comment=site2
add list=test address=1.2.3.0/24 comment=site3
add list=test address=youtube.com comment=site4
add list=test address=youtube.com comment=site5
add list=test address=4.5.6.0/24 comment=site6
`
	result := string(deduplicateSubnets([]byte(input)).data)
	lines := strings.Split(strings.TrimSpace(result), "\n")

	want := 5
	if len(lines) != want {
		t.Fatalf("got %d lines, want %d:\n%s", len(lines), want, result)
	}

	if strings.Count(result, "1.2.3.0/24") != 1 {
		t.Errorf("expected one occurrence of 1.2.3.0/24")
	}
	if strings.Count(result, "4.5.6.0/24") != 1 {
		t.Errorf("expected one occurrence of 4.5.6.0/24")
	}
	if strings.Count(result, "youtube.com") != 2 {
		t.Errorf("domain lines should not be deduplicated")
	}

	stats := deduplicateSubnets([]byte(input))
	if stats.linesIn != 7 {
		t.Errorf("linesIn = %d, want 7", stats.linesIn)
	}
	if stats.linesOut != 5 {
		t.Errorf("linesOut = %d, want 5", stats.linesOut)
	}
	if stats.removed != 2 {
		t.Errorf("removed = %d, want 2", stats.removed)
	}
}

func TestExtractCIDR(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"add list=test address=1.2.3.0/24 comment=foo", "1.2.3.0/24"},
		{"add list=test address=youtube.com comment=foo", ""},
		{"1.9.0.0/16", "1.9.0.0/16"},
		{"no cidr here", ""},
	}

	for _, tt := range tests {
		if got := extractCIDR(tt.line); got != tt.want {
			t.Errorf("extractCIDR(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestBuildUpstreamURL(t *testing.T) {
	got, err := buildUpstreamURL("https://iplist.opencck.org/", "format=mikrotik&data=domains&site=youtube.com")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://iplist.opencck.org/?format=mikrotik&data=domains&site=youtube.com"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
