package p2p

// Peer is an interface that represents the remote node
type Peer interface {
}

// Transport handles communication between nodes
// Can be TCP, UDP, WebSockets
type Transport interface {
	ListenAndAccept() error
}
