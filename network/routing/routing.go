package routing

import (
	"github.com/deffusion/netflux/p2p/node"
	"github.com/deffusion/netflux/types"
)

type PeerStore interface {
	PeersToBroadcast(hash types.Hash) []*node.Peer
	Know(key string) bool
	AddPeer(peer *node.Peer)
	RemovePeer(peer *node.Peer)
	Broadcast(tx *types.Message)
	Close()
}
