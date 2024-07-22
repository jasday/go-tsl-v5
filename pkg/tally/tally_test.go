package tally

import (
	"testing"

	"github.com/jasday/go-tsl-v5/pkg/display"
	"github.com/stretchr/testify/assert"
)

// TestEmptyTallyByteRepresentation checks that parsing a tally to a byte buffer returns the correct output
func TestEmptyTallyByteRepresentation(t *testing.T) {
	toTest := Tally{
		Version: 0,
		Flags: Flags{
			UnicodeStrings: false,
			ControlData:    false,
		},
		Screen: 0,
		DisplayMessages: []display.Message{
			{
				Index: 0,
				Control: display.Control{
					RightTally:  display.Off,
					TextTally:   display.Off,
					LeftTally:   display.Off,
					Brightness:  3,
					ControlData: false,
				},
				Data: display.Data{
					Text: "Test",
				},
			},
		},
	}
	want := []byte{0xe, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xc0, 0x0, 0x4, 0x0, 0x54, 0x65, 0x73, 0x74}
	got := toTest.Bytes()

	assert.Equal(t, want, got, "Byte arrays should match")
}

func TestFromBufferReturnsExpectedTally(t *testing.T) {
	input := []byte{0xe, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xc0, 0x0, 0x4, 0x0, 0x54, 0x65, 0x73, 0x74}

	want := Tally{
		Version: 0,
		Flags: Flags{
			UnicodeStrings: false,
			ControlData:    false,
		},
		Screen: 0,
		DisplayMessages: []display.Message{
			{
				Index: 0,
				Control: display.Control{
					RightTally:  display.Off,
					TextTally:   display.Off,
					LeftTally:   display.Off,
					Brightness:  3,
					ControlData: false,
				},
				Data: display.Data{
					Text: "Test",
				},
			},
		},
	}

	got := FromBuffer(input)
	compareTallies(t, want, *got)
}

func compareTallies(t *testing.T, tally1, tally2 Tally) {
	if tally1.Flags != tally2.Flags {
		assert.Fail(t, "Tally Flags do not match.")
	}

	if tally1.Screen != tally2.Screen {
		assert.Fail(t, "Tally Screen values do not match.")
	}

	if tally1.Version != tally2.Version {
		assert.Fail(t, "Tally Versions do not match.")
	}

	for i, msg := range tally1.DisplayMessages {
		if msg != tally2.DisplayMessages[i] {
			if msg.Control.Brightness != tally2.DisplayMessages[i].Control.Brightness {
				assert.Failf(t, "Control Brightness does not match", "Message Index: %d | Tally 1: %s - Tally 2: %s", i, msg.Control.Brightness, tally2.DisplayMessages[i].Control.Brightness)
			}

			if msg.Control.LeftTally != tally2.DisplayMessages[i].Control.LeftTally {
				assert.Failf(t, "Control Left Tallies do not match", "Message Index: %d | Tally 1: %s - Tally 2: %s", i, msg.Control.LeftTally, tally2.DisplayMessages[i].Control.LeftTally)
			}

			if msg.Control.RightTally != tally2.DisplayMessages[i].Control.RightTally {
				assert.Failf(t, "Control Right Tallies do not match", "Message Index: %d | Tally 1: %s - Tally 2: %s", i, msg.Control.RightTally, tally2.DisplayMessages[i].Control.RightTally)
			}

			if msg.Control.TextTally != tally2.DisplayMessages[i].Control.TextTally {
				assert.Failf(t, "Control Text Tallies do not match", "Message Index: %d | Tally 1: %s - Tally 2: %s", i, msg.Control.TextTally, tally2.DisplayMessages[i].Control.TextTally)
			}

			if msg.Control.ControlData != tally2.DisplayMessages[i].Control.ControlData {
				assert.Failf(t, "Control Data does not match", "Message Index: %d | Tally 1: %s - Tally 2: %s", i, msg.Control.ControlData, tally2.DisplayMessages[i].Control.ControlData)
			}

			if msg.Index != tally2.DisplayMessages[i].Index {
				assert.Failf(t, "Index does not match", "Message Index: %d | Tally 1: %s - Tally 2: %s", i, msg.Index, tally2.DisplayMessages[i].Index)
			}

			if msg.Data.Text != tally2.DisplayMessages[i].Data.Text {
				assert.Failf(t, "Text does not match", "Message Index: %d | Tally 1: %s - Tally 2: %s", i, msg.Data.Text, tally2.DisplayMessages[i].Data.Text)
			}

		}
	}
}
