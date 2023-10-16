package node

import (
	"bytes"
	"encoding/binary"
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/deffusion/netflux/types"
	"github.com/pkg/errors"
	"io"
	"log"
	"net"
)

type Peer struct {
	conn        net.Conn
	IP          string
	NATTCP      uint
	NATHTTP     uint
	readMsgCh   chan *types.Packet
	writeMsgCh  chan *types.Packet
	pendingMsgs chan *types.Message
	stopCh      chan struct{}
	closed      bool
	knownSet    mapset.Set[types.Hash]
}

var ErrConnClosed = errors.New("err connection closed")

func NewPeer(conn net.Conn, rCh, wCh chan *types.Packet, pendingMsgs chan *types.Message) *Peer {
	fmt.Println("new peer:", conn.RemoteAddr())
	remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
	return &Peer{
		conn:        conn,
		IP:          remoteAddr.IP.String(),
		readMsgCh:   rCh,
		writeMsgCh:  wCh,
		pendingMsgs: pendingMsgs,
		stopCh:      make(chan struct{}),
		knownSet:    mapset.NewSet[types.Hash](),
		closed:      false,
	}
}

func (p *Peer) ID() string {
	return fmt.Sprintf("%s:%d", p.IP, p.NATTCP)
}

func (p *Peer) Send(packet *types.Packet) {
	p.writeMsgCh <- packet
}

func (p *Peer) sendLoop() {
	for {
		select {
		case <-p.stopCh:
			return
		case msg, ok := <-p.writeMsgCh:
			if !ok {
				return
			}
			b := msg.BytesToSend()
			n, err := p.conn.Write(b)
			if err != nil {
				log.Printf("sent %d/%d\n", n, len(b))
				log.Println(err)
				p.Stop()
			}
		}
	}
}

func readFull(conn net.Conn, buf []byte) error {
	totalRead := 0
	for totalRead < len(buf) {
		n, err := conn.Read(buf[totalRead:])
		if err != nil {
			return err
		}
		totalRead += n
	}
	return nil
}

func (p *Peer) UnwrapOnePacket() (*types.Packet, error) {
	lenBuff := make([]byte, 4)
	if err := readFull(p.conn, lenBuff); err != nil {
		if err == io.EOF {
			log.Println("(Peer.UnwrapOnePacket) Connection closed by remote")
			return nil, ErrConnClosed
		}
		return nil, errors.WithMessage(err, "Failed to read message")
	}
	buf := bytes.NewBuffer(lenBuff)
	var length int32
	binary.Read(buf, binary.LittleEndian, &length)

	buff := make([]byte, length)
	if err := readFull(p.conn, buff); err != nil {
		if err == io.EOF {
			log.Println("(Peer.UnwrapOnePacket) Connection closed by remote")
			return nil, ErrConnClosed
		}
		return nil, errors.WithMessage(err, "(Peer.UnwrapOnePacket) Failed to read message")
	}
	return types.PacketFromBytes(buff), nil
}

func (p *Peer) receiveLoop() {
	for {
		packet, err := p.UnwrapOnePacket()
		if err != nil {
			fmt.Println("(Peer.receiveLoop) unwrap packet err:", err)
			p.Stop()
			break
		}
		var msg types.Message
		packet.Decode(&msg)
		txHash := msg.Hash()
		if p.KnowPacketHash(txHash) {
			log.Println("(Peer.receiveLoop) ignored resent transaction:", msg)
			continue
		}
		p.markPacketHash(txHash)
		select {
		case <-p.stopCh:
			return
		case p.pendingMsgs <- &msg:
		}
	}
}

func (p *Peer) Serv() {
	go p.receiveLoop()
	go p.sendLoop()
}

func (p *Peer) Stop() {
	if p.closed {
		return
	}
	p.conn.Close()
	p.closed = true
	close(p.stopCh)
}

func (p *Peer) KnowPacketHash(h types.Hash) bool {
	return p.knownSet.Contains(h)
}

func (p *Peer) markPacketHash(h types.Hash) {
	p.knownSet.Add(h)
}
