package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

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
		EncKey:            newEncryptionKey(),
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
	for i := 0; i < 2; i++ {
		data := bytes.NewReader([]byte("big data file"))
		s2.StoreData(fmt.Sprintf("test-%d", i), io.Reader(data))
		time.Sleep(5 * time.Millisecond)
	}

	// r, err := s1.Get("test-1")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// b, err := io.ReadAll(r)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// fmt.Println(string(b))
	// select {}
}
