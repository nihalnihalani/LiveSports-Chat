package store

import (
    "context"
    "time"

    "github.com/yourusername/sports-chat/internal/models"
)

type Store interface {
    // User operations
    CreateUser(ctx context.Context, user *models.User) error
    GetUser(ctx context.Context, id string) (*models.User, error)
    GetUserByUsername(ctx context.Context, username string) (*models.User, error)
    UpdateUser(ctx context.Context, user *models.User) error
    DeleteUser(ctx context.Context, id string) error

    // Sport operations
    CreateSport(ctx context.Context, sport *models.Sport) error
    GetSport(ctx context.Context, id string) (*models.Sport, error)
    ListSports(ctx context.Context) ([]*models.Sport, error)
    UpdateSport(ctx context.Context, sport *models.Sport) error
    DeleteSport(ctx context.Context, id string) error

    // Team operations
    CreateTeam(ctx context.Context, team *models.Team) error
    GetTeam(ctx context.Context, id string) (*models.Team, error)
    ListTeams(ctx context.Context, sportID string) ([]*models.Team, error)
    UpdateTeam(ctx context.Context, team *models.Team) error
    DeleteTeam(ctx context.Context, id string) error

    // Match operations
    CreateMatch(ctx context.Context, match *models.Match) error
    GetMatch(ctx context.Context, id string) (*models.Match, error)
    GetLiveMatches(ctx context.Context) ([]*models.Match, error)
    GetMatchesByStatus(ctx context.Context, status string) ([]*models.Match, error)
    GetUpcomingMatches(ctx context.Context, limit int) ([]*models.Match, error)
    UpdateMatch(ctx context.Context, match *models.Match) error
    DeleteMatch(ctx context.Context, id string) error

    // Chat room operations
    CreateChatRoom(ctx context.Context, room *models.ChatRoom) error
    GetChatRoom(ctx context.Context, id string) (*models.ChatRoom, error)
    GetMatchChatRoom(ctx context.Context, matchID string) (*models.ChatRoom, error)
    ListChatRooms(ctx context.Context) ([]*models.ChatRoom, error)
    UpdateChatRoom(ctx context.Context, room *models.ChatRoom) error
    DeleteChatRoom(ctx context.Context, id string) error

    // Message operations
    CreateMessage(ctx context.Context, message *models.Message) error
    GetMessage(ctx context.Context, id string) (*models.Message, error)
    GetRecentMessages(ctx context.Context, roomID string, limit int) ([]*models.Message, error)
    GetMessagesBefore(ctx context.Context, roomID string, before time.Time, limit int) ([]*models.Message, error)
    DeleteMessage(ctx context.Context, id string) error

    // Match event operations
    CreateMatchEvent(ctx context.Context, event *models.MatchEvent) error
    GetMatchEvents(ctx context.Context, matchID string) ([]*models.MatchEvent, error)
    GetRecentMatchEvents(ctx context.Context, matchID string, limit int) ([]*models.MatchEvent, error)

    // User presence operations
    JoinChatRoom(ctx context.Context, userID, roomID string) error
    LeaveChatRoom(ctx context.Context, userID, roomID string) error
    GetRoomUsers(ctx context.Context, roomID string) ([]*models.User, error)
    GetUserRooms(ctx context.Context, userID string) ([]*models.ChatRoom, error)

    // Search operations
    SearchMessages(ctx context.Context, query string, limit int) ([]*models.Message, error)
    SearchMatchEvents(ctx context.Context, query string, limit int) ([]*models.MatchEvent, error)

    // Statistics operations
    GetRoomStatistics(ctx context.Context, roomID string) (*RoomStatistics, error)
    GetUserStatistics(ctx context.Context, userID string) (*UserStatistics, error)
    GetMatchStatistics(ctx context.Context, matchID string) (*MatchStatistics, error)

    // Utility
    Close() error
}

type RoomStatistics struct {
    MessageCount  int       `json:"message_count"`
    UserCount     int       `json:"user_count"`
    LastActivity  time.Time `json:"last_activity"`
}

type UserStatistics struct {
    MessageCount      int       `json:"message_count"`
    RoomsJoined       int       `json:"rooms_joined"`
    LastActive        time.Time `json:"last_active"`
    FavoriteRooms     []string  `json:"favorite_rooms"`
}

type MatchStatistics struct {
    ViewerCount       int       `json:"viewer_count"`
    MessageCount      int       `json:"message_count"`
    PeakViewerCount   int       `json:"peak_viewer_count"`
    EventCount        int       `json:"event_count"`
}