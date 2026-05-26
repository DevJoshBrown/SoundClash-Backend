package hub

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // temp until clerk is implemented
	},
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request, battleID pgtype.UUID) {
	conn, err := upgrader.Upgrade(w, r, nil) //  converts a regular HTTP connection into a WebSocket connection.
	if err != nil {
		log.Printf("websocket: upgrade failed: %v", err)
		return
	}

	client := &Client{send: make(chan []byte, 256)}
	h := m.GetOrCreate(battleID)
	h.register(client)

	go writePump(conn, client, h)
	readPump(conn, client, h)
}

func readPump(conn *websocket.Conn, client *Client, h *Hub) {
	// disconnect the client once this func terminates
	defer func() {
		h.unregister(client)
		conn.Close()
	}()

	// sets a deadline to 60 seconds from now
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// once a pong is recived - resets the deadline = connection still active
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// continuiously read the websocket connection, break if we encounter an error
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writePump(conn *websocket.Conn, client *Client, h *Hub) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Close()

	}()

	for {
		select {
		case msg, ok := <-client.send:
			// check if client unregistered - close if so
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			conn.WriteMessage(websocket.TextMessage, msg)
		case <-ticker.C:
			// keep connection alive through idle periods (if we recieve a pong back)
			conn.WriteMessage(websocket.PingMessage, nil)
		}
	}
}
