package core

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/songgao/water"
	"nhooyr.io/websocket"
)

type Client struct {
	addr string
}

func NewClient(addr string) *Client {
	return &Client{addr: addr}
}

func (c *Client) Run() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	iface, err := water.New(Config)
	if err != nil {
		log.Panicf("error on water new: %s", err.Error())
	}

	log.Printf("iface: %s", iface.Name())

	ws, _, err := websocket.Dial(ctx, c.addr, &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		log.Panicf("error on websocket.Dial: %s", err.Error())
	}

	handleIPAssign(ctx, ws, iface)

	broadcastDomain := NewBroadcastDomain()
	broadcastDomain.Join(iface, true)
	id := broadcastDomain.Join(wsWrapper{ws}, false)

	waitSignal(syscall.SIGINT, syscall.SIGTERM)
	broadcastDomain.Leave(id)
}

func handleIPAssign(ctx context.Context, ws *websocket.Conn, iface *water.Interface) {
	_, ipMsg, err := ws.Read(ctx)
	if err != nil {
		log.Panicf("error on ws.Read: %s", err.Error())
	}
	if ipMsg[0] != PacketTypeIPAssign {
		log.Panicf("PacketTypeIPAssign type error: % x", ipMsg)
	}
	var ipBody IPAssignBody
	err = json.Unmarshal(ipMsg[1:], &ipBody)
	if err != nil {
		log.Panicf("Packet type error: % x", ipMsg)
	}
	log.Printf("handleIPAssign: %s", string(ipMsg[1:]))

	setupTapAddr(iface.Name(), &ipBody)
}

func waitSignal(sig ...os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig...)
	<-c
}
