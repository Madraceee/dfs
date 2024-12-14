package main

import (
	_ "bytes"
	"fmt"
	"io"
	"log"
	"time"
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

	go func() {
		s2.Start()
	}()

	time.Sleep(2 * time.Second)

	// s2.StoreData("test", io.Reader(bytes.NewBuffer([]byte("123456"))))
	//
	// time.Sleep(5 * time.Second)

	r, err := s2.Get("test")
	if err != nil {
		log.Fatal(err)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))
}
