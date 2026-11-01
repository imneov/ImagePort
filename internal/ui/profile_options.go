package ui

import "github.com/imneov/ImagePort/internal/domain"

// profileOptions creates unambiguous display values so Fyne selection preserves saved IDs.
func profileOptions(profiles []domain.Profile) []string {
	options := make([]string, len(profiles))
	counts := make(map[string]int)
	for index, profile := range profiles {
		options[index] = profile.Name + " · " + profile.Host
		counts[options[index]]++
	}
	for index, profile := range profiles {
		if counts[options[index]] > 1 {
			options[index] += " · " + profile.ID
		}
	}
	return options
}
