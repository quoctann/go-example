package streaming

import (
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

/*
Sequence number để detect packet loss
Tick-based updates: server gửi state 20 lần/giây (20Hz)
Heartbeat: để biết player còn online
No guaranteed delivery: nếu packet bị mất, ko sao, sử dụng state mới sau đó
*/

// Message in game protocol
const (
	MSG_WELCOME   = "WELCOME"
	MSG_JOIN      = "JOIN"
	MSG_MOVE      = "MOVE"
	MSG_SHOOT     = "SHOOT"
	MSG_STATE     = "STATE"
	MSG_HEARTBEAT = "HEARTBEAT"
	MSG_LEAVE     = "LEAVE"
)

type Player struct {
	ID       string
	Name     string
	X        float64
	Y        float64
	Health   int
	LastSeen time.Time
	Addr     *net.UDPAddr
}

type GameMessage struct {
	Type        string  `json:"type"`
	PlayerID    string  `json:"player_id"`
	Name        string  `json:"name,omitempty"`
	X           float64 `json:"x,omitempty"`
	Y           float64 `json:"y,omitempty"`
	Health      int     `json:"health,omitempty"`
	Timestamp   int64   `json:"timestamp"`
	SequenceNum uint32  `json:"seq,omitempty"`
}

type GameState struct {
	Players map[string]*Player `json:"players"`
}

type GameServer struct {
	conn          *net.UDPConn
	players       map[string]*Player
	playersByAddr map[string]string // addr -> playerID
	mu            sync.RWMutex
	tickRate      time.Duration
	sequenceNum   uint32
}

func newGameServer(port int) (*GameServer, error) {
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return nil, err
	}

	// Increase buffer size to prevent packet loss
	conn.SetReadBuffer(1048576) // 1MB
	conn.SetWriteBuffer(1048576)

	return &GameServer{
		conn:          conn,
		players:       make(map[string]*Player),
		playersByAddr: make(map[string]string),
		tickRate:      time.Millisecond * 50, // 20 updates/second (20Hz)
	}, nil
}

func (s *GameServer) handleMessage(data []byte, addr *net.UDPAddr) {
	var msg GameMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Invalid message: %v", err)
		return
	}

	addrStr := addr.String()
	msg.Timestamp = time.Now().UnixMilli()

	s.mu.Lock()
	defer s.mu.Unlock()

	switch msg.Type {
	case MSG_JOIN:
		// new player join
		player := &Player{
			ID:       msg.PlayerID,
			Name:     msg.Name,
			X:        100,
			Y:        100,
			Health:   100,
			LastSeen: time.Now(),
			Addr:     addr,
		}
		s.players[msg.PlayerID] = player
		s.playersByAddr[addrStr] = msg.PlayerID

		log.Printf("Player %s (%s) joined from %s", msg.Name, msg.PlayerID, addr)
		// send welcome message + current state
		s.sendWelcome(player)

	case MSG_MOVE:
		// update player location
		if playerId, exists := s.playersByAddr[addrStr]; exists {
			if player, ok := s.players[playerId]; ok {
				player.X = msg.X
				player.Y = msg.Y
				player.LastSeen = time.Now()
			}
		}

	case MSG_SHOOT:
		// decrease another player's health
		if playerId, exists := s.playersByAddr[addrStr]; exists {
			log.Printf("Player %s shoot at (%.2f, %.2f)", playerId, msg.X, msg.Y)
			// hit detection logic should be here
		}

	case MSG_HEARTBEAT:
		// update last seen to know player still online
		if playerId, exists := s.playersByAddr[addrStr]; exists {
			if player, ok := s.players[playerId]; ok {
				player.LastSeen = time.Now()
			}
		}

	}
}

func (s *GameServer) sendWelcome(player *Player) {
	welcome := GameMessage{
		Type:      MSG_WELCOME,
		PlayerID:  player.ID,
		Timestamp: time.Now().UnixMilli(),
	}

	data, _ := json.Marshal(welcome)
	s.conn.WriteToUDP(data, player.Addr)
}

// Game loop - broadcast state every tick
func (s *GameServer) gameLoop() {
	ticker := time.NewTicker(s.tickRate)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()

		// create game state snapshot
		state := GameMessage{
			Type:        MSG_STATE,
			Timestamp:   time.Now().UnixMilli(),
			SequenceNum: s.sequenceNum,
		}
		snapshot, _ := json.Marshal(state)
		s.sequenceNum++

		// serialize all game state
		stateData := GameState{Players: s.players}
		fullState, _ := json.Marshal(stateData)

		// broadcast to all players
		for _, player := range s.players {
			s.conn.WriteToUDP(fullState, player.Addr)
		}

		s.mu.RUnlock()

		// log status every 2s
		if s.sequenceNum%40 == 0 {
			s.mu.RLock()
			log.Printf("Tick #%d | Players online: %d | State size: %d bytes | Snapshot: %v",
				s.sequenceNum, len(s.players), len(fullState), string(snapshot))
			s.mu.RUnlock()
		}
	}
}

// Cleanup inactive player
func (s *GameServer) cleanupLoop() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()

		for id, player := range s.players {
			if now.Sub(player.LastSeen) > time.Second*10 {
				log.Printf("Player %s timed out", player.Name)
				delete(s.players, id)
				delete(s.playersByAddr, player.Addr.String())
			}
		}

		s.mu.Unlock()
	}
}

func (s *GameServer) start() {
	log.Printf("Game server started on UDP port %d", s.conn.LocalAddr().(*net.UDPAddr).Port)
	log.Printf("Tick rate: %v (%.0f updates/sec)", s.tickRate, 1000.0/float64(s.tickRate.Milliseconds()))
	log.Println("Waiting for players...")

	// start background task
	go s.gameLoop()
	go s.cleanupLoop()

	// listen for incoming packets, create a "bucket" buffer to read data
	buffer := make([]byte, 4096)

	for {
		n, addr, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Read error: %v", err)
			continue
		}

		// handle message in separate goroutine to not block the main loop
		// copy data to avoid overwrite in next read
		// only n bytes are valid
		data := make([]byte, n)
		copy(data, buffer[:n]) // 0 to n-1 bytes
		go s.handleMessage(data, addr)
	}
}

func RunGameServerExample(skip bool) {
	if skip {
		return
	}

	server, err := newGameServer(9999)
	if err != nil {
		log.Fatal(err)
	}

	server.start()
}
