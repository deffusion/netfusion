package p2p

import (
	"testing"
)

func TestConf(t *testing.T) {
	conf, err := confFromFile()
	if err != nil {
		t.Error(err)
	}
	t.Log("conf:", conf)
}
