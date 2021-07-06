package core

import (
	"context"
	"encoding/json"
	"log"
	"runtime"
	"time"

	"github.com/songgao/water"
	"nhooyr.io/websocket"
)

type Client struct {
	addr            string
	iface           *water.Interface
	ws              *websocket.Conn
	broadcastDomain *BroadcastDomain
	tapID           int
	wsID            int
}

func NewClient(addr string) *Client {
	cli := &Client{addr: addr}
	runtime.SetFinalizer(cli, (*Client).Close)
	return cli
}

func (c *Client) Run() {
	var err error

	c.iface, err = water.New(Config)
	if err != nil {
		log.Panicf("error on water new: %s", err.Error())
	}

	log.Printf("iface: %s", c.iface.Name())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c.ws, _, err = websocket.Dial(ctx, c.addr, &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		log.Panicf("error on websocket.Dial: %s", err.Error())
	}

	c.handleIPAssign()

	c.broadcastDomain = NewBroadcastDomain()
	c.tapID = c.broadcastDomain.Join(c.iface, true)
	c.wsID = c.broadcastDomain.Join(wsWrapper{c.ws}, false)
}

func (c *Client) Close() {
	runtime.SetFinalizer(c, nil)
	if c.broadcastDomain != nil {
		c.broadcastDomain.Leave(c.wsID)
		c.broadcastDomain.Leave(c.tapID)
	}
}

func (c *Client) handleIPAssign() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, ipMsg, err := c.ws.Read(ctx)
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

	setupTapAddr(c.iface.Name(), &ipBody)
}
