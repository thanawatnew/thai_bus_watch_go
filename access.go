package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const accessCookie = "buswatch_access"

type accessSession struct {
	rank     int
	lastSeen time.Time
}

type AccessGate struct {
	mu       sync.Mutex
	enabled  bool
	maxUsers int
	passes   map[string]int
	sessions map[string]accessSession
}

func NewAccessGate() *AccessGate {
	g := &AccessGate{passes: map[string]int{}, sessions: map[string]accessSession{}}
	g.enabled, _ = strconv.ParseBool(os.Getenv("PRIORITY_PASS_ENABLED"))
	g.maxUsers, _ = strconv.Atoi(os.Getenv("MAX_CONCURRENT_USERS"))
	if g.maxUsers < 1 {
		g.maxUsers = 10
	}
	if path := strings.TrimSpace(os.Getenv("PRIORITY_PASS_FILE")); path != "" {
		var passes []string
		if body, err := os.ReadFile(path); err == nil && json.Unmarshal(body, &passes) == nil {
			for i, pass := range passes {
				if pass = strings.TrimSpace(pass); pass != "" {
					g.passes[pass] = i + 1
				}
			}
		}
	}
	// Never lock everyone out because a pass file is missing or malformed.
	if len(g.passes) == 0 {
		g.enabled = false
	}
	return g
}

func (g *AccessGate) clean(now time.Time) {
	for token, session := range g.sessions {
		if now.Sub(session.lastSeen) > 2*time.Minute {
			delete(g.sessions, token)
		}
	}
}

func (g *AccessGate) session(r *http.Request, touch bool) (accessSession, bool) {
	cookie, err := r.Cookie(accessCookie)
	if err != nil {
		return accessSession{}, false
	}
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	g.clean(now)
	session, ok := g.sessions[cookie.Value]
	if ok && touch {
		session.lastSeen = now
		g.sessions[cookie.Value] = session
	}
	return session, ok
}

func (g *AccessGate) status(w http.ResponseWriter, r *http.Request) {
	session, authorized := g.session(r, true)
	g.mu.Lock()
	active := len(g.sessions)
	g.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": g.enabled, "authorized": !g.enabled || authorized,
		"active": active, "maxUsers": g.maxUsers, "rank": session.rank,
	})
}

func (g *AccessGate) enter(w http.ResponseWriter, r *http.Request) {
	if !g.enabled {
		g.status(w, r)
		return
	}
	var input struct {
		Pass string `json:"pass"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "priority pass is required"})
		return
	}
	rank, valid := g.passes[strings.TrimSpace(input.Pass)]
	if !valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid priority pass"})
		return
	}
	now := time.Now()
	g.mu.Lock()
	g.clean(now)
	if len(g.sessions) >= g.maxUsers {
		// A lower rank number has higher priority. When full, a higher-priority
		// pass can replace the currently lowest-priority session.
		worstToken, worstRank := "", 0
		for token, session := range g.sessions {
			if session.rank > worstRank {
				worstToken, worstRank = token, session.rank
			}
		}
		if worstToken == "" || rank >= worstRank {
			g.mu.Unlock()
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Thai Bus Watch is at its current user limit; a higher-priority place is required"})
			return
		}
		delete(g.sessions, worstToken)
	}
	var tokenBytes [24]byte
	_, _ = rand.Read(tokenBytes[:])
	token := hex.EncodeToString(tokenBytes[:])
	g.sessions[token] = accessSession{rank: rank, lastSeen: now}
	active := len(g.sessions)
	g.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: accessCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 7200})
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": true, "authorized": true, "active": active, "maxUsers": g.maxUsers, "rank": rank,
	})
}

func (g *AccessGate) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !g.enabled || !strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/api/access/") || r.URL.Path == "/api/telegram/webhook" {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := g.session(r, true); !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "priority pass required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
