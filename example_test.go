package tsl_test

import (
	"bytes"
	"fmt"

	tsl "github.com/jasday/go-tsl-v5"
)

func ExampleMarshal() {
	p := tsl.Packet{
		Screen: 0,
		Displays: []tsl.Display{
			{Index: 1, LeftTally: tsl.LampRed, Brightness: 3, Text: "CAM 1"},
		},
	}
	b, err := tsl.Marshal(&p)
	if err != nil {
		panic(err)
	}
	fmt.Printf("% x\n", b)
	// Output: 0f 00 00 00 00 00 01 00 d0 00 05 00 43 41 4d 20 31
}

func ExampleUnmarshal() {
	b := []byte{0x0f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0xd0, 0x00, 0x05, 0x00, 'C', 'A', 'M', ' ', '1'}

	var p tsl.Packet
	if err := tsl.Unmarshal(b, &p); err != nil {
		panic(err)
	}
	for _, d := range p.Displays {
		fmt.Printf("display %d: %q left=%v right=%v\n", d.Index, d.Text, d.LeftTally, d.RightTally)
	}
	// Output: display 1: "CAM 1" left=Red right=Off
}

func ExampleMarshalUDP() {
	p := tsl.Packet{Unicode: true}
	for i := range 200 {
		p.Displays = append(p.Displays, tsl.Display{Index: uint16(i), Text: fmt.Sprintf("Camera %d", i)})
	}

	datagrams, err := tsl.MarshalUDP(&p)
	if err != nil {
		panic(err)
	}
	// Send each datagram with conn.Write.
	fmt.Println(len(datagrams), "datagrams")
	// Output: 3 datagrams
}

func ExampleEncoder() {
	var conn bytes.Buffer // e.g. a net.Conn from net.Dial("tcp", addr)

	enc := tsl.NewEncoder(&conn)
	err := enc.Encode(&tsl.Packet{Displays: []tsl.Display{{Index: 0, RightTally: tsl.LampGreen, Text: "PGM"}}})
	if err != nil {
		panic(err)
	}

	dec := tsl.NewDecoder(&conn)
	var p tsl.Packet
	if err := dec.Decode(&p); err != nil {
		panic(err)
	}
	fmt.Println(p.Displays[0].Text, p.Displays[0].RightTally)
	// Output: PGM Green
}
