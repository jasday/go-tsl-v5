package tsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLampString(t *testing.T) {
	assert.Equal(t, "Off", LampOff.String())
	assert.Equal(t, "Red", LampRed.String())
	assert.Equal(t, "Green", LampGreen.String())
	assert.Equal(t, "Amber", LampAmber.String())
	assert.Equal(t, "Lamp(4)", Lamp(4).String())
}
