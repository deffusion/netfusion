package p2p

import (
	"fmt"
	"github.com/deffusion/netflux/p2p/httpclient"
	"github.com/deffusion/netflux/types"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func Route(r *gin.RouterGroup) {
	Test(r)
	Peer(r)
	Message(r)
	Meta(r)
	Control(r)
}

func Test(r *gin.RouterGroup) {
	path := "/test"
	r.GET(path+"/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})
}

func Peer(r *gin.RouterGroup) {
	path := "/peer"
	r.POST(path, func(ctx *gin.Context) {
		server := ctx.Value("server").(*Server)
		var metas []types.PeerMeta
		err := ctx.ShouldBindJSON(&metas)
		if err != nil {
			log.Println("bind json err:", err)
		}
		log.Println("metas:", metas)
		server.ConnectPeers(metas)
		ctx.String(http.StatusOK, "ok")
	})
}

func Message(r *gin.RouterGroup) {
	path := "/message"
	r.POST(path, func(ctx *gin.Context) {
		server := ctx.Value("server").(*Server)
		var testMsg httpclient.TestMsgType
		ctx.BindJSON(&testMsg)
		log.Println("init broadcast:", testMsg)
		msg := types.NewMessage(0, testMsg)
		fed := server.Feed(msg)
		if fed {
			err := httpclient.Report(fmt.Sprintf(":%d", server.Meta().TCP), server.BootNode, testMsg)
			if err != nil {
				log.Println(err)
				ctx.String(http.StatusInternalServerError, "boot server report failed")
				return
			}
		}
		ctx.String(http.StatusOK, "ok")
	})
}

func Meta(r *gin.RouterGroup) {
	path := "/meta"
	r.POST(path, func(ctx *gin.Context) {
		server := ctx.Value("server").(*Server)
		var meta types.PeerMeta
		err := ctx.BindJSON(&meta)
		if err != nil {
			log.Println("meta json err:", err)
		}
		log.Println("/meta:", meta)
		server.NATTCPPort = meta.TCP
		server.NATHTTPPort = meta.HTTP
		server.IP = meta.IP
		ctx.String(http.StatusOK, server.Meta().ID())
	})
}

func Control(r *gin.RouterGroup) {
	path := "/control"
	r.POST(path, func(ctx *gin.Context) {
		server := ctx.Value("server").(*Server)
		ctx.String(http.StatusOK, "ok")
		server.Stop()
	})
}
