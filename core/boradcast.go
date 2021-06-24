package core

import (
	"io"
	"log"
	"math/rand"
	"sync"
)

type BroadcastDomain struct {
	peers   sync.Map
	onLeave func(id int)
}

type broadcastPeer struct {
	id       int
	rw       io.ReadWriteCloser
	mu       sync.Mutex
	needLock bool
}

func NewBroadcastDomain() *BroadcastDomain {
	return &BroadcastDomain{}
}

func (b *BroadcastDomain) Join(rw io.ReadWriteCloser, needLock bool) int {
	id := rand.Intn(1000)
	i := 0
	for ; i < 3; i++ {
		if _, ok := b.peers.Load(id); !ok {
			break
		}
		id = rand.Intn(100)
	}
	if i == 3 {
		log.Panicf("you are so lucky")
	}

	b.peers.Store(id, &broadcastPeer{
		id:       id,
		rw:       rw,
		mu:       sync.Mutex{},
		needLock: needLock,
	})

	go func() {
		defer func() {
			err := recover()
			if err != nil {
				log.Printf("panic on rw read: %#v", err)
				b.Leave(id)
			}
		}()
		buffer := make([]byte, 2048)
		for {
			n, err := rw.Read(buffer)
			if err != nil {
				log.Panicf("error on rw.Read: %s", err.Error())
			}
			//log.Printf("Packet From %04d: % x\n", id, buffer[:n])

			var wg sync.WaitGroup
			b.peers.Range(func(key, value interface{}) bool {
				peerID, peer := key.(int), value.(*broadcastPeer)

				if peerID == id {
					return true
				}
				wg.Add(1)
				go func(peer *broadcastPeer) {
					defer wg.Done()
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
				return true
			})
			wg.Wait()
		}
	}()
	return id
}

func (b *BroadcastDomain) Leave(id int) {
	peer, ok := b.peers.LoadAndDelete(id)
	if ok {
		peer.(*broadcastPeer).rw.Close()
	}
	if b.onLeave != nil {
		b.onLeave(id)
	}
}

func (b *BroadcastDomain) OnLeave(onLeave func(id int)) {
	b.onLeave = onLeave
}
