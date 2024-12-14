package p2p

// Message holds any arbitrary data that is sent
// over each transport between two nodes
type RPC struct {
	From    string
	Payload []byte
}
