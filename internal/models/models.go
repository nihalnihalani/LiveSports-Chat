package models

import (
    "encoding/json"
    "time"
)

type User struct {
    ID           string    `json:"id" db:"id"`
    Username     string    `json:"username" db:"username"`
    Password     string    `json:"-" db:"password_hash"`
    Email        string    `json:"email" db:"email"`
    FavoriteTeam string    `json:"favorite_team" db:"favorite_team"`
    AvatarURL    string    `json:"avatar_url" db:"avatar_url"`
    IsAdmin      bool      `json:"is_admin" db:"is_admin"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Sport struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Description string    `json:"description" db:"description"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Team struct {
    ID        string    `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    SportID   string    `json:"sport_id" db:"sport_id"`
    LogoURL   string    `json:"logo_url" db:"logo_url"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Match struct {
    ID          string          `json:"id" db:"id"`
    SportID     string          `json:"sport_id" db:"sport_id"`
    HomeTeamID  string          `json:"home_team_id" db:"home_team_id"`
    AwayTeamID  string          `json:"away_team_id" db:"away_team_id"`
    StartTime   time.Time       `json:"start_time" db:"start_time"`
    Status      string          `json:"status" db:"status"`
    HomeScore   int            `json:"home_score" db:"home_score"`
    AwayScore   int            `json:"away_score" db:"away_score"`
    MatchData   json.RawMessage `json:"match_data" db:"match_data"`
    CreatedAt   time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`

    // Joined fields
    HomeTeam    *Team           `json:"home_team,omitempty" db:"-"`
    AwayTeam    *Team           `json:"away_team,omitempty" db:"-"`
    Sport       *Sport          `json:"sport,omitempty" db:"-"`
    Events      []*MatchEvent   `json:"events,omitempty" db:"-"`
}

type ChatRoom struct {
    ID          string    `json:"id" db:"id"`
    MatchID     string    `json:"match_id" db:"match_id"`
    Name        string    `json:"name" db:"name"`
    Description string    `json:"description" db:"description"`
    IsActive    bool      `json:"is_active" db:"is_active"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

    // Joined fields
    Match       *Match     `json:"match,omitempty" db:"-"`
    UserCount   int        `json:"user_count,omitempty" db:"-"`
}

type Message struct {
    ID          string    `json:"id" db:"id"`
    ChatRoomID  string    `json:"chat_room_id" db:"chat_room_id"`
    UserID      string    `json:"user_id" db:"user_id"`
    Content     string    `json:"content" db:"content"`
    MessageType string    `json:"message_type" db:"message_type"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`

    // Joined fields
    User        *User      `json:"user,omitempty" db:"-"`
}

type MatchEvent struct {
    ID          string    `json:"id" db:"id"`
    MatchID     string    `json:"match_id" db:"match_id"`
    EventType   string    `json:"event_type" db:"event_type"`
    EventTime   int       `json:"event_time" db:"event_time"`
    Description string    `json:"description" db:"description"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type UserChatRoom struct {
    UserID      string    `json:"user_id" db:"user_id"`
    ChatRoomID  string    `json:"chat_room_id" db:"chat_room_id"`
    LastReadAt  time.Time `json:"last_read_at" db:"last_read_at"`
}

// WebSocket message types
const (
    MessageTypeChat     = "chat"
    MessageTypeJoin     = "join"
    MessageTypeLeave    = "leave"
    MessageTypeTyping   = "typing"
    MessageTypeEvent    = "event"
    MessageTypeError    = "error"
)

// Match statuses
const (
    MatchStatusScheduled = "SCHEDULED"
    MatchStatusLive      = "LIVE"
    MatchStatusFinished  = "FINISHED"
    MatchStatusCancelled = "CANCELLED"
)

// Event types
const (
    EventTypeGoal         = "GOAL"
    EventTypeYellowCard   = "YELLOW_CARD"
    EventTypeRedCard      = "RED_CARD"
    EventTypeSubstitution = "SUBSTITUTION"
    EventTypePenalty      = "PENALTY"
    EventTypeKickoff      = "KICKOFF"
    EventTypeHalftime     = "HALFTIME"
    EventTypeFulltime     = "FULLTIME"
)

// WebSocket message struct
type WSMessage struct {
    Type      string          `json:"type"`
    ChatRoom  string          `json:"chat_room,omitempty"`
    Content   string          `json:"content,omitempty"`
    User      *User           `json:"user,omitempty"`
    Match     *Match          `json:"match,omitempty"`
    Event     *MatchEvent     `json:"event,omitempty"`
    Timestamp time.Time       `json:"timestamp"`
    Error     string          `json:"error,omitempty"`
    Data      json.RawMessage `json:"data,omitempty"`
}