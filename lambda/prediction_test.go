package main

import (
	"strings"
	"testing"
)

func TestValidatePodium(t *testing.T) {
	eligible := []string{"VER", "NOR", "LEC", "PIA"}
	for _, tc := range []struct {
		name      string
		podium    Podium
		eligible  []string
		wantError string
	}{
		{"valid", Podium{P1: "VER", P2: "NOR", P3: "LEC"}, eligible, ""},
		{"different order", Podium{P1: "LEC", P2: "VER", P3: "NOR"}, eligible, ""},
		{"missing first", Podium{P2: "NOR", P3: "LEC"}, eligible, "P1"},
		{"missing second", Podium{P1: "VER", P3: "LEC"}, eligible, "P2"},
		{"missing third", Podium{P1: "VER", P2: "NOR"}, eligible, "P3"},
		{"empty podium", Podium{}, eligible, "P1"},
		{"duplicate first second", Podium{P1: "VER", P2: "VER", P3: "LEC"}, eligible, "distinct"},
		{"duplicate first third", Podium{P1: "VER", P2: "NOR", P3: "VER"}, eligible, "distinct"},
		{"duplicate second third", Podium{P1: "VER", P2: "NOR", P3: "NOR"}, eligible, "distinct"},
		{"unknown first", Podium{P1: "UNKNOWN", P2: "NOR", P3: "LEC"}, eligible, "P1"},
		{"unknown second", Podium{P1: "VER", P2: "UNKNOWN", P3: "LEC"}, eligible, "P2"},
		{"unknown third", Podium{P1: "VER", P2: "NOR", P3: "UNKNOWN"}, eligible, "P3"},
		{"no eligible drivers", Podium{P1: "VER", P2: "NOR", P3: "LEC"}, nil, "eligible"},
		{"whitespace", Podium{P1: " ", P2: "NOR", P3: "LEC"}, eligible, "P1"},
		{"padded ID", Podium{P1: " VER ", P2: "NOR", P3: "LEC"}, eligible, "P1"},
		{"wrong case", Podium{P1: "ver", P2: "NOR", P3: "LEC"}, eligible, "P1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePodium(tc.podium, tc.eligible)
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected validation error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error = %v; want error containing %q", err, tc.wantError)
			}
		})
	}
}
