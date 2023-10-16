package httpclient

import (
	"fmt"
	"github.com/deffusion/netflux/types"
	"log"
)

type TestMsgType struct {
	Num int
	Str string
}

func Report(metaID string, bootNode types.PeerMeta, testMsg TestMsgType) error {
	url := fmt.Sprintf("http://%s:%d/report", bootNode.IP, bootNode.HTTP)
	log.Println("request:", url)
	resp, err := Post(url, struct {
		Node string
		Info string
	}{
		Node: metaID,
		Info: fmt.Sprintf("%d, %s", testMsg.Num, testMsg.Str),
	})
	if err != nil {
		return err
	}
	resp.Close()
	return nil
}
