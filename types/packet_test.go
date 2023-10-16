package types

import (
	"fmt"
	"testing"
)

type TestMsg struct {
	Num int
	Str string
}

func TestMessageEncDec(t *testing.T) {
	var payload = TestMsg{
		1,
		"hello",
	}
	msg := NewMessage(0, payload)
	// ...
	if msg.Tag == 0 {
		var decPayload TestMsg
		msg.Decode(&decPayload)
		fmt.Println(decPayload)
	}
}
