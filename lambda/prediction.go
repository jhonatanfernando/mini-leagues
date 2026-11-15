package main

import (
	"fmt"
	"slices"
)

// Podium holds driver IDs in finishing order. The same shape represents a
// member's prediction and the official result.
type Podium struct {
	P1 string
	P2 string
	P3 string
}

func (p Podium) drivers() [3]string {
	return [3]string{p.P1, p.P2, p.P3}
}

// ValidatePodium requires three distinct drivers from the race's eligible list.
// IDs are matched exactly: this function does not trim or change their case.
func ValidatePodium(podium Podium, eligibleDrivers []string) error {
	drivers := podium.drivers()
	for i, driver := range drivers {
		if driver == "" {
			return fmt.Errorf("P%d driver is required", i+1)
		}
		if !slices.Contains(eligibleDrivers, driver) {
			return fmt.Errorf("P%d driver must be eligible for the race", i+1)
		}
		if slices.Contains(drivers[:i], driver) {
			return fmt.Errorf("podium drivers must be distinct")
		}
	}
	return nil
}
