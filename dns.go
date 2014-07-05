package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
)

type DNSServer struct {
	Addr *net.UDPAddr
	Conn *net.UDPConn
}

var (
	srv      *net.UDPConn
	upstream *DNSServer
)

func main() {
	var err error

	upstream, err = newDNSServer("192.168.0.3", 53)
	if err != nil {
		log.Fatalln("Unable add upstream:", err)
	}

	go runServerProxy("192.168.0.11", 5354)

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

forever:
	for {
		select {
		case <-sig:
			log.Println("Signal received, stopping")
			break forever
		}
	}
}

func newDNSServer(addr string, port int) (ds *DNSServer, err error) {
	ds = &DNSServer{
		Addr: &net.UDPAddr{
			IP:   net.ParseIP(addr),
			Port: port,
		},
	}
	ds.Conn, err = net.DialUDP("udp", nil, ds.Addr)

	return
}

func runServerProxy(addr string, port int) {
	var err error

	fulladdr := fmt.Sprintf("%s:%d", addr, port)
	udpaddr := &net.UDPAddr{
		IP:   net.ParseIP(addr),
		Port: port,
	}

	srv, err = net.ListenUDP("udp", udpaddr)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	log.Println("DNS: Started server at", fulladdr)
	buf := make([]byte, 65536)
	oobuf := make([]byte, 512)
	for {
		n, _, _, addr, err := srv.ReadMsgUDP(buf, oobuf)
		if err != nil {
			log.Println("DNS ERROR:", err)
			continue
		}

		go handleDNS(buf[:n], addr)
	}

	panic("not reachable")
}

func handleDNS(payload []byte, addr *net.UDPAddr) {
	log.Println("DNS: Query from", addr)

	_, err := upstream.Conn.Write(payload)
	if err != nil {
		log.Println("DNS ERROR:", err)
		return
	}

	buf := make([]byte, 65536)
	oobuf := make([]byte, 512)
	n, _, _, _, err := upstream.Conn.ReadMsgUDP(buf, oobuf)
	if err != nil {
		log.Println("DNS ERROR:", err)
		return
	}
	buf = buf[:n]

	_, err = srv.WriteTo(buf, addr)
	if err != nil {
		log.Println("DNS ERROR:", err)
	}

	return
}

// vim: ts=4 sw=4 sts=4
