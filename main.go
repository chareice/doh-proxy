package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/miekg/dns"
)

var upstreamServer string

func main() {
	listenPort := flag.String("port", "53", " Server Listen Port")
	listenHost := flag.String("host", "127.0.0.1", " Server Listen Host")
	upstream := flag.String("upstream", "https://cloudflare-dns.com/dns-query", "Upstream Server")
	flag.Parse()

	upstreamServer = *upstream

	listenAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", *listenHost, *listenPort))
	if err != nil {
		log.Panic(err)
	}

	conn, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		log.Panic(err)
		return
	}

	defer conn.Close()

	log.Printf("DNS Server start %v\n", listenAddr)

	for {
		buf := make([]byte, 512)
		_, addr, _ := conn.ReadFromUDP(buf)

		go serveDNSQuery(conn, addr, buf)
	}
}

func serveDNSQuery(conn *net.UDPConn, addr *net.UDPAddr, buf []byte) {
	msg := new(dns.Msg)

	if err := msg.Unpack(buf); err != nil {
		fmt.Println(err)
		return
	}

	packedMessage, err := msg.Pack()
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Printf("Questions: %v\n", msg.Question)

	requestURL := fmt.Sprintf("%s?dns=%s", upstreamServer, base64.URLEncoding.EncodeToString(packedMessage))

	resp, err := http.Get(requestURL)

	if err != nil {
		log.Printf("DOH Response Error %v", err)
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("DOH Response Read Error %v", err)
		return
	}

	conn.WriteTo(body, addr)
}
