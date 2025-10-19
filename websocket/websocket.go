package websocket

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

/*
Note:
- Hub pattern: central hub to manage all clients and broadcast messages
- Channels:
	+ Broadcast: messages to all connected clients
	+ Register/Unregister: add/remove clients
	+ Send: queue messages to individual clients

- Goroutines:
	+ readPump: read messages from client
	+ writePump: send messages to client
	Run parrelly to handle multiple clients concurrently

- Thread safety:
	+ Use mutex to protect shared resources (clients map)

- Ping/Pong mechanism
	+ Server send "ping" every 54s (in this example)
	+ Client automatically answer "pong"
	+ If not receive any "pong" after 60s, connection considered as dead and clean up
	+ ReadDeadline (60s): timeout if client cannot send data
	+ WriteDeadline (10s): timeout if server cannot send the message

Production best practice
	1. Add rate limiting
	2. Authentication/Authorization
	3. Message validation & sanitization
	4. Monitoring (metrics, alerts)
	5. Load balancing with Redis Pub/Sub
	6. Connection pool limits
*/

// Message represents a WebSocket message structure
type Message struct {
	Username string `json:"username"`
	Content  string `json:"content"`
	Type     string `json:"type"` // e.g., "join", "leave", "message"
}

// Client present for single client connection
type Client struct {
	conn     *websocket.Conn
	username string
	send     chan Message // Channel sending messages to the client
}

// Hub manages all active WebSocket clients
type Hub struct {
	clients    map[*Client]bool // map of active clients
	broadcast  chan Message     // channel receiving broadcast messages from clients
	register   chan *Client     // channel to register new clients
	unregister chan *Client     // channel to unregister clients
	mu         sync.Mutex       // mutex to ensure thread-safe operations when accessing clients map
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// run is main loop for the Hub to handle hub events
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register: // add client into hub
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client %s connected. Total: %d", client.username, len(h.clients))

		case client := <-h.unregister: // remove client from hub and close its send channel
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client %s disconnected. Total: %d", client.username, len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast: // broadcast message to all clients
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
					// send successful
				default:
					// channel full or closed, remove client
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// readPump reads messages from the WebSocket and sends them to broadcast channel
func (c Client) readPump(hub *Hub) {
	// ensure connection is closed on exit
	defer func() {
		// send message before unregister
		hub.broadcast <- Message{
			Username: c.username,
			Content:  "left the chat",
			Type:     "leave",
		}

		hub.unregister <- &c
		c.conn.Close()

		log.Printf("Client %s clean up completed", c.username)
	}()

	// config timeout and ping/pong to detect dead connections
	// SetReadDeadline: if no message received in 60 seconds, connection is considered dead
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// SetPongHandler: when pong received, reset deadline
	c.conn.SetPongHandler(func(appData string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		// read json from WebSocket
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			// if error is unexpected close error, log it
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("abnormal client close: %v", err)
			} else if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("client %s normal close", c.username)
			} else {
				log.Printf("error read message from %s: %v", c.username, err)
			}
			break
		}

		// reset deadline after message
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// assign username to message
		msg.Username = c.username

		// send message to broadcast channel
		hub.broadcast <- msg
	}
}

// writePump sends messages from the send channel to the WebSocket
func (c *Client) writePump() {
	// ticker to send frequently message
	ticker := time.NewTicker(54 * time.Second) // ping every 54s (before timeout 60s)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send: // wait for message from send channel
			// set write deadline: have to write in 10s
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// channel closed, send close message
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// send json message to WebSocket
			err := c.conn.WriteJSON(msg)
			if err != nil {
				log.Printf("error write: %v", err)
				return
			}

		case <-ticker.C:
			// send ping message to detect connection alive or not
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("ping failed of %s: %v", c.username, err)
				return
			}
		}
	}
}

// upgrader changes HTTP connection to WebSocket connection
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections by default - in production, you might want to check the origin here
		return true
	},
}

func serveWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "Anonymous"
	}

	// upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// create new client
	client := &Client{
		conn:     conn,
		username: username,
		send:     make(chan Message, 256), // buffered 256 messages
	}

	// register client to hub
	hub.register <- client

	// send join message
	hub.broadcast <- Message{
		Username: client.username,
		Content:  "joined the chat",
		Type:     "join",
	}

	// run write pump and read pump in separate goroutines
	go client.writePump()
	go client.readPump(hub)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlContent))
}

func RunExample(isSkip bool) {
	if isSkip {
		return
	}

	hub := NewHub() // create new hub
	go hub.run()    // run hub in separate goroutine

	// route for WebSocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWebSocket(hub, w, r)
	})

	// route for html client
	http.HandleFunc("/", serveHome)

	// start HTTP server
	addr := ":8080"
	log.Printf("WebSocket server started at %s", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
