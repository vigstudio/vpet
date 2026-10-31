package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shourenso/vpet/internal/game"
	"github.com/shourenso/vpet/internal/store"
	"golang.org/x/crypto/argon2"
)

const (
	sessionCookie = "vpet_session"
	csrfCookie    = "vpet_csrf"
	maxAuthBody   = 16 << 10
	maxChatBody   = 2 << 10
	maxGameBody   = 8 << 10
	sessionTTL    = 7 * 24 * time.Hour
)

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{2,31}$`)
	chatURLPattern  = regexp.MustCompile(`(?i)(https?://|www\.)`)
)

type Server struct {
	Store          *store.Postgres
	Game           *game.Engine
	PublicURL      *url.URL
	TransferSecret []byte
	SecureCookies  bool
	limiter        *limiter
	chat           *chatHub
}

type contextKey struct{}

type principal struct {
	Account *store.Account
	Token   []byte
	CSRF    string
}

type chatBody struct {
	Body      string `json:"body"`
	MessageID int64  `json:"message_id"`
}

type authBody struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Locale      string `json:"locale"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

func New(db *store.Postgres, engine *game.Engine, publicURL string, transferSecret []byte, secure bool) (*Server, error) {
	u, err := url.Parse(publicURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, errors.New("PUBLIC_URL must be an absolute URL")
	}
	if u.Scheme == "https" && !secure {
		return nil, errors.New("HTTPS PUBLIC_URL requires secure cookies")
	}
	return &Server{Store: db, Game: engine, PublicURL: u, TransferSecret: transferSecret, SecureCookies: secure, limiter: newLimiter(), chat: newChatHub()}, nil
}

func (s *Server) Handler(static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]bool{"ok": true}) })
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/claim", s.claim)
	mux.Handle("GET /api/auth/me", s.auth(http.HandlerFunc(s.me)))
	mux.Handle("POST /api/auth/logout", s.auth(s.csrf(http.HandlerFunc(s.logout))))
	mux.Handle("POST /api/auth/password", s.auth(s.csrf(http.HandlerFunc(s.changePassword))))
	mux.Handle("POST /api/auth/profile", s.auth(s.csrf(http.HandlerFunc(s.updateProfile))))
	mux.Handle("GET /api/v1/users/search", s.auth(http.HandlerFunc(s.searchUsers)))
	mux.Handle("GET /api/v1/world/messages", s.auth(http.HandlerFunc(s.worldMessages)))
	mux.Handle("POST /api/v1/world/messages", s.auth(s.csrf(http.HandlerFunc(s.sendWorldMessage))))
	mux.Handle("POST /api/v1/world/report", s.auth(s.csrf(http.HandlerFunc(s.reportWorldMessage))))
	mux.Handle("GET /api/v1/notifications/stream", s.auth(http.HandlerFunc(s.notifications)))
	mux.Handle("/api/v1/", s.auth(s.csrf(http.HandlerFunc(s.game))))
	mux.Handle("/", static)
	return s.securityHeaders(mux)
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "chưa đăng nhập")
			return
		}
		token, err := base64.RawURLEncoding.DecodeString(cookie.Value)
		if err != nil || len(token) != 32 {
			writeError(w, http.StatusUnauthorized, "phiên đăng nhập không hợp lệ")
			return
		}
		sess, err := s.Store.Session(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "phiên đăng nhập đã hết hạn")
			return
		}
		account, err := s.Store.AccountByID(r.Context(), sess.AccountID)
		if err != nil || account.Disabled {
			writeError(w, http.StatusUnauthorized, "tài khoản không hoạt động")
			return
		}
		ctx := context.WithValue(r.Context(), contextKey{}, &principal{Account: account, Token: token, CSRF: sess.CSRF})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		p := current(r)
		if subtleEqual(r.Header.Get("X-CSRF-Token"), p.CSRF) == false || !s.sameOrigin(r) {
			writeError(w, http.StatusForbidden, "yêu cầu bảo mật không hợp lệ")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Scheme, s.PublicURL.Scheme) && strings.EqualFold(u.Host, s.PublicURL.Host) {
		return true
	}
	return strings.EqualFold(u.Host, r.Host) && s.PublicURL.Hostname() == "localhost" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var b authBody
	if !s.decode(w, r, &b) || !s.limit(w, r, "register:"+store.NormalizeUsername(b.Username)) {
		return
	}
	if !usernamePattern.MatchString(b.Username) || len([]rune(b.DisplayName)) > 60 {
		writeError(w, 400, "tên đăng nhập hoặc tên hiển thị không hợp lệ")
		return
	}
	hash, err := hashPassword(b.Password)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	id, err := randomHex(16)
	if err != nil {
		writeError(w, 500, "không tạo được tài khoản")
		return
	}
	account, err := s.Store.CreateAccount(r.Context(), id, b.Username, b.DisplayName, b.Locale, hash)
	if err != nil {
		writeError(w, http.StatusConflict, "tên đăng nhập đã được dùng")
		return
	}
	s.issueSession(w, r, account)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var b authBody
	if !s.decode(w, r, &b) || !s.limit(w, r, "login:"+store.NormalizeUsername(b.Username)) {
		return
	}
	account, err := s.Store.AccountByUsername(r.Context(), b.Username)
	if err != nil || account.Disabled || !verifyPassword(account.PasswordHash, b.Password) {
		writeError(w, http.StatusUnauthorized, "tên đăng nhập hoặc mật khẩu không đúng")
		return
	}
	s.issueSession(w, r, account)
}

func (s *Server) claim(w http.ResponseWriter, r *http.Request) {
	var b authBody
	if !s.decode(w, r, &b) || !s.limit(w, r, "claim:"+store.NormalizeUsername(b.Username)) {
		return
	}
	batch, userID, err := verifyClaimCode(b.Code, s.TransferSecret)
	if err != nil || !usernamePattern.MatchString(b.Username) {
		writeError(w, 400, "mã chuyển hoặc thông tin tài khoản không hợp lệ")
		return
	}
	hash, err := hashPassword(b.Password)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	account, err := s.Store.RedeemClaim(r.Context(), batch, userID, b.Username, strings.TrimSpace(b.DisplayName), locale(b.Locale), hash)
	if err != nil {
		writeError(w, http.StatusConflict, "mã chuyển hoặc thông tin tài khoản không hợp lệ")
		return
	}
	s.issueSession(w, r, account)
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, account *store.Account) {
	token := make([]byte, 32)
	csrf := make([]byte, 24)
	if _, err := rand.Read(token); err != nil {
		writeError(w, 500, "không tạo được phiên")
		return
	}
	if _, err := rand.Read(csrf); err != nil {
		writeError(w, 500, "không tạo được phiên")
		return
	}
	csrfText := base64.RawURLEncoding.EncodeToString(csrf)
	if err := s.Store.CreateSession(r.Context(), token, account.ID, csrfText, time.Now().Add(sessionTTL)); err != nil {
		writeError(w, 500, "không tạo được phiên")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: base64.RawURLEncoding.EncodeToString(token), Path: "/", HttpOnly: true, Secure: s.SecureCookies, SameSite: http.SameSiteLaxMode, MaxAge: int(sessionTTL.Seconds())})
	http.SetCookie(w, &http.Cookie{Name: csrfCookie, Value: csrfText, Path: "/", HttpOnly: false, Secure: s.SecureCookies, SameSite: http.SameSiteLaxMode, MaxAge: int(sessionTTL.Seconds())})
	writeJSON(w, 200, publicAccount(account))
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, publicAccount(current(r).Account))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	_ = s.Store.RevokeSession(r.Context(), current(r).Token)
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.SecureCookies, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: csrfCookie, Path: "/", MaxAge: -1, Secure: s.SecureCookies, SameSite: http.SameSiteLaxMode})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var b authBody
	if !s.decode(w, r, &b) {
		return
	}
	hash, err := hashPassword(b.NewPassword)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err = s.Store.ChangePassword(r.Context(), current(r).Account.ID, hash); err != nil {
		writeError(w, 500, "không đổi được mật khẩu")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	var b authBody
	if !s.decode(w, r, &b) {
		return
	}
	name := strings.TrimSpace(b.DisplayName)
	if len([]rune(name)) > 60 || strings.ContainsAny(name, "\n\r\t") {
		writeError(w, 400, "tên hiển thị tối đa 60 ký tự và không xuống dòng")
		return
	}
	account, err := s.Store.UpdateProfile(r.Context(), current(r).Account.ID, name, locale(b.Locale))
	if err != nil {
		writeError(w, 500, "không lưu được hồ sơ")
		return
	}
	writeJSON(w, 200, publicAccount(account))
}

func (s *Server) searchUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Store.SearchUsers(r.Context(), r.URL.Query().Get("q"), current(r).Account.ID)
	if err != nil {
		writeError(w, 500, "không tìm được người chơi")
		return
	}
	writeJSON(w, 200, users)
}

func (s *Server) worldMessages(w http.ResponseWriter, r *http.Request) {
	messages, err := s.Store.WorldMessages(r.Context(), 100)
	if err != nil {
		writeError(w, 500, "không tải được trò chuyện")
		return
	}
	writeJSON(w, 200, messages)
}

func (s *Server) sendWorldMessage(w http.ResponseWriter, r *http.Request) {
	account := current(r).Account
	if !s.limit(w, r, "world-chat:"+account.ID) {
		return
	}
	var body chatBody
	if !s.decodeLimit(w, r, &body, maxChatBody) {
		return
	}
	text, ok := cleanChatMessage(body.Body)
	if !ok {
		writeError(w, 400, "tin nhắn cần 1–200 ký tự và không chứa liên kết")
		return
	}
	message, err := s.Store.CreateWorldMessage(r.Context(), account.ID, text)
	if err != nil {
		writeError(w, 500, "không gửi được tin nhắn")
		return
	}
	s.chat.publish(*message)
	writeJSON(w, 200, message)
}

// cleanChatMessage trims a world-chat line and says whether it may be published. Links are
// refused outright: a shared square with no moderation queue is not a place to hand players a
// working URL, and the few lines of rules here are cheaper than cleaning up after one.
func cleanChatMessage(raw string) (string, bool) {
	text := strings.TrimSpace(raw)
	length := len([]rune(text))
	if length < 1 || length > 200 || strings.Count(text, "\n") > 2 || chatURLPattern.MatchString(text) {
		return "", false
	}
	return text, true
}

func (s *Server) reportWorldMessage(w http.ResponseWriter, r *http.Request) {
	var body chatBody
	if !s.decodeLimit(w, r, &body, maxChatBody) || body.MessageID < 1 {
		return
	}
	if err := s.Store.ReportWorldMessage(r.Context(), body.MessageID, current(r).Account.ID); err != nil {
		writeError(w, 400, "không thể báo cáo tin nhắn này")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func allianceRateKey(accountID, action string) string {
	switch action {
	case "create", "join", "invite", "leave", "kick", "promote", "demote", "transfer", "open", "disband", "target", "abandon", "garrison", "attack", "claim-season":
	default:
		action = "unknown"
	}
	return "alliance:" + accountID + ":" + action
}

func (s *Server) game(w http.ResponseWriter, r *http.Request) {
	account := current(r).Account
	if r.Method == http.MethodPost && strings.TrimPrefix(r.URL.Path, "/api/v1") == "/alliance" {
		var body struct {
			Action string `json:"action"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxGameBody)
		raw, err := io.ReadAll(r.Body)
		if err != nil || json.Unmarshal(raw, &body) != nil {
			writeError(w, http.StatusBadRequest, "dữ liệu không hợp lệ")
			return
		}
		r.Body = io.NopCloser(strings.NewReader(string(raw)))
		if !s.limit(w, r, allianceRateKey(account.ID, body.Action)) {
			return
		}
	}
	s.Game.ServeHTTP(w, r, account.ID)
}

func (s *Server) notifications(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "trình duyệt không hỗ trợ thông báo")
		return
	}
	after, _ := strconv.ParseInt(r.Header.Get("Last-Event-ID"), 10, 64)
	// The client says which chat line it already has, so a reconnect resumes rather than
	// skipping whatever was said while it was away.
	chatAfter, _ := strconv.ParseInt(r.URL.Query().Get("chat_after"), 10, 64)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	poll := time.NewTicker(2 * time.Second)
	keepalive := time.NewTicker(15 * time.Second)
	chat, unsubscribe := s.chat.subscribe()
	defer poll.Stop()
	defer keepalive.Stop()
	defer unsubscribe()
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	for {
		rows, err := s.Store.NotificationsAfter(r.Context(), current(r).Account.ID, after)
		if err != nil {
			return
		}
		for _, row := range rows {
			id := row["id"].(int64)
			raw, _ := json.Marshal(row)
			fmt.Fprintf(w, "id: %d\ndata: %s\n\n", id, raw)
			after = id
		}
		// The hub is only a shortcut for low latency; this replay is what guarantees delivery,
		// so a message dropped by a full buffer or sent between two connections still arrives.
		said, err := s.Store.WorldMessagesAfter(r.Context(), chatAfter, 100)
		if err != nil {
			return
		}
		for _, message := range said {
			raw, _ := json.Marshal(message)
			fmt.Fprintf(w, "event: world-chat\ndata: %s\n\n", raw)
			chatAfter = message.ID
		}
		flusher.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-chat:
			// Wake up early and let the replay above do the sending, so every line goes out
			// through one path and the cursor can never skip an id.
		case <-poll.C:
		case <-keepalive.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) decode(w http.ResponseWriter, r *http.Request, out any) bool {
	return s.decodeLimit(w, r, out, maxAuthBody)
}

func (s *Server) decodeLimit(w http.ResponseWriter, r *http.Request, out any, limit int64) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeError(w, 400, "dữ liệu không hợp lệ")
		return false
	}
	return true
}

func (s *Server) limit(w http.ResponseWriter, r *http.Request, key string) bool {
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}
	if !s.limiter.allow(ip+":"+key, time.Now()) {
		w.Header().Set("Retry-After", "60")
		writeError(w, http.StatusTooManyRequests, "thử lại sau một phút")
		return false
	}
	return true
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; connect-src 'self'; script-src 'self'")
		next.ServeHTTP(w, r)
	})
}

func hashPassword(password string) (string, error) {
	if len(password) < 10 || len(password) > 128 {
		return "", errors.New("mật khẩu cần 10–128 ký tự")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	var memory, iterations uint32
	var parallelism uint8
	if n, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil || n != 3 || fmt.Sprintf("m=%d,t=%d,p=%d", memory, iterations, parallelism) != parts[3] {
		return false
	}
	salt, err1 := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	expected, err2 := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err1 != nil || err2 != nil || len(salt) != 16 || len(expected) != 32 || memory < 8 || memory > 256*1024 || iterations < 1 || iterations > 10 || parallelism < 1 || parallelism > 8 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return hmac.Equal(actual, expected)
}

func verifyClaimCode(code string, secret []byte) (string, string, error) {
	if len(secret) < 32 {
		return "", "", errors.New("transfer secret missing")
	}
	parts := strings.Split(code, ".")
	if len(parts) != 4 || parts[0] != "v1" {
		return "", "", errors.New("bad code")
	}
	payload := strings.Join(parts[:3], ".")
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	got, err := hex.DecodeString(parts[3])
	if err != nil || !hmac.Equal(got, mac.Sum(nil)) {
		return "", "", errors.New("bad code")
	}
	batch, err1 := base64.RawURLEncoding.DecodeString(parts[1])
	user, err2 := base64.RawURLEncoding.DecodeString(parts[2])
	if err1 != nil || err2 != nil || len(batch) == 0 || len(user) == 0 {
		return "", "", errors.New("bad code")
	}
	return string(batch), string(user), nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func current(r *http.Request) *principal { return r.Context().Value(contextKey{}).(*principal) }
func publicAccount(a *store.Account) map[string]string {
	return map[string]string{"id": a.ID, "username": a.Username, "display_name": a.DisplayName, "locale": a.Locale}
}
func locale(v string) string {
	if v == "en" || v == "ja" {
		return v
	}
	return "vi"
}
func subtleEqual(a, b string) bool { return len(a) == len(b) && hmac.Equal([]byte(a), []byte(b)) }
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type limiter struct {
	mu        sync.Mutex
	values    map[string][]time.Time
	lastSweep time.Time
}

func newLimiter() *limiter { return &limiter{values: map[string][]time.Time{}} }
func (l *limiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-time.Minute)
	if l.lastSweep.IsZero() || now.Sub(l.lastSweep) >= time.Minute {
		for candidate, attempts := range l.values {
			if len(attempts) == 0 || !attempts[len(attempts)-1].After(cutoff) {
				delete(l.values, candidate)
			}
		}
		l.lastSweep = now
	}
	values := l.values[key][:0]
	for _, at := range l.values[key] {
		if at.After(cutoff) {
			values = append(values, at)
		}
	}
	if len(values) >= 8 {
		l.values[key] = values
		return false
	}
	l.values[key] = append(values, now)
	return true
}

type chatHub struct {
	mu   sync.Mutex
	next int
	subs map[int]chan store.WorldMessage
}

func newChatHub() *chatHub { return &chatHub{subs: make(map[int]chan store.WorldMessage)} }

func (h *chatHub) subscribe() (<-chan store.WorldMessage, func()) {
	h.mu.Lock()
	id := h.next
	h.next++
	ch := make(chan store.WorldMessage, 8)
	h.subs[id] = ch
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *chatHub) publish(message store.WorldMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.subs {
		select {
		case ch <- message:
		default:
		}
	}
}
