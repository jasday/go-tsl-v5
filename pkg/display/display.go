package display

type Tally uint16

const (
	Off   Tally = 0
	Red   Tally = 1
	Green Tally = 2
	Amber Tally = 3
)

type Message struct {
	Index   uint16
	Control Control
	Data    Data
}

type Control struct {
	RightTally  Tally
	TextTally   Tally
	LeftTally   Tally
	Brightness  uint8
	ControlData bool
}

type Data struct {
	Length uint16
	Text   string
}
