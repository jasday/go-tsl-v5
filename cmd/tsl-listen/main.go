// Command tsl-listen prints TSL v5 packets received over UDP or TCP.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	tsl "github.com/jasday/go-tsl-v5"
)

func main() {
	proto := flag.String("proto", "udp", "transport: udp or tcp")
	addr := flag.String("addr", ":5900", "address to listen on")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, *proto, *addr, printPacket); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, proto, addr string, handle Handler) error {
	switch proto {
	case "udp":
		conn, err := net.ListenPacket("udp", addr)
		if err != nil {
			return err
		}
		log.Printf("listening on udp %v", conn.LocalAddr())
		return ServeUDP(ctx, conn, handle)
	case "tcp":
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		log.Printf("listening on tcp %v", ln.Addr())
		return ServeTCP(ctx, ln, handle)
	default:
		return fmt.Errorf("unknown protocol %q", proto)
	}
}

func printPacket(p tsl.Packet, from net.Addr) {
	log.Printf("%v: screen=%d version=%d unicode=%t screen-control=%t", from, p.Screen, p.Version, p.Unicode, p.ScreenControl)
	for _, d := range p.Displays {
		if d.ControlData {
			log.Printf("  display %d: control data", d.Index)
			continue
		}
		log.Printf("  display %d: %q lh=%v text=%v rh=%v brightness=%d", d.Index, d.Text, d.LeftTally, d.TextTally, d.RightTally, d.Brightness)
	}
}
