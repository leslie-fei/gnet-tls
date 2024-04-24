package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/leslie-fei/gnettls"
	"github.com/leslie-fei/gnettls/tls"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

func main() {
	logging.Infof("version: 0.0.1")
	go func() {
		err := http.ListenAndServe("0.0.0.0:6060", nil)
		if nil != err {
			log.Fatal(err)
		}
	}()
	runHTTPServer()
}

func runHTTPServer() {
	var port int
	var multicore bool

	flag.IntVar(&port, "port", 443, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore with multiple CPU cores")
	flag.Parse()

	addr := fmt.Sprintf("tcp://:%d", port)
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{mustLoadCertificate()},
	}
	hs := &httpsServer{
		addr:      addr,
		multicore: multicore,
		pool:      goroutine.Default(),
	}

	options := []gnet.Option{
		gnet.WithMulticore(multicore),
		gnet.WithTCPKeepAlive(time.Minute * 5),
		gnet.WithReusePort(true),
	}

	log.Fatal(gnettls.Run(hs, hs.addr, tlsConfig, options...))
}

type httpsServer struct {
	gnet.BuiltinEventEngine

	addr      string
	multicore bool
	eng       gnet.Engine
	pool      *goroutine.Pool
}

/*func (hs *httpsServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	// logging.Infof("OnOpen addr: %s", c.RemoteAddr().String())
	return []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 12\r\n\r\nHello world!"), gnet.None
}*/

func (hs *httpsServer) OnTraffic(c gnet.Conn) (action gnet.Action) {
	// read all get http request
	// TODO decode http codec
	// TODO handling http request and response content, should decode http request for yourself
	// Must read the complete HTTP packet before responding.
	if hs.isHTTPRequestComplete(c) {
		_, _ = c.Next(-1)
		// for example hello response
		_, _ = c.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 12\r\n\r\nHello world!"))
	}
	return
}

func (hs *httpsServer) isHTTPRequestComplete(c gnet.Conn) bool {
	buf, _ := c.Peek(c.InboundBuffered())
	return bytes.Contains(buf, []byte("\r\n\r\n"))
}

func (hs *httpsServer) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	log.Printf("Closed connection on %s, error: %v", c.RemoteAddr().String(), err)
	return
}

func mustLoadCertificate() tls.Certificate {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load server certificate: %v", err)
	}
	return cert
}
