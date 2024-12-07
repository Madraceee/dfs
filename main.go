package main

import (
	"fmt"
	"log"

	"github.com/madraceee/dfs/p2p"
)

func OnPeer(p p2p.Peer) error {
	p.Close()
	fmt.Println("doing some logic NON-TCP login")
	return nil
}

func main() {
	opts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}

	tr := p2p.NewTCPTransport(opts)

	go func() {
		for {
			msg := <-tr.Consume()
			fmt.Printf("%v\n", msg)
		}
	}()

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}
