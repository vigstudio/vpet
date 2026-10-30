package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shourenso/vpet/internal/game"
)

type Postgres struct {
	Pool *pgxpool.Pool
}

type Account struct {
	ID           string
	Username     string
	DisplayName  string
	Locale       string
	PasswordHash string
	Disabled     bool
}

type Session struct {
	AccountID string
	CSRF      string
	ExpiresAt time.Time
}

type WorldMessage struct {
	ID          int64          `json:"id"`
	AccountID   string         `json:"account_id"`
	Username    string         `json:"username"`
	DisplayName string         `json:"display_name"`
	Body        string         `json:"body"`
	PetParts    map[string]int `json:"pet_parts,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

func Open(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{Pool: pool}, nil
}

func (s *Postgres) Close() { s.Pool.Close() }

func (s *Postgres) Migrate(ctx context.Context, sql string) error {
	_, err := s.Pool.Exec(ctx, sql)
	return err
}

func (s *Postgres) RecoverMarketJournal(ctx context.Context) error {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('market_pending', 0))`); err != nil {
		return err
	}
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT pvalue FROM game_kv WHERE pkey='market_pending' FOR UPDATE`).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}
	var rows map[string]json.RawMessage
	if err = json.Unmarshal(raw, &rows); err != nil {
		return fmt.Errorf("invalid market_pending journal: %w", err)
	}
	keys := make([]string, 0, len(rows))
	for key := range rows {
		if key == "" || key == "market_pending" {
			return fmt.Errorf("invalid market_pending key %q", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := rows[key]
		if string(value) == "null" {
			if _, err = tx.Exec(ctx, `DELETE FROM game_kv WHERE pkey=$1`, key); err != nil {
				return err
			}
			continue
		}
		if !json.Valid(value) {
			return fmt.Errorf("invalid market_pending value for %q", key)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO game_kv(pkey,pvalue) VALUES($1,$2)
			ON CONFLICT(pkey) DO UPDATE SET pvalue=excluded.pvalue, version=game_kv.version+1, updated_at=now()`, key, value); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM game_kv WHERE pkey='market_pending'`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Postgres) KVGet(key string) ([]byte, error) {
	var raw []byte
	err := s.Pool.QueryRow(context.Background(), `SELECT pvalue FROM game_kv WHERE pkey=$1`, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return raw, err
}

func (s *Postgres) KVSet(key string, raw []byte) error {
	if !json.Valid(raw) {
		return errors.New("game value is not valid JSON")
	}
	_, err := s.Pool.Exec(context.Background(), `
		INSERT INTO game_kv(pkey,pvalue) VALUES($1,$2)
		ON CONFLICT(pkey) DO UPDATE SET pvalue=excluded.pvalue, version=game_kv.version+1, updated_at=now()`, key, raw)
	return err
}

func (s *Postgres) KVDelete(key string) error {
	_, err := s.Pool.Exec(context.Background(), `DELETE FROM game_kv WHERE pkey=$1`, key)
	return err
}

func (s *Postgres) KVList(page, perPage int) ([]string, error) {
	rows, err := s.Pool.Query(context.Background(), `SELECT pkey FROM game_kv ORDER BY pkey LIMIT $1 OFFSET $2`, perPage, page*perPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// Apply atomically writes every participant in a multiplayer action. Advisory locks cover
// missing rows too; sorted acquisition prevents deadlocks when two actions touch the same users.
func (s *Postgres) Apply(values map[string]any) error {
	ctx := context.Background()
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
			return err
		}
	}
	for _, key := range keys {
		value := values[key]
		if value == nil {
			if _, err = tx.Exec(ctx, `DELETE FROM game_kv WHERE pkey=$1`, key); err != nil {
				return err
			}
			continue
		}
		raw, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.Exec(ctx, `INSERT INTO game_kv(pkey,pvalue) VALUES($1,$2)
			ON CONFLICT(pkey) DO UPDATE SET pvalue=excluded.pvalue, version=game_kv.version+1, updated_at=now()`, key, raw); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Postgres) User(id string) (*game.User, error) {
	a, err := s.account(`SELECT id,username,display_name,locale,password_hash,disabled FROM accounts WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	return &game.User{ID: a.ID, Username: a.Username, DisplayName: a.DisplayName, Locale: a.Locale, Disabled: a.Disabled}, nil
}

func (s *Postgres) UserByUsername(username string) (*game.User, error) {
	a, err := s.account(`SELECT id,username,display_name,locale,password_hash,disabled FROM accounts WHERE username_normalized=$1`, NormalizeUsername(username))
	if err != nil {
		return nil, err
	}
	return &game.User{ID: a.ID, Username: a.Username, DisplayName: a.DisplayName, Locale: a.Locale, Disabled: a.Disabled}, nil
}

func (s *Postgres) AccountByUsername(ctx context.Context, username string) (*Account, error) {
	return s.accountContext(ctx, `SELECT id,username,display_name,locale,password_hash,disabled FROM accounts WHERE username_normalized=$1`, NormalizeUsername(username))
}

func (s *Postgres) AccountByID(ctx context.Context, id string) (*Account, error) {
	return s.accountContext(ctx, `SELECT id,username,display_name,locale,password_hash,disabled FROM accounts WHERE id=$1`, id)
}

func (s *Postgres) account(query string, arg any) (*Account, error) {
	return s.accountContext(context.Background(), query, arg)
}

func (s *Postgres) accountContext(ctx context.Context, query string, arg any) (*Account, error) {
	var a Account
	err := s.Pool.QueryRow(ctx, query, arg).Scan(&a.ID, &a.Username, &a.DisplayName, &a.Locale, &a.PasswordHash, &a.Disabled)
	return &a, err
}

func (s *Postgres) CreateAccount(ctx context.Context, id, username, displayName, locale, passwordHash string) (*Account, error) {
	username = strings.TrimSpace(username)
	if displayName == "" {
		displayName = username
	}
	if locale != "en" && locale != "ja" {
		locale = "vi"
	}
	result, err := s.Pool.Exec(ctx, `INSERT INTO accounts(id,username,username_normalized,display_name,locale,password_hash,claimed_at)
		SELECT $1,$2,$3,$4,$5,$6,now()
		WHERE NOT EXISTS (
			SELECT 1 FROM migration_claims c JOIN migration_batches b ON b.id=c.batch_id
			WHERE c.username_normalized=$3 AND c.redeemed_at IS NULL AND b.enabled=true AND (b.expires_at IS NULL OR b.expires_at>now())
		)`, id, username, NormalizeUsername(username), displayName, locale, passwordHash)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() != 1 {
		return nil, errors.New("username is reserved for migration")
	}
	return s.AccountByID(ctx, id)
}

func (s *Postgres) RedeemClaim(ctx context.Context, batch, userID, username, displayName, locale, passwordHash string) (*Account, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var expected string
	var redeemed *time.Time
	err = tx.QueryRow(ctx, `SELECT username_normalized,redeemed_at FROM migration_claims c JOIN migration_batches b ON b.id=c.batch_id
		WHERE c.batch_id=$1 AND c.mattermost_user_id=$2 AND b.enabled=true AND (b.expires_at IS NULL OR b.expires_at>now()) FOR UPDATE`, batch, userID).Scan(&expected, &redeemed)
	if err != nil || redeemed != nil || expected != NormalizeUsername(username) {
		return nil, errors.New("claim is invalid or already redeemed")
	}
	if displayName == "" {
		displayName = username
	}
	if _, err = tx.Exec(ctx, `INSERT INTO accounts(id,username,username_normalized,display_name,locale,password_hash,claimed_at)
		VALUES($1,$2,$3,$4,$5,$6,now())`, userID, username, expected, displayName, locale, passwordHash); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE migration_claims SET redeemed_account_id=$3,redeemed_at=now() WHERE batch_id=$1 AND mattermost_user_id=$2`, batch, userID, userID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.AccountByID(ctx, userID)
}

// UpdateProfile changes what a player shows other people. The username is deliberately not
// editable: it is the login, migration claims are keyed on username_normalized, and the
// market and arena print it, so renaming would break the trail back to a trade or a duel.
func (s *Postgres) UpdateProfile(ctx context.Context, id, displayName, locale string) (*Account, error) {
	displayName = strings.TrimSpace(displayName)
	if locale != "en" && locale != "ja" {
		locale = "vi"
	}
	if _, err := s.Pool.Exec(ctx, `UPDATE accounts SET
		display_name=COALESCE(NULLIF($2,''), username), locale=$3, updated_at=now() WHERE id=$1`, id, displayName, locale); err != nil {
		return nil, err
	}
	return s.AccountByID(ctx, id)
}

func (s *Postgres) ChangePassword(ctx context.Context, id, passwordHash string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE accounts SET password_hash=$2, updated_at=now() WHERE id=$1`, id, passwordHash)
	return err
}

func (s *Postgres) CreateSession(ctx context.Context, token []byte, accountID, csrf string, expires time.Time) error {
	hash := sha256.Sum256(token)
	_, err := s.Pool.Exec(ctx, `INSERT INTO sessions(token_hash,account_id,csrf_token,expires_at) VALUES($1,$2,$3,$4)`, hash[:], accountID, csrf, expires)
	return err
}

func (s *Postgres) Session(ctx context.Context, token []byte) (*Session, error) {
	hash := sha256.Sum256(token)
	var v Session
	err := s.Pool.QueryRow(ctx, `SELECT account_id,csrf_token,expires_at FROM sessions
		WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now()`, hash[:]).Scan(&v.AccountID, &v.CSRF, &v.ExpiresAt)
	return &v, err
}

func (s *Postgres) RevokeSession(ctx context.Context, token []byte) error {
	hash := sha256.Sum256(token)
	_, err := s.Pool.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE token_hash=$1`, hash[:])
	return err
}

func (s *Postgres) Notify(userID, reason, message, tab, duel string) {
	if _, err := s.Pool.Exec(context.Background(), `INSERT INTO notifications(account_id,reason,message,tab,duel) VALUES($1,$2,$3,$4,$5)`, userID, reason, message, tab, duel); err != nil {
		s.LogError("vpet: notification", "account", userID, "reason", reason, "err", err.Error())
	}
}

func (s *Postgres) LogError(message string, fields ...any) {
	fmt.Printf("%s %v\n", message, fields)
}

func (s *Postgres) NotificationsAfter(ctx context.Context, userID string, after int64) ([]map[string]any, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,reason,message,tab,duel,created_at FROM notifications WHERE account_id=$1 AND id>$2 ORDER BY id LIMIT 100`, userID, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var reason, message, tab, duel string
		var created time.Time
		if err := rows.Scan(&id, &reason, &message, &tab, &duel, &created); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "reason": reason, "message": message, "tab": tab, "duel": duel, "created_at": created})
	}
	return out, rows.Err()
}

// worldMessageQuery is shared by the backlog and the catch-up read so both return exactly the
// same shape, including the sender's pet.
const worldMessageQuery = `SELECT m.id,m.account_id,a.username,a.display_name,m.body,m.created_at,
		COALESCE((SELECT pvalue->'pets'->0->'parts' FROM game_kv WHERE pkey='u_'||m.account_id), 'null'::jsonb)
		FROM world_messages m JOIN accounts a ON a.id=m.account_id
		WHERE a.disabled=false`

func (s *Postgres) scanWorldMessages(rows pgx.Rows) ([]WorldMessage, error) {
	defer rows.Close()
	var messages []WorldMessage
	for rows.Next() {
		var message WorldMessage
		var parts []byte
		if err := rows.Scan(&message.ID, &message.AccountID, &message.Username, &message.DisplayName, &message.Body, &message.CreatedAt, &parts); err != nil {
			return nil, err
		}
		if len(parts) > 0 && string(parts) != "null" {
			_ = json.Unmarshal(parts, &message.PetParts)
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

// WorldMessagesAfter returns everything said since id, oldest first. The live hub can drop a
// message when a reader is slow or between two SSE connections, so the stream replays from
// this cursor on every tick: the database, not the hub, decides what a player has seen.
func (s *Postgres) WorldMessagesAfter(ctx context.Context, after int64, limit int) ([]WorldMessage, error) {
	rows, err := s.Pool.Query(ctx, worldMessageQuery+` AND m.id>$1 ORDER BY m.id LIMIT $2`, after, limit)
	if err != nil {
		return nil, err
	}
	return s.scanWorldMessages(rows)
}

func (s *Postgres) WorldMessages(ctx context.Context, limit int) ([]WorldMessage, error) {
	rows, err := s.Pool.Query(ctx, worldMessageQuery+` ORDER BY m.id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	messages, err := s.scanWorldMessages(rows)
	if err != nil {
		return nil, err
	}
	// The newest rows were cheapest to find, but the screen reads oldest first.
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, nil
}

func (s *Postgres) CreateWorldMessage(ctx context.Context, accountID, body string) (*WorldMessage, error) {
	var message WorldMessage
	var parts []byte
	err := s.Pool.QueryRow(ctx, `WITH inserted AS (
		INSERT INTO world_messages(account_id,body) VALUES($1,$2) RETURNING id,account_id,body,created_at
	) SELECT i.id,i.account_id,a.username,a.display_name,i.body,i.created_at,
		COALESCE((SELECT pvalue->'pets'->0->'parts' FROM game_kv WHERE pkey='u_'||i.account_id), 'null'::jsonb)
		FROM inserted i JOIN accounts a ON a.id=i.account_id`, accountID, body).
		Scan(&message.ID, &message.AccountID, &message.Username, &message.DisplayName, &message.Body, &message.CreatedAt, &parts)
	if err != nil {
		return nil, err
	}
	if len(parts) > 0 && string(parts) != "null" {
		_ = json.Unmarshal(parts, &message.PetParts)
	}
	return &message, nil
}

func (s *Postgres) ReportWorldMessage(ctx context.Context, messageID int64, reporterID string) error {
	result, err := s.Pool.Exec(ctx, `INSERT INTO world_message_reports(message_id,reporter_id)
		SELECT id,$2 FROM world_messages WHERE id=$1 AND account_id<>$2 ON CONFLICT DO NOTHING`, messageID, reporterID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("message not found or already reported")
	}
	return nil
}

func (s *Postgres) SearchUsers(ctx context.Context, term, except string) ([]game.User, error) {
	term = "%" + NormalizeUsername(term) + "%"
	rows, err := s.Pool.Query(ctx, `SELECT id,username,display_name,locale,disabled FROM accounts
		WHERE id<>$1 AND disabled=false AND (username_normalized LIKE $2 OR lower(display_name) LIKE $2)
		ORDER BY username LIMIT 8`, except, term)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []game.User
	for rows.Next() {
		var u game.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Locale, &u.Disabled); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func NormalizeUsername(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
