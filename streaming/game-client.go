package streaming

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

type GameClient struct {
	conn            *net.UDPConn
	serverAddr      *net.UDPAddr
	playerID        string
	playerName      string
	x, y            float64
	lastSequence    uint32
	packetsReceived int
	packetsLost     int
}

func newGameClient(serverHost, playerName string) (*GameClient, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", serverHost)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return nil, err
	}

	return &GameClient{
		conn:       conn,
		serverAddr: serverAddr,
		playerID:   fmt.Sprintf("player_%d", rand.Intn(10000)),
		playerName: playerName,
		x:          100,
		y:          100,
	}, nil
}

func (c *GameClient) sendMessage(msg GameMessage) error {
	msg.Timestamp = time.Now().UnixMilli()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(data)
	return err
}

func (c *GameClient) join() error {
	log.Printf("Joining game as '%s' (ID: %s)", c.playerName, c.playerID)
	return c.sendMessage(GameMessage{
		Type:     MSG_JOIN,
		PlayerID: c.playerID,
		Name:     c.playerName,
	})
}

func (c *GameClient) move(dx, dy float64) {
	c.x += dx
	c.y += dy

	// Clamp location in map
	if c.x < 0 {
		c.x = 0
	}
	if c.x > 1000 {
		c.x = 1000
	}
	if c.y < 0 {
		c.y = 0
	}
	if c.y > 1000 {
		c.y = 1000
	}

	c.sendMessage(GameMessage{
		Type:     MSG_MOVE,
		PlayerID: c.playerID,
		X:        c.x,
		Y:        c.y,
	})
}

func (c *GameClient) shoot(targetX, targetY float64) {
	c.sendMessage(GameMessage{
		Type:     MSG_SHOOT,
		PlayerID: c.playerID,
		X:        targetX,
		Y:        targetY,
	})
	log.Printf("Shoot at (%.0f, %.0f)", targetX, targetY)
}

// Listen update for state from server
func (c *GameClient) listenForUpdates() {
	buffer := make([]byte, 4096)

	for {
		c.conn.SetReadDeadline(time.Now().Add(time.Second * 15))
		n, err := c.conn.Read(buffer)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Println("Server timeout - connection lost")
				return
			}
			log.Printf("Read error: %v", err)
			continue
		}

		// parse message type
		var msg map[string]any
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			continue
		}
		msgType, _ := msg["type"].(string)

		switch msgType {
		case MSG_WELCOME:
			log.Println("Connected to server")

		case MSG_STATE:
			// parse full game state
			var state GameState
			if err := json.Unmarshal(buffer[:n], &state); err == nil {
				c.packetsReceived++

				// detect packet loss by sequence number
				if seq, ok := msg["seq"].(float64); ok {
					seqNum := uint32(seq)
					if c.lastSequence > 0 && seqNum > c.lastSequence+1 {
						lost := seqNum - c.lastSequence - 1
						c.packetsLost += int(lost)
					}
					c.lastSequence = seqNum
				}

				// show state every 20 updates
				if c.packetsReceived%20 == 0 {
					c.displayState(&state)
				}
			}
		}
	}
}

func (c *GameClient) displayState(state *GameState) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("GAME STATE (Tick #%d)\n", c.lastSequence)
	fmt.Printf("Packets: %d received | %d lost (%.1f%% loss rate)\n",
		c.packetsReceived, c.packetsLost,
		float64(c.packetsLost)/float64(c.packetsReceived+c.packetsLost)*100)
	fmt.Println(strings.Repeat("-", 60))

	for _, player := range state.Players {
		marker := "  "
		if player.ID == c.playerID {
			marker = "-> "
		}
		fmt.Printf("%s%s [%s] at (%.0f, %.0f) HP: %d\n",
			marker, player.Name, player.ID, player.X, player.Y, player.Health)
	}
	fmt.Println(strings.Repeat("=", 60))
}

// Send heart beet periodically
func (c *GameClient) heartbeatLoop() {
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

	for range ticker.C {
		c.sendMessage(GameMessage{
			Type:     MSG_HEARTBEAT,
			PlayerID: c.playerID,
		})
	}
}

// Bot mode make random move and shoot
func (c *GameClient) botMode() {
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	directions := []struct{ dx, dy float64 }{
		{10, 0}, {-10, 0}, {0, 10}, {0, -10},
		{7, 7}, {-7, 7}, {7, -7}, {-7, -7},
	}

	for range ticker.C {
		d := directions[rand.Intn(len(directions))]
		c.move(d.dx, d.dy)

		// random shoot 5% chance
		if rand.Float32() < 0.05 {
			c.shoot(rand.Float64()*1000, rand.Float64()*1000)
		}
	}
}

func (c *GameClient) interactiveMode() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf(`Controls:
	w/a/s/d - Move
	space   - Shoot at random
	q       - Quit`)

	for {
		fmt.Print("\n>>> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "w":
			c.move(0, -20)
		case "s":
			c.move(0, 20)
		case "a":
			c.move(-20, 0)
		case "d":
			c.move(20, 0)
		case " ", "space":
			c.shoot(rand.Float64()*1000, rand.Float64()*1000)
		case "q", "quit":
			log.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid command")
		}
	}
}

func RunGameClientExample(skip bool) {
	if skip {
		return
	}

	if len(os.Args) < 2 {
		fmt.Printf(`
		Usage: go run client.go <player_name> [bot]
		Examples:
			go run client.go Alice
			go run client.go BotPlayer bot
		`)
		os.Exit(1)
	}

	playerName := os.Args[1]
	botMode := len(os.Args) > 2 && os.Args[2] == "bot"

	client, err := newGameClient("localhost:9999", playerName)
	if err != nil {
		log.Fatal(err)
	}
	defer client.conn.Close()

	// join game
	if err := client.join(); err != nil {
		log.Fatal(err)
	}

	// start listening for updates
	go client.listenForUpdates()

	// start heart beat
	go client.heartbeatLoop()

	// wait for connection
	time.Sleep(time.Millisecond * 500)

	// start game loop
	if botMode {
		log.Println("Bot mode activated - auto playing")
		client.botMode()
	} else {
		client.interactiveMode()
	}
}
