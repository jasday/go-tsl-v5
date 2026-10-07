# go-tsl-v5

Go implementation of the [TSL UMD v5.0 protocol](https://tslproducts.com/media/1959/tsl-umd-protocol.pdf) for tally and under-monitor displays.

```sh
go get github.com/jasday/go-tsl-v5
```

```go
import tsl "github.com/jasday/go-tsl-v5"
```

## Packets

A `Packet` is addressed to a screen and holds one or more `Display` messages.

```go
p := tsl.Packet{
	Screen: 0,
	Displays: []tsl.Display{
		{Index: 1, LeftTally: tsl.LampRed, Brightness: 3, Text: "CAM 1"},
	},
}
```

- Set `Packet.Unicode` to send text as UTF-16LE. Otherwise text must be ASCII.
- `tsl.BroadcastIndex` (`0xFFFF`) addresses all screens or all displays.
- Lamps are `LampOff`, `LampRed`, `LampGreen` and `LampAmber`. Brightness is 0–3.

## UDP

Each datagram carries one packet, of at most 2048 bytes.

```go
datagrams, err := tsl.MarshalUDP(&p) // splits displays across packets if needed
for _, b := range datagrams {
	conn.Write(b)
}

var p tsl.Packet
err := tsl.Unmarshal(buf[:n], &p)
```

## TCP and serial

Byte streams use DLE/STX framing.

```go
err := tsl.NewEncoder(conn).Encode(&p)

dec := tsl.NewDecoder(conn)
for {
	var p tsl.Packet
	if err := dec.Decode(&p); err != nil {
		// ErrMalformed/ErrShortPacket: bad frame, keep reading.
		// io.EOF: stream closed.
	}
}
```

## Errors

| Error | Meaning |
|---|---|
| `ErrShortPacket` | Buffer is shorter than the packet's byte count |
| `ErrMalformed` | Packet contents are inconsistent |
| `ErrInvalidValue` | `Packet` cannot be encoded as given |
| `ErrPacketTooLarge` | Packet, text or UDP display exceeds its size limit |

## Commands

```sh
go run ./cmd/tsl-listen -proto udp -addr :5900
go run ./cmd/tsl-send -proto udp -addr 127.0.0.1:5900 -index 1 -text "CAM 1" -lh red
```

Use `-proto tcp` for TCP.

## Not supported

Screen control (`FLAGS` bit 1) and display control data (`CONTROL` bit 15) are undefined in v5.0. They're exposed as `Packet.ScreenControl` and `Display.ControlData`, with no payload.
