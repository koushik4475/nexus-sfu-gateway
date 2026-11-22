package signaling

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrPeerNotFound = errors.New("peer not found")
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Non-browser clients commonly omit Origin; browser clients must
			// present the same host as the signaling endpoint.
			return true
		}
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return false
		}
		return strings.EqualFold(parsed.Host, r.Host)
	},
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// SignalMessageType defines protocol actions.
type SignalMessageType string

const (
	MsgJoin         SignalMessageType = "join"
	MsgOffer        SignalMessageType = "offer"
	MsgAnswer       SignalMessageType = "answer"
	MsgCandidate    SignalMessageType = "candidate"
	MsgLeave        SignalMessageType = "leave"
	MsgPeerJoined   SignalMessageType = "peer_joined"
	MsgPeerLeft     SignalMessageType = "peer_left"
	MsgTrackAdded   SignalMessageType = "track_added"
	MsgError        SignalMessageType = "error"
)

// SignalMessage is the JSON envelope exchanged over WebSocket.
type SignalMessage struct {
	Type      SignalMessageType           `json:"type"`
	RoomID    string                      `json:"roomId,omitempty"`
	PeerID    string                      `json:"peerId,omitempty"`
	SDP       *webrtc.SessionDescription  `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidateInit    `json:"candidate,omitempty"`
	TrackID   string                      `json:"trackId,omitempty"`
	Error     string                      `json:"error,omitempty"`
}

// Client represents a connected WebSocket client session.
type Client struct {
	ID        string
	RoomID    string
	Conn      *websocket.Conn
	SendChan  chan []byte
	closed    bool
	mu        sync.Mutex
	lastPing  time.Time
}

// Send queues a message for transmission to the client.
func (c *Client) Send(msg *SignalMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("client connection closed")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.SendChan <- data:
		return nil
	default:
		return errors.New("send buffer full")
	}
}

// Close disconnects the client.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.SendChan)
		_ = c.Conn.Close()
	}
}

// Room manages client membership and dispatching within a meeting context.
type Room struct {
	ID      string
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewRoom creates a room instance.
func NewRoom(id string) *Room {
	return &Room{
		ID:      id,
		clients: make(map[string]*Client),
	}
}

// AddClient attaches a client to the room.
func (r *Room) AddClient(c *Client) {
	r.mu.Lock()
	r.clients[c.ID] = c
	r.mu.Unlock()

	// Broadcast peer_joined to all other peers
	r.BroadcastExcept(c.ID, &SignalMessage{
		Type:   MsgPeerJoined,
		RoomID: r.ID,
		PeerID: c.ID,
	})
}

// RemoveClient detaches a client and notifies remaining participants.
func (r *Room) RemoveClient(clientID string) {
	r.mu.Lock()
	delete(r.clients, clientID)
	r.mu.Unlock()

	r.BroadcastExcept(clientID, &SignalMessage{
		Type:   MsgPeerLeft,
		RoomID: r.ID,
		PeerID: clientID,
	})
}

// BroadcastExcept dispatches a message to all peers in the room except excludedPeerID.
func (r *Room) BroadcastExcept(excludedPeerID string, msg *SignalMessage) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for id, client := range r.clients {
		if id != excludedPeerID {
			_ = client.Send(msg)
		}
	}
}

// ClientCount returns the number of active participants in the room.
func (r *Room) ClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// RoomManager coordinates multi-room lifecycle.
type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

// NewRoomManager instantiates room registry.
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

// GetOrCreateRoom returns the room for roomID, creating one if not existing.
func (rm *RoomManager) GetOrCreateRoom(roomID string) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		room = NewRoom(roomID)
		rm.rooms[roomID] = room
	}
	return room
}

// GetRoom retrieves existing room.
func (rm *RoomManager) GetRoom(roomID string) (*Room, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return nil, ErrRoomNotFound
	}
	return room, nil
}

// RemoveRoom deletes an empty room.
func (rm *RoomManager) RemoveRoom(roomID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, roomID)
}

// Server handles WebSocket HTTP endpoints for WebRTC signaling.
type Server struct {
	RoomManager *RoomManager
	mu          sync.RWMutex
	clients     map[string]*Client
}

// NewServer creates a signaling HTTP server instance.
func NewServer() *Server {
	return &Server{
		RoomManager: NewRoomManager(),
		clients:     make(map[string]*Client),
	}
}

// HandleWebSocket upgrades incoming HTTP connections to WebSocket signaling channels.
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	peerID := r.URL.Query().Get("peerId")
	roomID := r.URL.Query().Get("roomId")
	if peerID == "" {
		peerID = fmt.Sprintf("peer-%d", time.Now().UnixNano())
	}
	if roomID == "" {
		roomID = "default"
	}

	client := &Client{
		ID:       peerID,
		RoomID:   roomID,
		Conn:     conn,
		SendChan: make(chan []byte, 128),
		lastPing: time.Now(),
	}

	s.mu.Lock()
	s.clients[peerID] = client
	s.mu.Unlock()

	room := s.RoomManager.GetOrCreateRoom(roomID)
	room.AddClient(client)

	// Start write and read pumps
	go s.writePump(client)
	go s.readPump(client, room)
}

func (s *Server) writePump(c *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case msg, ok := <-c.SendChan:
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) readPump(c *Client, room *Room) {
	defer func() {
		room.RemoveClient(c.ID)
		if room.ClientCount() == 0 {
			s.RoomManager.RemoveRoom(room.ID)
		}
		s.mu.Lock()
		delete(s.clients, c.ID)
		s.mu.Unlock()
		c.Close()
	}()

	c.Conn.SetReadLimit(65536)
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg SignalMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		msg.PeerID = c.ID
		msg.RoomID = room.ID

		switch msg.Type {
		case MsgOffer, MsgAnswer, MsgCandidate, MsgTrackAdded:
			// Forward signaling messages to other peers in room
			room.BroadcastExcept(c.ID, &msg)
		case MsgLeave:
			return
		}
	}
}
