package main

import (
	"fmt"
	"slices"
)

// ScorePrediction awards 5 points for an exact position, 2 for a driver in a
// different podium position, and 0 otherwise. These awards never stack.
// Invalid predictions or results return zero and an error, not a partial score.
func ScorePrediction(prediction, result Podium, eligibleDrivers []string) (int, error) {
	if err := ValidatePodium(prediction, eligibleDrivers); err != nil {
		return 0, fmt.Errorf("invalid prediction: %w", err)
	}
	if err := ValidatePodium(result, eligibleDrivers); err != nil {
		return 0, fmt.Errorf("invalid result: %w", err)
	}

	official := result.drivers()
	total := 0
	for i, driver := range prediction.drivers() {
		switch {
		case driver == official[i]:
			total += 5
		case slices.Contains(official[:], driver):
			total += 2
		}
	}
	return total, nil
}
