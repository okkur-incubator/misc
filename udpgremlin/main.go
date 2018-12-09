package main

import (
	"flag"
	"log"
	"net"
)

func main() {
	port := flag.String("port", ":5000", "UDP listener port")
	endpoint := flag.String("endpoint", "", "Target address to forward to")
	buffer := flag.Int("endpoint", 10240, "max buffer size for the socket io")
	flag.Parse()

	// Initialize UDP addresses
	sourceAddr, err := net.ResolveUDPAddr("udp", *port)
	if err != nil {
		log.Fatalf("Could not resolve address: %s", err.Error())
	}
	targetAddr, err := net.ResolveUDPAddr("udp", *endpoint)
	if err != nil {
		log.Fatalf("Could not resolve target address %s: %s", *endpoint, err.Error())
	}

	// Start UDP listener on the given port
	sourceConn, err := net.ListenUDP("udp", sourceAddr)
	if err != nil {
		log.Fatalf("Could not listen on address: %s", *port)
	}
	defer sourceConn.Close()

	// Initialize UDP dialer
	targetConn, err := net.DialUDP("udp", nil, targetAddr)
	if err != nil {
		log.Fatalf("Could not connect to target address: %s", targetAddr.String())
		return
	}
	defer targetConn.Close()

	for {
		b := make([]byte, *buffer)
		n, _, err := sourceConn.ReadFromUDP(b)

		if err != nil {
			log.Println("Could not receive a packet")
			continue
		}

		if _, err := targetConn.Write(b[0:n]); err != nil {
			log.Println("Could not forward packet.")
		}
	}

}
