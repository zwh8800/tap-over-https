package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/songgao/water"
	"nhooyr.io/websocket"
)

var (
	cmdIsServer bool
	cmdAddr     string
)

func main() {
	parseCmd()

	iface, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		log.Panicf("error on water new: %s", err.Error())
	}

	log.Printf("iface: %s", iface.Name())

	if !cmdIsServer {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		ws, _, err := websocket.Dial(ctx, cmdAddr, nil)
		if err != nil {
			log.Panicf("error on websocket.Dial: %s", err.Error())
		}
		defer ws.Close(websocket.StatusInternalError, "the sky is falling")

		connectTunnel(ws, iface)
	} else {
		http.HandleFunc("/vpn", func(w http.ResponseWriter, r *http.Request) {
			ws, err := websocket.Accept(w, r, nil)
			if err != nil {
				log.Panicf("error on websocket.Accept: %s", err.Error())
			}
			defer ws.Close(websocket.StatusInternalError, "the sky is falling")
			connectTunnel(ws, iface)
		})
		http.ListenAndServe(cmdAddr, nil)
	}

	//packet := make([]byte, 2000)
	//for {
	//	n, err := iface.Read(packet)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	log.Printf("Packet Received: % x\n", packet[:n])
	//}
}

func connectTunnel(ws *websocket.Conn, iface *water.Interface) {
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		wsWriter, err := ws.Writer(ctx, websocket.MessageBinary)
		if err != nil {
			log.Panicf("error on ws.Writer: %s", err.Error())
		}

		_, err = io.Copy(wsWriter, iface)
		if err != nil {
			log.Panicf("error on io.Copy(wsWriter, iface): %s", err.Error())
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()

		_, wsReader, err := ws.Reader(ctx)
		if err != nil {
			log.Panicf("error on ws.Reader: %s", err.Error())
		}

		_, err = io.Copy(iface, wsReader)
		if err != nil {
			log.Panicf("error on io.Copy(iface, wsReader): %s", err.Error())
		}
	}()
	wg.Wait()
}

func parseCmd() {
	flag.BoolVar(&cmdIsServer, "s", false, "run as server mode")
	flag.StringVar(&cmdAddr, "addr", ":8012", "vpn server address in client mode\nbind address in server mode")
	flag.Parse()
}
