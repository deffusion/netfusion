package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"log"
)

const (
	PacketTypeMessage Type = iota
	PacketTypePeerMeta
)

type Packet struct {
	T    Type
	Data []byte
}

func NewPacket(t Type, data any) *Packet {
	encoded := gobEncode(data)
	return &Packet{
		t,
		encoded.Bytes(),
	}
}

func PacketFromBytes(b []byte) *Packet {
	var p Packet
	buff := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buff)
	dec.Decode(&p)
	return &p
}

func (p *Packet) BytesToSend() []byte {
	payloadBuff := gobEncode(p)
	lenBuff := bytes.Buffer{}
	binary.Write(&lenBuff, binary.LittleEndian, int32(payloadBuff.Len()))
	return append(lenBuff.Bytes(), payloadBuff.Bytes()...)
}

func (p *Packet) Decode(m any) {
	gobDecode(p.Data, m)
}

func gobEncode(m any) bytes.Buffer {
	payloadBuff := bytes.Buffer{}
	enc := gob.NewEncoder(&payloadBuff)
	err := enc.Encode(m)
	if err != nil {
		log.Println("gob encode err:", err)
	}
	return payloadBuff
}

func gobDecode(b []byte, m any) {
	buff := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buff)
	err := dec.Decode(m)
	if err != nil {
		log.Println("gob decode err:", err)
	}
}

type Message struct {
	Tag int // to differentiate different type of messages
	Raw []byte
	//Timestamp int64
}

func NewMessage(tag int, data any) *Message {
	rawBuff := gobEncode(data)
	return &Message{
		tag,
		rawBuff.Bytes(),
	}
}

func (m *Message) Decode(v any) {
	gobDecode(m.Raw, v)
}

func (m *Message) Hash() Hash {
	encoded := gobEncode(m)
	return sha256.Sum256(encoded.Bytes())
}

type PeerMeta struct {
	IP   string
	TCP  uint
	HTTP uint
}

func (m PeerMeta) ID() string {
	return fmt.Sprintf("%s:%d", m.IP, m.TCP)
}
