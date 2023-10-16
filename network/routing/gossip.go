package routing

import (
	"github.com/deffusion/netflux/p2p/node"
	"github.com/deffusion/netflux/types"
	"log"
)

type GossipPS struct {
	ps     map[string]*node.Peer
	max    int
	closed bool
}

func NewGossipPeerStore(max int) PeerStore {
	return &GossipPS{
		ps:     make(map[string]*node.Peer),
		max:    max,
		closed: false,
	}
}

func (g *GossipPS) PeersToBroadcast(hash types.Hash) []*node.Peer {
	ps := make([]*node.Peer, 0, len(g.ps))
	for _, peer := range g.ps {
		if !peer.KnowPacketHash(hash) {
			ps = append(ps, peer)
		}
	}
	return ps
}

func (g *GossipPS) Know(key string) bool {
	_, ok := g.ps[key]
	return ok
}

func (g *GossipPS) Broadcast(msg *types.Message) {
	ps := g.PeersToBroadcast(msg.Hash())
	packet := types.NewPacket(types.PacketTypeMessage, *msg)

	for _, p := range ps {
		p.Send(packet)
	}
}

func (g *GossipPS) AddPeer(peer *node.Peer) {
	if g.Know(peer.ID()) {
		log.Println("already connected to:", peer.NATTCP)
		return
	}
	g.ps[peer.ID()] = peer
	peer.Serv()
}

func (g *GossipPS) RemovePeer(peer *node.Peer) {
	delete(g.ps, peer.ID())
}

func (g *GossipPS) Close() {
	if g.closed {
		return
	}
	g.closed = true
	for _, peer := range g.ps {
		peer.Stop()
	}
}
