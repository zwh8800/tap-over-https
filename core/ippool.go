package core

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

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
	var cur = make(net.IP, len(start))
	copy(cur, start) // net.IP is a slice, need deep copy
	return &IPv4Pool{
		mu:    sync.Mutex{},
		start: start,
		end:   end,
		cur:   cur,
		pool:  make(map[uint32]bool),
	}, nil
}

func (p *IPv4Pool) Get() net.IP {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := binary.BigEndian.Uint32(p.cur)
	start := binary.BigEndian.Uint32(p.start)
	end := binary.BigEndian.Uint32(p.end)
	next := p.nextIP(cur, start, end)
	if next == 0 {
		return nil
	}
	binary.BigEndian.PutUint32(p.cur, next)
	return p.cur
}

func (p *IPv4Pool) nextIP(cur uint32, start uint32, end uint32) uint32 {
	for i := cur + 1; i < end; i++ {
		if !p.pool[i] {
			p.pool[i] = true
			return i
		}
	}
	for i := start; i < cur+1; i++ {
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
	delete(p.pool, binary.BigEndian.Uint32(ip.To4()))
}
