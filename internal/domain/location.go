package domain

import "time"

// LocationEntry is a browsable item with immutable local content when available.
type LocationEntry struct {
	ID, Name, Kind, Path, Repository string
	Size                             int64
	Modified                         time.Time
	Local                            LocalSource
	Children                         []LocationEntry
}

// LocationPage preserves server pagination and totals alongside displayable items.
type LocationPage struct {
	Entries  []LocationEntry
	Engine   Engine
	Next     string
	Total    int
	HasTotal bool
}
