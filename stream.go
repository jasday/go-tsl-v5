package tsl

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

// Stream framing bytes. A frame starts with DLE/STX and any DLE in the packet,
// including the PBC, is sent as DLE/DLE. Byte counts exclude the stuffed bytes.
const (
	dle = 0xFE
	stx = 0x02
)

// Encoder writes DLE/STX framed packets to a byte stream such as TCP or serial.
type Encoder struct {
	w   io.Writer
	buf []byte
}

// NewEncoder returns an Encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes p to the stream as a single frame.
func (e *Encoder) Encode(p *Packet) error {
	if p == nil {
		return fmt.Errorf("%w: nil packet", ErrInvalidValue)
	}
	// Encode the packet after the frame header, then stuff it into place.
	packet, err := p.appendTo(e.buf[:0])
	if err != nil {
		return err
	}
	n := len(packet)
	frame := append(packet, dle, stx)
	for _, b := range packet[:n] {
		frame = append(frame, b)
		if b == dle {
			frame = append(frame, dle)
		}
	}
	e.buf = frame
	_, err = e.w.Write(frame[n:])
	return err
}

// Decoder reads DLE/STX framed packets from a byte stream such as TCP or serial.
type Decoder struct {
	r *bufio.Reader
	// inFrame is set when a DLE/STX has already been consumed, after a frame
	// was cut short by the start of the next one.
	inFrame bool
	buf     []byte
}

// NewDecoder returns a Decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &Decoder{r: br}
}

// Decode reads the next frame into p. Bytes before a DLE/STX are skipped.
//
// It returns io.EOF if the stream ends between frames and io.ErrUnexpectedEOF
// if it ends inside one. A frame that cannot be parsed returns an error
// wrapping ErrMalformed or ErrShortPacket; decoding can continue with the next call.
func (d *Decoder) Decode(p *Packet) error {
	if !d.inFrame {
		if err := d.sync(); err != nil {
			return err
		}
	}
	d.inFrame = false

	b, err := d.readUnstuffed(d.buf[:0], 2)
	if err != nil {
		return err
	}
	pbc := int(binary.LittleEndian.Uint16(b))
	if b, err = d.readUnstuffed(b, pbc); err != nil {
		return err
	}
	d.buf = b
	return Unmarshal(b, p)
}

// sync consumes bytes up to and including the next DLE/STX.
func (d *Decoder) sync() error {
	for {
		b, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		if b != dle {
			continue
		}
		b, err = d.r.ReadByte()
		if err != nil {
			return err
		}
		if b == stx {
			return nil
		}
		// DLE/DLE is a stuffed data byte; anything else is noise.
	}
}

// readUnstuffed appends n unstuffed bytes from the frame to b.
func (d *Decoder) readUnstuffed(b []byte, n int) ([]byte, error) {
	for range n {
		c, err := d.r.ReadByte()
		if err != nil {
			return nil, unexpectedEOF(err)
		}
		if c == dle {
			c, err = d.r.ReadByte()
			if err != nil {
				return nil, unexpectedEOF(err)
			}
			switch c {
			case dle:
			case stx:
				d.inFrame = true
				return nil, fmt.Errorf("%w: frame interrupted by DLE/STX", ErrMalformed)
			default:
				return nil, fmt.Errorf("%w: DLE followed by %#02x", ErrMalformed, c)
			}
		}
		b = append(b, c)
	}
	return b, nil
}

func unexpectedEOF(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}
