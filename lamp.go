package tsl

import "strconv"

// Lamp is the 2-bit state of a tally lamp.
type Lamp uint8

// Tally lamp states.
const (
	LampOff   Lamp = 0
	LampRed   Lamp = 1
	LampGreen Lamp = 2
	LampAmber Lamp = 3
)

var lampNames = [...]string{"Off", "Red", "Green", "Amber"}

func (l Lamp) String() string {
	if int(l) < len(lampNames) {
		return lampNames[l]
	}
	return "Lamp(" + strconv.Itoa(int(l)) + ")"
}
