package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/madraceee/dfs/p2p"
)

func makeServer(listenAddr, root string, encKey []byte, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// TODO onPeer
	}

	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)
	fileServerOpts := FileServerOpts{
		EncKey:            encKey,
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
	encKey := newEncryptionKey()
	s1 := makeServer(":3000", "3000_network", encKey, "")
	s2 := makeServer(":4000", "4000_network", encKey, ":3000")
	s3 := makeServer(":5000", "5000_network", encKey, ":3000", ":4000")

	go func() {
		log.Fatal(s1.Start())
	}()

	go func() {
		log.Fatal(s2.Start())
	}()

	time.Sleep(5 * time.Second)
	go func() {
		s3.Start()
	}()

	time.Sleep(2 * time.Second)
	for i := 0; i < 10; i++ {
		data := bytes.NewReader([]byte("big data file"))
		s2.StoreData(fmt.Sprintf("test-%d", i), io.Reader(data))
		time.Sleep(5 * time.Millisecond)
	}

	cmd := exec.Command("rm", "-rf", "./3000_network")
	cmd.Run()

	for i := 0; i < 10; i += 2 {
		r, err := s1.Get(fmt.Sprintf("test-%d", i))
		if err != nil {
			log.Fatal(err)
		}

		b, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(b))
	}
	select {}
}
