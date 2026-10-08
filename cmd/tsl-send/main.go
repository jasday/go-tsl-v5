// Command tsl-send sends a single TSL v5 display message over UDP or TCP.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"

	tsl "github.com/jasday/go-tsl-v5"
)

func main() {
	proto := flag.String("proto", "udp", "transport: udp or tcp")
	addr := flag.String("addr", "127.0.0.1:5900", "address to send to")
	screen := flag.Uint("screen", 0, "screen index (65535 = broadcast)")
	index := flag.Uint("index", 0, "display index (65535 = broadcast)")
	text := flag.String("text", "", "display text")
	lh := flag.String("lh", "off", "left tally: off, red, green or amber")
	txt := flag.String("txt", "off", "text tally: off, red, green or amber")
	rh := flag.String("rh", "off", "right tally: off, red, green or amber")
	brightness := flag.Uint("brightness", 3, "brightness 0-3")
	unicode := flag.Bool("unicode", false, "send text as UTF-16LE")
	flag.Parse()

	p, err := buildPacket(*screen, *index, *text, *lh, *txt, *rh, *brightness, *unicode)
	if err != nil {
		log.Fatal(err)
	}
	if err := send(*proto, *addr, p); err != nil {
		log.Fatal(err)
	}
}

func buildPacket(screen, index uint, text, lh, txt, rh string, brightness uint, unicode bool) (*tsl.Packet, error) {
	if screen > 0xFFFF || index > 0xFFFF {
		return nil, errors.New("screen and index must be at most 65535")
	}
	if brightness > 3 {
		return nil, errors.New("brightness must be 0-3")
	}
	d := tsl.Display{Index: uint16(index), Brightness: uint8(brightness), Text: text}
	for _, l := range []struct {
		name string
		dst  *tsl.Lamp
	}{{lh, &d.LeftTally}, {txt, &d.TextTally}, {rh, &d.RightTally}} {
		lamp, err := parseLamp(l.name)
		if err != nil {
			return nil, err
		}
		*l.dst = lamp
	}
	return &tsl.Packet{Unicode: unicode, Screen: uint16(screen), Displays: []tsl.Display{d}}, nil
}

func parseLamp(s string) (tsl.Lamp, error) {
	for _, l := range []tsl.Lamp{tsl.LampOff, tsl.LampRed, tsl.LampGreen, tsl.LampAmber} {
		if strings.EqualFold(s, l.String()) {
			return l, nil
		}
	}
	return 0, fmt.Errorf("unknown lamp %q: want off, red, green or amber", s)
}

func send(proto, addr string, p *tsl.Packet) error {
	conn, err := net.Dial(proto, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	switch proto {
	case "udp":
		datagrams, err := tsl.MarshalUDP(p)
		if err != nil {
			return err
		}
		for _, b := range datagrams {
			if _, err := conn.Write(b); err != nil {
				return err
			}
		}
		return nil
	case "tcp":
		return tsl.NewEncoder(conn).Encode(p)
	default:
		return fmt.Errorf("unknown protocol %q", proto)
	}
}
