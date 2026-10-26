package domain

import (
	"fmt"
	"time"
)

// SeasonStartOn uses the October–September calendar confirmed by the user.
func SeasonStartOn(date time.Time) int {
	if date.Month() < time.October {
		return date.Year() - 1
	}
	return date.Year()
}
func SeasonLabel(start int) string       { return fmt.Sprintf("%d/%02d", start, (start+1)%100) }
func SeasonAge(start, birthYear int) int { return start + 1 - birthYear }
func IndividualGroup(age int) (low, high int) {
	if age < 20 {
		return 0, 0
	}
	if age >= 100 {
		return 100, 0
	}
	low = age / 5 * 5
	return low, low + 4
}
func RelayCategories() []int { return []int{80, 100, 120, 160, 200, 240, 280, 320} }
func ValidRelayCategory(category int) bool {
	for _, v := range RelayCategories() {
		if v == category {
			return true
		}
	}
	return false
}
func NaturalRelayCategory(age int) int {
	if age < 80 {
		return 0
	}
	category := 80
	for _, v := range RelayCategories() {
		if age >= v {
			category = v
		}
	}
	return category
}

func ageReference(start int) string { return fmt.Sprintf("31/12/%d", start+1) }
