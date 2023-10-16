package types

import "fmt"

type Hash [32]byte

func (h Hash) Bytes() []byte {
	return h[:]
}

func (h Hash) String() string {
	return fmt.Sprintf("%x", h[:])
}

type Type int
