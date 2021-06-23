package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

type BroadcastDomain struct {
	mu    sync.Mutex
	peers map[int]*broadcastPeer
}

type broadcastPeer struct {
	id       int
	rw       io.ReadWriteCloser
	mu       sync.Mutex
	needLock bool
}

func NewBroadcastDomain() *BroadcastDomain {
	return &BroadcastDomain{
		mu:    sync.Mutex{},
		peers: make(map[int]*broadcastPeer),
	}
}

func (b *BroadcastDomain) Join(rw io.ReadWriteCloser, needLock bool) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := rand.Intn(1000)
	i := 0
	for ; i < 3; i++ {
		if _, ok := b.peers[id]; !ok {
			break
		}
		id = rand.Intn(100)
	}
	if i == 3 {
		log.Panicf("you are so lucky")
	}

	b.peers[id] = &broadcastPeer{
		id:       id,
		rw:       rw,
		mu:       sync.Mutex{},
		needLock: needLock,
	}

	go func() {
		defer func() {
			err := recover()
			if err != nil {
				log.Printf("panic on rw read: %#v", err)
				b.Leave(id)
			}
		}()
		for {
			buffer := make([]byte, 2048)

			n, err := rw.Read(buffer)
			if err != nil {
				log.Panicf("error on rw.Read: %s", err.Error())
			}
			//log.Printf("Packet From %04d: % x\n", id, buffer[:n])

			b.mu.Lock()
			for peerID, peer := range b.peers {
				if peerID == id {
					continue
				}
				go func(peer *broadcastPeer) {
					defer func() {
						err := recover()
						if err != nil {
							log.Printf("panic on rw write: %#v", err)
							b.Leave(peer.id)
						}
					}()
					if peer.needLock {
						peer.mu.Lock()
						defer peer.mu.Unlock()
					}
					_, err := peer.rw.Write(buffer[:n])
					if err != nil {
						log.Panicf("error on rw.Write: %s", err.Error())
					}

				}(peer)
			}
			b.mu.Unlock()
		}
	}()
	return id
}

func (b *BroadcastDomain) Leave(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	peer := b.peers[id]
	if peer != nil {
		peer.rw.Close()
	}
	delete(b.peers, id)
}

func main() {
	parseCmd()

	iface, err := water.New(Config)
	if err != nil {
		log.Panicf("error on water new: %s", err.Error())
	}

	log.Printf("iface: %s", iface.Name())

	if !cmdIsServer {
		runAsClient(iface)
	} else {
		runAsServer(iface)
	}
}

func runAsServer(iface *water.Interface) {
	startIP := net.ParseIP(cmdIpStart)
	endIP := net.ParseIP(cmdIpEnd)
	ipPool, err := NewIPPool(startIP, endIP)
	if err != nil {
		log.Panicf("error on NewIPPool: %s", err.Error())
	}

	createBridge(iface.Name())

	broadcastDomain := NewBroadcastDomain()
	broadcastDomain.Join(iface, true)

	http.HandleFunc("/vpn", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			log.Panicf("error on websocket.Accept: %s", err.Error())
		}

		if err := assignIP(ctx, ipPool, ws); err != nil {
			ws.Write(ctx, websocket.MessageText, []byte(err.Error()))
			return
		}

		broadcastDomain.Join(wsWrapper{ws}, false)
	})
	http.ListenAndServe(cmdAddr, nil)
}

func runAsClient(iface *water.Interface) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	ws, _, err := websocket.Dial(ctx, cmdAddr, &websocket.DialOptions{
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

func waitSignal(sig ...os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig...)
	<-c
}

type wsWrapper struct {
	c *websocket.Conn
}

func (w wsWrapper) Read(p []byte) (int, error) {
	ctx := context.Background()
	_, data, err := w.c.Read(ctx)
	if err != nil {
		return 0, err
	}
	if len(data) < 1 || data[0] != PacketTypeData {
		return 0, fmt.Errorf("PacketTypeData type error: % x", data)
	}

	return copy(p, data[1:]), nil
}

func (w wsWrapper) Write(p []byte) (int, error) {
	ctx := context.Background()
	err := w.c.Write(ctx, websocket.MessageBinary, append([]byte{PacketTypeData}, p...))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w wsWrapper) Close() error {
	return w.c.Close(websocket.StatusNormalClosure, "bye")
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

func parseCmd() {
	flag.BoolVar(&cmdIsServer, "s", false, "run as server mode")
	flag.StringVar(&cmdAddr, "addr", ":8012", "vpn server address in client mode\nbind address in server mode")
	flag.StringVar(&cmdIpStart, "ip-start", "10.0.0.80", "ip start for client")
	flag.StringVar(&cmdIpEnd, "ip-end", "10.0.0.100", "ip end for client")
	flag.StringVar(&cmdIFaceBridge, "i", "br-lan", "bridge interface")
	flag.Parse()
}
