package p2p

import "net"

// Peer is an interface that represents the remote node
type Peer interface {
	net.Conn
	Send([]byte) error
	CloseStream()
}

// Transport handles communication between nodes
// Can be TCP, UDP, WebSockets
type Transport interface {
	Addr() string
	ListenAndAccept() error
	Dial(string) error
	Consume() <-chan RPC
	Close() error
}
