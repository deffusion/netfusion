// This is a tiny receiving and relaying example

package main

import (
	"fmt"
	"github.com/deffusion/netflux/p2p"
	"github.com/deffusion/netflux/p2p/httpclient"
	"log"
)

func main() {
	conf, err := p2p.LoadConf()
	if err != nil {
		log.Println(err)
		return
	}
	s := p2p.NewServer(conf)
	go takeThenFeed(s)
	s.Serve()
}

func takeThenFeed(s *p2p.Server) {
	for {
		msg, err := s.Take()
		if err != nil {
			break
		}
		if msg.Tag == 0 {
			var testMsg httpclient.TestMsgType
			msg.Decode(&testMsg)
			log.Println("received message:", testMsg)
			fed := s.Feed(msg)
			if fed {
				err := httpclient.Report(fmt.Sprintf(":%d", s.Meta().TCP), s.BootNode, testMsg)
				if err != nil {
					log.Println(err)
					break
				}
			}
		}
	}
}
