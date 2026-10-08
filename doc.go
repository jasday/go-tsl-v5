// Package tsl implements the TSL UMD v5.0 tally protocol.
//
// Import it with an explicit name, as the module path does not match the package name:
//
//	import tsl "github.com/jasday/go-tsl-v5"
//
// A Packet holds a header and the display messages (Display) addressed to one screen.
//
// Over UDP, each datagram carries exactly one packet. Use MarshalUDP to encode,
// which splits packets that would exceed MaxUDPPacketSize, and Unmarshal to decode.
//
// Over TCP, serial or any other byte stream, packets are wrapped in DLE/STX
// framing. Use Encoder and Decoder.
//
// Screen control data (FLAGS bit 1) and display control data (CONTROL bit 15)
// are not defined by v5.0. They are exposed as Packet.ScreenControl and
// Display.ControlData and carry no payload.
package tsl
