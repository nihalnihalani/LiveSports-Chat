package websocket

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/gorilla/websocket"
    "go.uber.org/zap"
    "golang.org/x/time/rate"

    "github.com/yourusername/sports-chat/internal/metrics"
    "github.com/yourusername/sports-chat/internal/models"
    "github.com/yourusername/sports-chat/internal/store"
)

type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte
    user     *models.User
    rooms    map[string]bool
    limiter  *rate.Limiter
    mu       sync.RWMutex
}

type Hub struct {
    // Clients and rooms
    clients    map[*Client]bool
    rooms      map[string]map[*Client]bool
    
    // Channels for client registration and message broadcasting
    register   chan *Client
    unregister chan *Client
    broadcast  chan *models.WSMessage
    
    // Dependencies
    store      store.Store
    metrics    *metrics.Metrics
    logger     *zap.Logger
    
    // Synchronization
    mu         sync.RWMutex
    
    // Match updates
    matches    map[string]*models.Match
    matchMu    sync.RWMutex
    
    // Rate limiting
    roomLimiters map[string]*rate.Limiter
}

func NewHub(store store.Store, metrics *metrics.Metrics, logger *zap.Logger) *Hub {
    return &Hub{
        clients:      make(map[*Client]bool),
        rooms:        make(map[string]map[*Client]bool),
        register:     make(chan *Client),
        unregister:   make(chan *Client),
        broadcast:    make(chan *models.WSMessage),
        store:        store,
        metrics:      metrics,
        logger:       logger,
        matches:      make(map[string]*models.Match),
        roomLimiters: make(map[string]*rate.Limiter),
    }
}

func (h *Hub) Run() {
    // Start match update goroutine
    go h.updateMatches()

    for {
        select {
        case client := <-h.register:
            h.handleRegister(client)

        case client := <-h.unregister:
            h.handleUnregister(client)

        case message := <-h.broadcast:
            h.handleBroadcast(message)
        }
    }
}

func (h *Hub) handleRegister(client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.clients[client] = true

    // Send recent match events and chat history
    go h.sendInitialData(client)

    // Broadcast user join to relevant rooms
    for room := range client.rooms {
        if _, exists := h.rooms[room]; !exists {
            h.rooms[room] = make(map[*Client]bool)
        }
        h.rooms[room][client] = true

        joinMsg := &models.WSMessage{
            Type:      models.MessageTypeJoin,
            ChatRoom:  room,
            User:      client.user,
            Timestamp: time.Now(),
        }
        h.broadcastToRoom(room, joinMsg)
    }

    // Update metrics
    h.metrics.ConnectedClients.Inc()
}

func (h *Hub) handleUnregister(client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()

    if _, ok := h.clients[client]; ok {
        delete(h.clients, client)
        close(client.send)

        // Remove from all rooms and broadcast leave message
        for room := range client.rooms {
            if clients, exists := h.rooms[room]; exists {
                delete(clients, client)
                if len(clients) == 0 {
                    delete(h.rooms, room)
                }

                leaveMsg := &models.WSMessage{
                    Type:      models.MessageTypeLeave,
                    ChatRoom:  room,
                    User:      client.user,
                    Timestamp: time.Now(),
                }
                h.broadcastToRoom(room, leaveMsg)
            }
        }

        // Update metrics
        h.metrics.ConnectedClients.Dec()
    }
}

func (h *Hub) handleBroadcast(message *models.WSMessage) {
    // Validate rate limits
    if !h.checkRateLimit(message.ChatRoom) {
        return
    }

    // Store chat message if it's a chat type message
    if message.Type == models.MessageTypeChat {
        go h.persistMessage(message)
    }

    // Broadcast to room
    h.broadcastToRoom(message.ChatRoom, message)

    // Update metrics
    h.metrics.MessagesSent.Inc()
}

func (h *Hub) broadcastToRoom(room string, message *models.WSMessage) {
    payload, err := json.Marshal(message)
    if err != nil {
        h.logger.Error("Failed to marshal message",
            zap.Error(err),
            zap.String("room", room))
        return
    }

    h.mu.RLock()
    clients := h.rooms[room]
    h.mu.RUnlock()

    for client := range clients {
        select {
        case client.send <- payload:
        default:
            go h.unregister <- client
        }
    }
}

func (h *Hub) checkRateLimit(room string) bool {
    h.mu.Lock()
    limiter, exists := h.roomLimiters[room]
    if !exists {
        limiter = rate.NewLimiter(rate.Every(time.Second), 10) // 10 messages per second per room
        h.roomLimiters[room] = limiter
    }
    h.mu.Unlock()

    return limiter.Allow()
}

func (h *Hub) persistMessage(message *models.WSMessage) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    msg := &models.Message{
        ChatRoomID:  message.ChatRoom,
        UserID:      message.User.ID,
        Content:     message.Content,
        MessageType: models.MessageTypeChat,
        CreatedAt:   message.Timestamp,
    }

    if err := h.store.CreateMessage(ctx, msg); err != nil {
        h.logger.Error("Failed to persist message",
            zap.Error(err),
            zap.String("room", message.ChatRoom),
            zap.String("user_id", message.User.ID))
    }
}

func (h *Hub) sendInitialData(client *Client) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    for room := range client.rooms {
        // Send recent messages
        messages, err := h.store.GetRecentMessages(ctx, room, 50)
        if err != nil {
            h.logger.Error("Failed to get recent messages",
                zap.Error(err),
                zap.String("room", room))
            continue
        }

        for _, msg := range messages {
            wsMsg := &models.WSMessage{
                Type:      models.MessageTypeChat,
                ChatRoom:  room,
                Content:   msg.Content,
                User:      msg.User,
                Timestamp: msg.CreatedAt,
            }

            payload, err := json.Marshal(wsMsg)
            if err != nil {
                continue
            }

            select {
            case client.send <- payload:
            default:
                return
            }
        }

        // Send match data if available
        // Send match data if available
        h.matchMu.RLock()
        if match, exists := h.matches[room]; exists {
            matchMsg := &models.WSMessage{
                Type:      models.MessageTypeEvent,
                ChatRoom:  room,
                Match:     match,
                Timestamp: time.Now(),
            }

            payload, err := json.Marshal(matchMsg)
            if err == nil {
                client.send <- payload
            }
        }
        h.matchMu.RUnlock()
    }
}

// Match update goroutine
func (h *Hub) updateMatches() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            h.fetchMatchUpdates()
        }
    }
}

func (h *Hub) fetchMatchUpdates() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Get all live matches
    matches, err := h.store.GetLiveMatches(ctx)
    if err != nil {
        h.logger.Error("Failed to fetch live matches", zap.Error(err))
        return
    }

    h.matchMu.Lock()
    defer h.matchMu.Unlock()

    for _, match := range matches {
        roomID := match.ID // Using match ID as room ID
        existingMatch, exists := h.matches[roomID]

        // Check if match needs update
        if !exists || matchNeedsUpdate(existingMatch, match) {
            h.matches[roomID] = match

            // Broadcast update
            updateMsg := &models.WSMessage{
                Type:      models.MessageTypeEvent,
                ChatRoom:  roomID,
                Match:     match,
                Timestamp: time.Now(),
            }

            h.broadcast <- updateMsg
        }
    }
}

func matchNeedsUpdate(old, new *models.Match) bool {
    if old.HomeScore != new.HomeScore || old.AwayScore != new.AwayScore {
        return true
    }
    if old.Status != new.Status {
        return true
    }
    return false
}

// Client write pump
func (c *Client) writePump() {
    ticker := time.NewTicker(54 * time.Second)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }

            w.Write(message)

            // Add queued chat messages to the current websocket message
            n := len(c.send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.send)
            }

            if err := w.Close(); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

// Client read pump
func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                c.hub.logger.Error("Websocket read error",
                    zap.Error(err),
                    zap.String("user_id", c.user.ID))
            }
            break
        }

        var wsMessage models.WSMessage
        if err := json.Unmarshal(message, &wsMessage); err != nil {
            c.hub.logger.Error("Failed to unmarshal message",
                zap.Error(err),
                zap.String("user_id", c.user.ID))
            continue
        }

        // Rate limit check
        if !c.limiter.Allow() {
            errorMsg := &models.WSMessage{
                Type:    models.MessageTypeError,
                Content: "Rate limit exceeded",
            }
            if payload, err := json.Marshal(errorMsg); err == nil {
                c.send <- payload
            }
            continue
        }

        // Add user and timestamp to message
        wsMessage.User = c.user
        wsMessage.Timestamp = time.Now()

        // Validate room membership
        if !c.canAccessRoom(wsMessage.ChatRoom) {
            errorMsg := &models.WSMessage{
                Type:    models.MessageTypeError,
                Content: "Room access denied",
            }
            if payload, err := json.Marshal(errorMsg); err == nil {
                c.send <- payload
            }
            continue
        }

        c.hub.broadcast <- &wsMessage
    }
}

func (c *Client) canAccessRoom(room string) bool {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.rooms[room]
}

const (
    writeWait = 10 * time.Second
    pongWait = 60 * time.Second
    pingPeriod = (pongWait * 9) / 10
    maxMessageSize = 4096
)