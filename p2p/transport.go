package p2p

import "net"

// Peer is an interface that represents the remote node
type Peer interface {
	net.Conn
	Send([]byte) error
}

// Transport handles communication between nodes
// Can be TCP, UDP, WebSockets
type Transport interface {
	ListenAndAccept() error
	Dial(string) error
	Consume() <-chan RPC
	Close() error
}
