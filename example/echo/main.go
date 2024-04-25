package main

import (
	tls2 "crypto/tls"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	_ "net/http/pprof"
	"time"

	"github.com/leslie-fei/gnettls"
	"github.com/leslie-fei/gnettls/tls"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

func main() {
	var port int
	var multicore bool

	flag.IntVar(&port, "port", 443, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore with multiple CPU cores")
	flag.Parse()

	addr := fmt.Sprintf("tcp://:%d", port)

	time.AfterFunc(time.Second, func() {
		conn, err := tls2.Dial("tcp", fmt.Sprintf(":%d", port), &tls2.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		for {
			data := []byte("HelloWorld")
			// encode data
			head := make([]byte, 4)
			packetLen := uint32(len(data) + 4)
			binary.LittleEndian.PutUint32(head, packetLen)
			packet := append(head, data...)
			_, err = conn.Write(packet)
			if err != nil {
				log.Fatal(err)
			}
			// read server response
			rsp := make([]byte, packetLen)
			_, err = io.ReadFull(conn, rsp)
			if err != nil {
				log.Fatal(err)
			}
			// server response data
			serverData := rsp[4:]
			logging.Infof("read from server: %s", serverData)

			time.Sleep(time.Second)
		}
	})

	runEchoServer(addr, multicore)
}

type echoServer struct {
	*gnet.BuiltinEventEngine
	clients   uint32
	addr      string
	multicore bool
}

func (s *echoServer) OnTraffic(c gnet.Conn) (action gnet.Action) {
	// decode data
	// packet 4bytes packet length + body length
	// if packet data not enough next round to read
	if c.InboundBuffered() < 4 {
		return
	}

	packetLenBytes, err := c.Peek(4)
	if err != nil {
		logging.Errorf("peek packet length failed err: %v", err)
		return gnet.Close
	}

	packetLen := binary.LittleEndian.Uint32(packetLenBytes)
	// if packet data not enough next round to read
	if c.InboundBuffered() < int(packetLen) {
		return
	}

	// have enough data read and decode it
	// discard 4 bytes for head
	_, _ = io.CopyN(io.Discard, c, 4)
	bodyLen := packetLen - 4
	var body = make([]byte, bodyLen)
	_, _ = c.Read(body)

	logging.Infof("server OnTraffic data: %s", body)
	// write back to client
	// encode packet data
	_, _ = c.Writev([][]byte{packetLenBytes, body})

	return
}

func runEchoServer(addr string, multicore bool) {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{mustLoadCertificate()},
	}

	hs := &echoServer{
		addr:      addr,
		multicore: multicore,
	}
	options := []gnet.Option{
		gnet.WithMulticore(multicore),
		gnet.WithTCPKeepAlive(time.Minute * 5),
		gnet.WithReusePort(true),
	}

	log.Fatal(gnettls.Run(hs, hs.addr, tlsConfig, options...))
}

func mustLoadCertificate() tls.Certificate {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load server certificate: %v", err)
	}
	return cert
}
