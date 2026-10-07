package tsl

import (
	"bufio"
	"encoding/binary"
	"io"
)

// Start of Message sequence is DLE/STX
const (
	dle = 0xFE
	stx = 0x02
)

// ReadMessage reads the next complete TSL v5 message from a byte stream such as TCP or serial.
func ReadMessage(r *bufio.Reader) (*Packet, error) {
	buf := make([]byte, 1)

Delimiter:
	for {
		for {
			if _, err := r.Read(buf); err != nil {
				return nil, err
			}
			if buf[0] == dle {
				break
			}
		}

		for {
			if _, err := r.Read(buf); err != nil {
				return nil, err
			}
			if buf[0] == dle {
				continue
			}
			if buf[0] == stx {
				break Delimiter
			}
			break
		}
	}

	length := make([]byte, 2)
	if _, err := io.ReadFull(r, length); err != nil {
		return nil, err
	}

	buffer := make([]byte, binary.LittleEndian.Uint16(length))
	if _, err := io.ReadFull(r, buffer); err != nil {
		return nil, err
	}

	result := make([]byte, 0, len(buffer)+2)
	result = append(result, length...)
	for i := 0; i < len(buffer); i++ {
		b := buffer[i]
		if b == dle && i+1 < len(buffer) && buffer[i+1] == dle {
			result = append(result, dle)
			i++
		} else {
			result = append(result, b)
		}
	}

	p := &Packet{}
	if err := Unmarshal(result, p); err != nil {
		return nil, err
	}
	return p, nil
}

// WriteMessage encodes p into a DLE/STX framed TSL v5 message and writes it to w.
func WriteMessage(w io.Writer, p *Packet) error {
	payload, err := Marshal(p)
	if err != nil {
		return err
	}

	stuffed := make([]byte, 0, len(payload))
	for i, b := range payload {
		stuffed = append(stuffed, b)
		if i == 0 || i == 1 {
			continue
		}
		if b == dle {
			stuffed = append(stuffed, dle)
		}
	}

	frame := make([]byte, 0, 4+len(stuffed))
	frame = append(frame, dle, stx)
	frame = binary.LittleEndian.AppendUint16(frame, uint16(len(stuffed)))
	frame = append(frame, stuffed...)

	_, err = w.Write(frame)
	return err
}
