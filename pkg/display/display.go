package display

type Lamp uint16

const (
	Off   Lamp = 0
	Red   Lamp = 1
	Green Lamp = 2
	Amber Lamp = 3
)

type Message struct {
	Index   uint16
	Control Control
	Data    Data
}

type Control struct {
	RightTally  Lamp
	TextTally   Lamp
	LeftTally   Lamp
	Brightness  uint8
	ControlData bool
}

type Data struct {
	Length uint16
	Text   string
}
