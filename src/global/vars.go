package global

import (
	"database/sql"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	goredis "github.com/redis/go-redis/v9"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// WaapiBase es la URL base de la API de WAAPI.
// WaapiToken se lee desde la variable de entorno WAAPI_TOKEN.
var (
	WaapiBase  = "https://waapi.app/api/v1/instances"
	WaapiToken = getEnvOrDefault("WAAPI_TOKEN", "")
	IaURL      = "https://apifreellm.com/api/v1/chat"
	IaKey      = getEnvOrDefault("LEGACY_API_KEY", "")
)

// getEnvOrDefault retorna el valor de la variable de entorno o un valor por defecto.
func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

var (
	Containers   = make(map[int]*sqlstore.Container)
	ContainersMu sync.Mutex
	ActiveBots   = make(map[int]bool)
	ActiveMu     sync.Mutex
)

const (
	ADMIN_USERNAME = "admin"
	ADMIN_EMAIL    = "admin@example.com"
	ADMIN_PHONE    = "+1234567890"
	ADMIN_PASS     = "admin123"
)

const (
	MAX_BOTS              = 50
	MAX_CONNECT_RETRIES   = 5
	SUBSCRIPTION_DURATION = 7 * 24 * time.Hour
	// MAX_HISTORY: número máximo de turnos (user+assistant) a recuperar de DB/Redis
	MAX_HISTORY = 8
	// MAX_HISTORY_CHARS: límite de caracteres totales del historial en el prompt
	// (~2000 chars ≈ ~500 tokens; ajustable según el modelo usado)
	MAX_HISTORY_CHARS     = 2000
	SESSION_EXPIRATION    = 1 * time.Hour
	RATE_LIMIT_PER_MINUTE = 10
	MAX_PROMPT_LENGTH     = 2000 // reducido de 5000 → ahorra tokens del sistema
	// MAX_MSG_LENGTH: longitud máxima de un mensaje de usuario que procesamos
	MAX_MSG_LENGTH = 500
	// AI_TIMEOUT_TOTAL: tiempo máximo que esperamos a la IA antes de dar error
	AI_TIMEOUT_TOTAL = 40 * time.Second
	// DEDUP_WINDOW: ventana de tiempo para ignorar mensajes duplicados por JID
	DEDUP_WINDOW = 3 * time.Second
)

var (
	ConfigDB      *sql.DB
	RedisClient   *goredis.Client
	SessionMW     fiber.Handler
	EncryptionKey []byte
)

// ============================================================
// Cache en memoria de prompts del sistema (Mejora 1)
// Evita una query a DB en cada mensaje recibido.
// ============================================================

type promptCache struct {
	mu      sync.RWMutex
	entries map[int]promptEntry
}

type promptEntry struct {
	value     string
	expiresAt time.Time
}

var PromptCache = &promptCache{
	entries: make(map[int]promptEntry),
}

// Get retorna el prompt cacheado si no ha expirado.
func (c *promptCache) Get(botID int) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[botID]
	if !ok || time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.value, true
}

// Set guarda el prompt con TTL de 5 minutos.
func (c *promptCache) Set(botID int, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[botID] = promptEntry{
		value:     value,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
}

// Invalidate elimina el prompt del cache (llamar al actualizar el prompt).
func (c *promptCache) Invalidate(botID int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, botID)
}

// ============================================================
// Semáforo por usuario — evita respuestas dobles (Mejora 6)
// ============================================================

type userSemaphore struct {
	mu      sync.Mutex
	active  map[string]bool
}

var UserSem = &userSemaphore{
	active: make(map[string]bool),
}

// TryLock retorna true y adquiere el lock si el usuario no está siendo procesado.
func (s *userSemaphore) TryLock(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active[key] {
		return false
	}
	s.active[key] = true
	return true
}

// Unlock libera el lock del usuario.
func (s *userSemaphore) Unlock(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.active, key)
}

// ============================================================
// Deduplicador de mensajes por ID (Mejora 2)
// ============================================================

type msgDedup struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

var MsgDedup = &msgDedup{
	seen: make(map[string]time.Time),
}

// IsDuplicate retorna true si el messageID ya fue procesado recientemente.
func (d *msgDedup) IsDuplicate(messageID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	// Limpiar entradas viejas
	now := time.Now()
	for id, t := range d.seen {
		if now.Sub(t) > DEDUP_WINDOW {
			delete(d.seen, id)
		}
	}
	if _, exists := d.seen[messageID]; exists {
		return true
	}
	d.seen[messageID] = now
	return false
}
