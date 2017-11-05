package list

// This file contains AutoMode, which enumerates over autoselection modes.
// It also contains functions for converting AutoModes to and from strings.
// For the actual autoselection logic, see 'list.go'.

import "fmt"

// AutoMode is the type of autoselection modes.
type AutoMode int

const (
	// AutoOff is a selection mode that does nothing when a track ends.
	AutoOff AutoMode = iota
	// AutoDrop is a selection mode that ejects when a track ends.
	AutoDrop
	// AutoNext is a selection mode that loads the next track when a track ends.
	AutoNext
	// AutoShuffle is a selection mode that selects the next track in a pseudorandom permuation when a track ends.
	AutoShuffle
	// FirstAuto points to the first AutoMode constant.
	FirstAuto = AutoOff
	// LastAuto points to the last AutoMode constant.
	LastAuto = AutoNext
)

// String gets the Bifrost name of an AutoMode as a string.
func (a AutoMode) String() string {
	switch a {
	case AutoOff:
		return "off"
	case AutoDrop:
		return "drop"
	case AutoNext:
		return "next"
	case AutoShuffle:
		return "shuffle"
	default:
		return "?unknown?"
	}
}

// ParseAutoMode tries to parse an AutoMode from a string.
func ParseAutoMode(s string) (AutoMode, error) {
	switch s {
	case "off":
		return AutoOff, nil
	case "drop":
		return AutoDrop, nil
	case "next":
		return AutoNext, nil
	case "shuffle":
		return AutoShuffle, nil
	default:
		return AutoOff, fmt.Errorf("invalid automode")
	}
}
