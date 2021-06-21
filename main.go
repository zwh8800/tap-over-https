package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/zwh8800/tap-over-https/conf"

	"github.com/songgao/water"
	"nhooyr.io/websocket"
)

var (
	cmdIsServer    bool
	cmdAddr        string
	cmdIpStart     string
	cmdIpEnd       string
	cmdIFaceBridge string
)

const (
	PacketTypeData byte = iota
	PacketTypeIPAssign
)

type IPAssignBody struct {
	IP string `json:"ip"`
}

type IPv4Pool struct {
	mu    sync.Mutex
	start net.IP
	end   net.IP
	cur   net.IP
	pool  map[uint32]bool
}

func NewIPPool(start, end net.IP) (*IPv4Pool, error) {
	start = start.To4()
	end = end.To4()
	if start == nil || end == nil {
		return nil, fmt.Errorf("start end must be ipv4")
	}
	return &IPv4Pool{
		mu:    sync.Mutex{},
		start: start,
		end:   end,
		cur:   start,
		pool:  make(map[uint32]bool),
	}, nil
}

func (p *IPv4Pool) Get() net.IP {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := binary.BigEndian.Uint32(p.cur)
	start := binary.BigEndian.Uint32(p.start)
	end := binary.BigEndian.Uint32(p.end)
	next := p.nextIP(cur, end, start)
	if next == 0 {
		return nil
	}
	binary.BigEndian.PutUint32(p.cur, next)
	return p.cur
}

func (p *IPv4Pool) nextIP(cur uint32, end uint32, start uint32) uint32 {
	for i := cur + 1; i < end; i++ {
		if !p.pool[i] {
			p.pool[i] = true
			return i
		}
	}
	for i := start; i < cur; i++ {
		if !p.pool[i] {
			p.pool[i] = true
			return i
		}
	}
	return 0
}

func (p *IPv4Pool) Put(ip net.IP) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pool, binary.BigEndian.Uint32(ip))
}

func main() {
	parseCmd()

	iface, err := water.New(conf.Config)
	if err != nil {
		log.Panicf("error on water new: %s", err.Error())
	}

	log.Printf("iface: %s", iface.Name())

	if !cmdIsServer {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ws, _, err := websocket.Dial(ctx, cmdAddr, nil)
		if err != nil {
			log.Panicf("error on websocket.Dial: %s", err.Error())
		}
		defer ws.Close(websocket.StatusNormalClosure, "bye")

		handleIPAssign(ctx, ws, iface)

		connectTunnel(ws, iface)
	} else {
		startIP := net.ParseIP(cmdIpStart)
		endIP := net.ParseIP(cmdIpEnd)
		ipPool, err := NewIPPool(startIP, endIP)
		if err != nil {
			log.Panicf("error on NewIPPool: %s", err.Error())
		}

		createBridge(iface.Name())

		http.HandleFunc("/vpn", func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()

			ws, err := websocket.Accept(w, r, nil)
			if err != nil {
				log.Panicf("error on websocket.Accept: %s", err.Error())
			}
			defer ws.Close(websocket.StatusNormalClosure, "bye")

			if err := assignIP(ctx, ipPool, ws); err != nil {
				ws.Write(ctx, websocket.MessageText, []byte(err.Error()))
				return
			}

			connectTunnel(ws, iface)
		})
		http.ListenAndServe(cmdAddr, nil)
	}
}

func assignIP(ctx context.Context, ipPool *IPv4Pool, ws *websocket.Conn) error {
	ip := ipPool.Get()
	if ip == nil {
		return errors.New("ip pool empty")
	}
	ipMsg, _ := json.Marshal(IPAssignBody{IP: ip.To4().String()})
	ipMsg = append([]byte{PacketTypeIPAssign}, ipMsg...)
	err := ws.Write(ctx, websocket.MessageBinary, ipMsg)
	if err != nil {
		log.Panicf("error on ws.Write: %s", err.Error())
	}
	return nil
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

func connectTunnel(ws *websocket.Conn, iface *water.Interface) {
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		packet := make([]byte, 4*1024*1024)
		packet[0] = PacketTypeData
		for {
			n, err := iface.Read(packet[1:])
			if err != nil {
				log.Panicf("error on iface.Read: %s", err.Error())
			}
			log.Printf("Packet From tap: % x\n", packet[1:n+1])

			err = ws.Write(ctx, websocket.MessageBinary, packet[:n+1])
			if err != nil {
				log.Panicf("error on ws.Write: %s", err.Error())
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			_, packet, err := ws.Read(ctx)
			if err != nil {
				log.Panicf("error on ws.Read: %s", err.Error())
			}
			log.Printf("Packet From ws : % x\n", packet)
			if len(packet) < 1 || packet[0] != PacketTypeData {
				log.Panicf("PacketTypeData type error: % x", packet)
			}

			_, err = iface.Write(packet[1:])
			if err != nil {
				log.Panicf("error on iface.Write: %s", err.Error())
			}
		}
	}()
	wg.Wait()
}

func parseCmd() {
	flag.BoolVar(&cmdIsServer, "s", false, "run as server mode")
	flag.StringVar(&cmdAddr, "addr", ":8012", "vpn server address in client mode\nbind address in server mode")
	flag.StringVar(&cmdIpStart, "ip-start", "10.0.0.80", "ip start for client")
	flag.StringVar(&cmdIpEnd, "ip-end", "10.0.0.100", "ip end for client")
	flag.StringVar(&cmdIFaceBridge, "i", "br-lan", "bridge interface")
	flag.Parse()
}
