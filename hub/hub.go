package hub

import (
	"time"

	"github.com/gorilla/websocket"
)

//SuperHub keeps track of all repoId and there Hubs
type SuperHub map[string]*SingleHub

//SendEventToRepo to send data to the repo channel
func (sh SuperHub) SendEventToRepo(repoName string, data []byte) {
	channel, ok := sh[repoName]

	if ok {
		channel.Broadcast <- data
	}
}

//SingleHub a single hub entity which
type SingleHub struct {
	//repoName for which hub is created
	RepoName string

	// Registered clients.
	Clients map[*Client]bool

	// Inbound messages from the clients.
	Broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client
}

//Run Start the hub to listen to events on it
func (h *SingleHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}

//Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub *SingleHub

	// The websocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound messages.
	Send chan []byte
}

func (client *Client) Write() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		client.Hub.Unregister <- client
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				// The hub closed the channel.
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)

			if err != nil {
				return
			}

			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(client.Send)

			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

//CreateNewHub to create new hub for repo
func CreateNewHub(repoName string) *SingleHub {
	return &SingleHub{
		RepoName:   repoName,
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}
}

//SuperHubInstance the global hub instance to be used everywhere
var SuperHubInstance SuperHub = make(SuperHub)
