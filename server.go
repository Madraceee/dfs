package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/madraceee/dfs/p2p"
)

type FileServerOpts struct {
	EncKey            []byte
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitch chan struct{}
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}
	return &FileServer{
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
		FileServerOpts: opts,
	}
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(key) {
		log.Printf("[%s] serving file (%s) from local disk", s.Transport.Addr(), key)
		_, r, err := s.store.Read(key)
		return r, err
	}

	log.Printf("dont have file (%s) locally, fetching from network\n", key)

	msg := Message{
		Payload: MessageGetFile{
			Key: hashKey(key),
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(1 * time.Millisecond)

	for _, peer := range s.peers {
		// Read file size sent from the server before reading the file
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)
		n, err := s.store.WriteDecrypt(s.EncKey, key, io.LimitReader(peer, fileSize))
		if err != nil {
			peer.CloseStream()
			return nil, err
		}

		log.Printf("[%s] received bytes over the network: %d\n", s.Transport.Addr(), n)
		peer.CloseStream()
	}

	_, r, err := s.store.Read(key)
	return r, err
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := s.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  hashKey(key),
			Size: size + 16,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(1 * time.Millisecond)

	peers := []io.Writer{}

	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})
	n, err := copyEncrypt(s.EncKey, fileBuffer, mw)
	if err != nil {
		return err
	}
	log.Printf("[%s] received and written %v bytes to disk", s.Transport.Addr(), n)

	return nil

}

func (s *FileServer) loop() {
	defer func() {
		log.Println("file server stopped")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Printf("decode err: %s\n", err)
				continue
			}

			if err := s.handleMessage(rpc.From, &msg); err != nil {
				log.Printf("handle message err: %s", err)
				continue
			}

		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) OnPeer(peer p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[peer.RemoteAddr().String()] = peer

	log.Printf("connected with remote peer %s\n", peer.RemoteAddr())
	return nil
}

func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	}
	return nil
}

// handleMessageStoreFile gets the message. Once the message is received,
// it reads data from the peer before leaving. Hence file is store
func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}
	if _, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size)); err != nil {
		return nil
	}

	peer.CloseStream()

	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.Key) {
		return fmt.Errorf("[%s] requested file (%s), but does not exist on disk", s.Transport.Addr(), msg.Key)
	}
	fileSize, r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok {
		defer rc.Close()
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}

	// First send "IncomingStream" byte to peer
	// Then send file size
	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fileSize)
	time.Sleep(1 * time.Millisecond)
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	log.Printf("[%s] written %d bytes over the network to %s\n", s.Transport.Addr(), n, from)

	return nil
}

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			if err := s.Transport.Dial(addr); err != nil {
				log.Printf("[%s] dial error: %v", s.Transport.Addr(), err)
			}
		}(addr)
	}

	return nil
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.bootstrapNetwork()
	s.loop()

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
