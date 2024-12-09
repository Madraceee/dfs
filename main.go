package main

import (
	"log"
	_ "time"

	"github.com/madraceee/dfs/p2p"
)

func makeServer(listenAddr, root string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// TODO onPeer
	}

	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)
	fileServerOpts := FileServerOpts{
		ListenAddr:        listenAddr,
		StorageRoot:       root,
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}
	s := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer

	return s
}

func main() {
	s1 := makeServer(":3000", "3000_network", "")
	s2 := makeServer(":4000", "4000_network", ":3000")

	go func() {
		log.Fatal(s1.Start())
	}()

	s2.Start()
}
