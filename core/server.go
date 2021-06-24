package core

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"

	"github.com/songgao/water"
	"nhooyr.io/websocket"
)

type Server struct {
	addr        string
	ipStart     string
	ipEnd       string
	iFaceBridge string
}

func NewServer(addr string, ipStart string, ipEnd string, iFaceBridge string) *Server {
	return &Server{addr: addr, ipStart: ipStart, ipEnd: ipEnd, iFaceBridge: iFaceBridge}
}

func (s *Server) Run() {
	startIP := net.ParseIP(s.ipStart)
	endIP := net.ParseIP(s.ipEnd)
	ipPool, err := NewIPPool(startIP, endIP)
	if err != nil {
		log.Panicf("error on NewIPPool: %s", err.Error())
	}

	iface, err := water.New(Config)
	if err != nil {
		log.Panicf("error on water new: %s", err.Error())
	}

	log.Printf("iface: %s", iface.Name())

	createBridge(s.iFaceBridge, iface.Name())

	broadcastDomain := NewBroadcastDomain()
	broadcastDomain.Join(iface, true)

	id2ip := make(map[int]net.IP)
	broadcastDomain.OnLeave(func(id int) {
		ip, ok := id2ip[id]
		if ok {
			ipPool.Put(ip)
		}
	})

	http.HandleFunc("/vpn", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			log.Panicf("error on websocket.Accept: %s", err.Error())
		}

		ip, err := assignIP(ctx, ipPool, ws)
		if err != nil {
			ws.Write(ctx, websocket.MessageText, []byte(err.Error()))
			return
		}

		id := broadcastDomain.Join(wsWrapper{ws}, false)
		id2ip[id] = ip
	})
	http.ListenAndServe(s.addr, nil)
}

func assignIP(ctx context.Context, ipPool *IPv4Pool, ws *websocket.Conn) (net.IP, error) {
	ip := ipPool.Get()
	if ip == nil {
		return nil, errors.New("ip pool empty")
	}
	ipMsg, _ := json.Marshal(IPAssignBody{IP: ip.To4().String()})
	ipMsg = append([]byte{PacketTypeIPAssign}, ipMsg...)
	err := ws.Write(ctx, websocket.MessageBinary, ipMsg)
	if err != nil {
		log.Panicf("error on ws.Write: %s", err.Error())
	}
	return ip, nil
}
