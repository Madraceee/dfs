package p2p

// Peer is an interface that represents the remote node
type Peer interface {
	Close() error
}

// Transport handles communication between nodes
// Can be TCP, UDP, WebSockets
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
