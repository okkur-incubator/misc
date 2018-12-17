package main

import (
	"flag"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
)

func main() {
	var requests [][]byte

	killc := make(chan os.Signal, 1)
	signal.Notify(killc, os.Interrupt)

	port := flag.String("port", ":5000", "UDP listener port")
	endpoint := flag.String("endpoint", "", "Target address to forward to")
	drop := flag.Float64("drop", 5, "Requests drop rate (1-100)%")
	buffer := flag.Int("buffer", 10240, "max buffer size for the socket io")
	flag.Parse()

	if *drop < 1 || *drop > 100 {
		log.Fatalf("Wrong input for requests drop rate %d", *drop)
	}

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

	// Initialize kill signal listener
	go func() {
		for range killc {
			rate := int(math.Round(*drop / 100 * float64(len(requests))))
			requests = requests[0 : len(requests)-rate-1]
			for _, request := range requests {
				if _, err := targetConn.Write(request); err != nil {
					log.Println("Could not forward packet.")
				}
			}
			os.Exit(0)
		}
	}()

	for {
		b := make([]byte, *buffer)
		n, _, err := sourceConn.ReadFromUDP(b)

		if err != nil {
			log.Println("Could not receive a packet")
			continue
		}
		requests = append(requests, b[0:n])
	}

}
