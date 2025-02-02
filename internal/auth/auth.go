package auth

import (
    "context"
    "crypto/subtle"
    "errors"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v4"
    "github.com/google/uuid"
    "golang.org/x/crypto/argon2"
    "go.uber.org/zap"

    "github.com/yourusername/sports-chat/internal/models"
)

var (
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrTokenExpired      = errors.New("token expired")
    ErrInvalidToken      = errors.New("invalid token")
    ErrUserBlocked       = errors.New("user is blocked")
    ErrTooManyAttempts   = errors.New("too many login attempts")
)

type Service struct {
    jwtSecret    []byte
    logger       *zap.Logger
    argon2Params *Argon2Params
}

type Argon2Params struct {
    memory      uint32
    iterations  uint32
    parallelism uint8
    saltLength  uint32
    keyLength   uint32
}

type Claims struct {
    jwt.RegisteredClaims
    UserID      string   `json:"uid"`
    Username    string   `json:"username"`
    IsAdmin     bool     `json:"is_admin"`
    SessionID   string   `json:"sid"`
}

type TokenPair struct {
    AccessToken   string    `json:"access_token"`
    RefreshToken  string    `json:"refresh_token"`
    ExpiresAt     time.Time `json:"expires_at"`
}

func NewService(jwtSecret string, logger *zap.Logger) *Service {
    return &Service{
        jwtSecret: []byte(jwtSecret),
        logger:    logger,
        argon2Params: &Argon2Params{
            memory:      64 * 1024,
            iterations:  3,
            parallelism: 2,
            saltLength:  16,
            keyLength:   32,
        },
    }
}

func (s *Service) GenerateTokenPair(user *models.User) (*TokenPair, error) {
    sessionID := uuid.New().String()

    // Generate access token
    accessToken, accessExpiresAt, err := s.generateAccessToken(user, sessionID)
    if err != nil {
        return nil, fmt.Errorf("failed to generate access token: %w", err)
    }

    // Generate refresh token
    refreshToken, err := s.generateRefreshToken(user.ID, sessionID)
    if err != nil {
        return nil, fmt.Errorf("failed to generate refresh token: %w", err)
    }

    return &TokenPair{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresAt:    accessExpiresAt,
    }, nil
}

func (s *Service) generateAccessToken(user *models.User, sessionID string) (string, time.Time, error) {
    expiresAt := time.Now().Add(15 * time.Minute)
    
    claims := Claims{
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expiresAt),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            ID:        uuid.New().String(),
        },
        UserID:    user.ID,
        Username:  user.Username,
        IsAdmin:   user.IsAdmin,
        SessionID: sessionID,
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
    signedToken, err := token.SignedString(s.jwtSecret)
    if err != nil {
        return "", time.Time{}, err
    }

    return signedToken, expiresAt, nil
}

func (s *Service) generateRefreshToken(userID, sessionID string) (string, error) {
    return uuid.NewString(), nil
}

func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.jwtSecret, nil
    })

    if err != nil {
        if errors.Is(err, jwt.ErrTokenExpired) {
            return nil, ErrTokenExpired
        }
        return nil, ErrInvalidToken
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }

    return claims, nil
}

func (s *Service) HashPassword(password string) (string, error) {
    salt := make([]byte, s.argon2Params.saltLength)
    if _, err := uuid.New().SetVersion(4).SetVariant(uuid.VariantRFC4122); err != nil {
        return "", fmt.Errorf("failed to generate salt: %w", err)
    }

    hash := argon2.IDKey(
        []byte(password),
        salt,
        s.argon2Params.iterations,
        s.argon2Params.memory,
        s.argon2Params.parallelism,
        s.argon2Params.keyLength,
    )

    // Format: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
    return fmt.Sprintf(
        "$argon2id$v=19$m=%d,t=%d,p=%d$%x$%x",
        s.argon2Params.memory,
        s.argon2Params.iterations,
        s.argon2Params.parallelism,
        salt,
        hash,
    ), nil
}

func (s *Service) VerifyPassword(hashedPassword, password string) (bool, error) {
    params, salt, hash, err := s.decodeHash(hashedPassword)
    if err != nil {
        return false, err
    }

    otherHash := argon2.IDKey(
        []byte(password),
        salt,
        params.iterations,
        params.memory,
        params.parallelism,
        params.keyLength,
    )

    return subtle.ConstantTimeCompare(hash, otherHash) == 1, nil
}

func (s *Service) decodeHash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
    vals := strings.Split(encodedHash, "$")
    if len(vals) != 5 {
        return nil, nil, nil, errors.New("invalid hash format")
    }

    var version int
    if _, err := fmt.Sscanf(vals[2], "v=%d", &version); err != nil {
        return nil, nil, nil, err
    }

    p := &Argon2Params{}
    if _, err := fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d",
        &p.memory, &p.iterations, &p.parallelism); err != nil {
        return nil, nil, nil, err
    }

    salt, err := hex.DecodeString(vals[3])
    if err != nil {
        return nil, nil, nil, err
    }
    p.saltLength = uint32(len(salt))

    hash, err := hex.DecodeString(vals[4])
    if err != nil {
        return nil, nil, nil, err
    }
    p.keyLength = uint32(len(hash))

    return p, salt, hash, nil
}

// Middleware for protecting routes
func (s *Service) AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Remove "Bearer " prefix if present
        token = strings.TrimPrefix(token, "Bearer ")

        claims, err := s.ValidateAccessToken(token)
        if err != nil {
            if err == ErrTokenExpired {
                http.Error(w, "Token expired", http.StatusUnauthorized)
            } else {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
            }
            return
        }

        // Add claims to request context
        ctx := context.WithValue(r.Context(), "claims", claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Middleware for admin-only routes
func (s *Service) AdminMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims, ok := r.Context().Value("claims").(*Claims)
        if !ok || !claims.IsAdmin {
            http.Error(w, "Admin access required", http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Helper functions for WebSocket authentication
func (s *Service) AuthenticateWebSocket(token string) (*models.User, error) {
    claims, err := s.ValidateAccessToken(token)
    if err != nil {
        return nil, err
    }

    // Create a basic user object from claims
    // In a real application, you might want to fetch the full user from the database
    user := &models.User{
        ID:       claims.UserID,
        Username: claims.Username,
        IsAdmin:  claims.IsAdmin,
    }

    return user, nil
}

// RefreshToken handles token refresh requests
func (s *Service) RefreshToken(refreshToken string) (*TokenPair, error) {
    // Validate refresh token
    // In a real application, you would verify this against stored refresh tokens
    userID := "user-id" // This would come from your refresh token validation
    
    // Create a basic user object
    user := &models.User{
        ID:       userID,
        Username: "username",
        IsAdmin:  false,
    }

    // Generate new token pair
    return s.GenerateTokenPair(user)
}

// Track failed login attempts
type LoginAttempt struct {
    Count     int
    LastTry   time.Time
    BlockedAt *time.Time
}

var loginAttempts = make(map[string]*LoginAttempt)
var loginMutex sync.RWMutex

func (s *Service) TrackLoginAttempt(identifier string, success bool) error {
    loginMutex.Lock()
    defer loginMutex.Unlock()

    attempt, exists := loginAttempts[identifier]
    if !exists {
        attempt = &LoginAttempt{}
        loginAttempts[identifier] = attempt
    }

    if success {
        delete(loginAttempts, identifier)
        return nil
    }

    attempt.Count++
    attempt.LastTry = time.Now()

    if attempt.Count >= 5 {
        now := time.Now()
        attempt.BlockedAt = &now
        return ErrTooManyAttempts
    }

    return nil
}

func (s *Service) IsBlocked(identifier string) bool {
    loginMutex.RLock()
    defer loginMutex.RUnlock()

    attempt, exists := loginAttempts[identifier]
    if !exists {
        return false
    }

    if attempt.BlockedAt == nil {
        return false
    }

    // Block for 15 minutes
    return time.Since(*attempt.BlockedAt) < 15*time.Minute
}