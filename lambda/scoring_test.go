package main

import (
	"strings"
	"testing"
)

func TestScorePrediction(t *testing.T) {
	eligible := []string{"VER", "NOR", "LEC", "PIA", "RUS", "HAM"}
	result := Podium{P1: "VER", P2: "NOR", P3: "LEC"}
	for _, tc := range []struct {
		name string
		pick Podium
		want int
	}{
		{"all exact without stacking", Podium{P1: "VER", P2: "NOR", P3: "LEC"}, 15},
		{"one exact two displaced", Podium{P1: "VER", P2: "LEC", P3: "NOR"}, 9},
		{"second exact", Podium{P1: "LEC", P2: "NOR", P3: "VER"}, 9},
		{"third exact", Podium{P1: "NOR", P2: "VER", P3: "LEC"}, 9},
		{"cyclic permutation", Podium{P1: "NOR", P2: "LEC", P3: "VER"}, 6},
		{"reverse cycle", Podium{P1: "LEC", P2: "VER", P3: "NOR"}, 6},
		{"no matches", Podium{P1: "PIA", P2: "RUS", P3: "HAM"}, 0},
		{"one exact", Podium{P1: "VER", P2: "RUS", P3: "HAM"}, 5},
		{"one displaced", Podium{P1: "PIA", P2: "VER", P3: "HAM"}, 2},
		{"mixed matches", Podium{P1: "VER", P2: "LEC", P3: "HAM"}, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ScorePrediction(tc.pick, result, eligible)
			if err != nil || got != tc.want {
				t.Fatalf("score = %d, error = %v; want %d", got, err, tc.want)
			}
		})
	}
}

func TestScorePredictionRejectsInvalidPodiums(t *testing.T) {
	valid := Podium{P1: "VER", P2: "NOR", P3: "LEC"}
	eligible := []string{"VER", "NOR", "LEC"}
	for _, tc := range []struct {
		name         string
		pick, result Podium
		eligible     []string
		wantError    string
	}{
		{"duplicate prediction", Podium{P1: "VER", P2: "VER", P3: "LEC"}, valid, eligible, "prediction"},
		{"duplicate result", valid, Podium{P1: "VER", P2: "NOR", P3: "NOR"}, eligible, "result"},
		{"incomplete prediction", Podium{P1: "VER"}, valid, eligible, "prediction"},
		{"incomplete result", valid, Podium{P1: "VER"}, eligible, "result"},
		{"unknown prediction", Podium{P1: "UNKNOWN", P2: "NOR", P3: "LEC"}, valid, eligible, "prediction"},
		{"unknown result", valid, Podium{P1: "UNKNOWN", P2: "NOR", P3: "LEC"}, eligible, "result"},
		{"missing eligible drivers", valid, valid, nil, "eligible"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			score, err := ScorePrediction(tc.pick, tc.result, tc.eligible)
			if score != 0 || err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("score = %d, error = %v; want zero and error containing %q", score, err, tc.wantError)
			}
		})
	}
}
