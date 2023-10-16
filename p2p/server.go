package p2p

import (
	"bytes"
	"context"
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/deffusion/netflux/network/routing"
	"github.com/deffusion/netflux/p2p/node"
	"github.com/deffusion/netflux/types"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type Config struct {
	BootNode    types.PeerMeta
	PeerLimit   int
	TCPPort     uint
	HTTPPort    uint
	IP          string
	NATTCPPort  uint
	NATHTTPPort uint
}

type configuration struct {
	BootNodeIP   string `yaml:"BootNodeIP"`
	BootNodeHTTP uint   `yaml:"BootNodeHTTP"`
	TCP          uint   `yaml:"TCP"`
	HTTP         uint   `yaml:"HTTP"`
}

func confFromFile() (configuration, error) {
	f, err := os.Open("./config/config.yml")
	if err != nil {
		log.Println("err:", err)
		return configuration{}, nil
	}
	var buff bytes.Buffer
	buff.ReadFrom(f)
	var c configuration
	err = yaml.Unmarshal(buff.Bytes(), &c)
	return c, err
}

func LoadConf() (Config, error) {
	conf, err := confFromFile()
	fmt.Println("conf:", conf)
	if err != nil {
		return Config{}, err
	}
	return Config{
		BootNode: types.PeerMeta{
			IP:   conf.BootNodeIP,
			HTTP: conf.BootNodeHTTP,
		}, //
		PeerLimit:   15,
		TCPPort:     conf.TCP,
		HTTPPort:    conf.HTTP,
		NATTCPPort:  3721,
		NATHTTPPort: 3824,
	}, nil
}

var ErrTakeChClosed = errors.New("takeCh closed")
var ErrServerStopped = errors.New("server stopped")

type Server struct {
	Config
	peerStore    routing.PeerStore
	listener     net.Listener
	httpServer   *http.Server
	pendingMsgCh chan *types.Message // messages from the peers
	takeCh       chan *types.Message // messages to be taken
	stopCh       chan struct{}
	closed       bool
	knownSet     mapset.Set[types.Hash]
}

func NewServer(conf Config) *Server {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", conf.TCPPort))
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	s := &Server{
		Config:       conf,
		peerStore:    routing.NewGossipPeerStore(conf.PeerLimit),
		listener:     listener,
		pendingMsgCh: make(chan *types.Message),
		takeCh:       make(chan *types.Message),
		stopCh:       make(chan struct{}),
		closed:       false,
		knownSet:     mapset.NewSet[types.Hash](),
	}
	rg := r.Group("/", func(ctx *gin.Context) {
		ctx.Set("server", s)
	})
	Route(rg)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.HTTPPort),
		Handler: r,
	}

	return s
}

func (s *Server) Meta() types.PeerMeta {
	return types.PeerMeta{
		IP:   s.IP,
		TCP:  s.NATTCPPort,
		HTTP: s.NATHTTPPort,
	}
}

func (s *Server) handlePendingLoop() {
	for {
		select {
		case <-s.stopCh:
			return
		case msg := <-s.pendingMsgCh:
			if s.KnowHash(msg.Hash()) {
				log.Println("received redundant msg")
				continue
			}
			s.takeCh <- msg
		}
	}
}

func (s *Server) acceptConn() {
	for {
		if s.closed {
			break
		}
		conn, err := s.listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		peer := s.peerFromConn(conn)
		// one meta info packet need to be sent firstly by the remote
		packet, err := peer.UnwrapOnePacket()
		if err != nil {
			log.Println(err)
			peer.Stop()
			continue
		}
		var meta types.PeerMeta
		// TODO: type checking
		packet.Decode(&meta)
		log.Println("(Server.acceptConn) peer tcp port:", meta.TCP)
		peer.NATTCP = meta.TCP
		peer.NATHTTP = meta.HTTP
		s.peerStore.AddPeer(peer)
	}
}

func (s *Server) ConnectPeers(metas []types.PeerMeta) {
	for _, meta := range metas {
		log.Println("peer tcp port:", meta.TCP)
		if meta.ID() == s.Meta().ID() {
			log.Println("(Server.ConnectPeers) refuse to connect to it self")
			continue
		}
		if s.peerStore.Know(meta.ID()) {
			log.Println("(Server.ConnectPeers) refuse to connect to peer it knows")
			continue
		}
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", meta.IP, meta.TCP))
		if err != nil {
			log.Println(err)
			continue
		}
		// send meta info to the remote if conn was established
		p := types.NewPacket(types.PacketTypePeerMeta, s.Meta())
		b := p.BytesToSend()
		_, err = conn.Write(b)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("(Server.ConnectPeers) meta info sent to:%s\n", conn.RemoteAddr())
		peer := s.peerFromConn(conn)
		peer.NATTCP = meta.TCP
		peer.NATHTTP = meta.HTTP
		s.peerStore.AddPeer(peer)
	}
}

func (s *Server) peerFromConn(conn net.Conn) *node.Peer {
	rCh := make(chan *types.Packet)
	wCh := make(chan *types.Packet)
	peer := node.NewPeer(conn, rCh, wCh, s.pendingMsgCh)
	return peer
}

func (s *Server) Serve() {
	defer s.listener.Close()
	go s.acceptConn()
	go s.handlePendingLoop()
	log.Println("tcp listening port:", s.TCPPort)
	log.Println("http listening port:", s.HTTPPort)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("HTTP server stopped: ", err)
	}
	<-s.stopCh
}

func (s *Server) Stop() {
	if s.closed {
		return
	}
	s.closed = true
	s.peerStore.Close()
	close(s.stopCh)
	tCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(tCtx); err != nil {
		log.Fatal("HTTP server shutdown error: ", err)
	}
}

func (s *Server) KnowHash(hash types.Hash) bool {
	return s.knownSet.Contains(hash)
}

func (s *Server) MarkHash(hash types.Hash) {
	s.knownSet.Add(hash)
}

// Feed broadcasts messages into the network
func (s *Server) Feed(msg *types.Message) (fed bool) {
	if s.KnowHash(msg.Hash()) {
		log.Println("refuse to broadcast message already received")
		return false
	}
	s.MarkHash(msg.Hash())
	s.peerStore.Broadcast(msg)
	return true
}

func (s *Server) Take() (*types.Message, error) {
	select {
	case msg, ok := <-s.takeCh:
		if !ok {
			return nil, ErrTakeChClosed
		}
		return msg, nil
	case <-s.stopCh:
		return nil, ErrServerStopped
	}
}
