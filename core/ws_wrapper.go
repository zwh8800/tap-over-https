package core

import (
	"context"
	"fmt"
	"time"

	"nhooyr.io/websocket"
)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := w.c.Write(ctx, websocket.MessageBinary, append([]byte{PacketTypeData}, p...))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w wsWrapper) Close() error {
	return w.c.Close(websocket.StatusNormalClosure, "bye")
}
