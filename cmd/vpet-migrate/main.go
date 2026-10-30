package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shourenso/vpet/internal/store"
)

type Snapshot struct {
	Schema     int                        `json:"schema"`
	Batch      string                     `json:"batch"`
	ExportedAt time.Time                  `json:"exported_at"`
	Users      []SnapshotUser             `json:"users"`
	KV         map[string]json.RawMessage `json:"kv"`
	Checksums  map[string]string          `json:"checksums"`
}

type SnapshotUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Locale      string `json:"locale"`
	Disabled    bool   `json:"disabled"`
}

func main() {
	var databaseURL, file, batch string
	flag.StringVar(&databaseURL, "database", os.Getenv("DATABASE_URL"), "standalone PostgreSQL URL")
	flag.StringVar(&file, "file", "", "snapshot JSON file")
	flag.StringVar(&batch, "batch", "", "migration batch id")
	flag.Parse()
	if file == "" {
		log.Fatal("-file is required")
	}
	command := "validate"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}
	snapshot, err := readSnapshot(file)
	if err != nil {
		log.Fatal(err)
	}
	if batch != "" {
		snapshot.Batch = batch
	}
	if err = validate(snapshot); err != nil {
		log.Fatal(err)
	}
	if command == "validate" {
		printSummary(snapshot)
		return
	}
	if command != "import" || databaseURL == "" {
		log.Fatal("usage: vpet-migrate -file snapshot.json [-batch id] validate|import; import also needs -database")
	}
	ctx := context.Background()
	db, err := store.Open(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = db.MigrateAll(ctx); err != nil {
		log.Fatal(err)
	}
	if err = importSnapshot(ctx, db, snapshot); err != nil {
		log.Fatal(err)
	}
	printSummary(snapshot)
}

func readSnapshot(path string) (*Snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(bufio.NewReader(f))
	dec.DisallowUnknownFields()
	var snapshot Snapshot
	if err = dec.Decode(&snapshot); err != nil {
		return nil, err
	}
	if err = dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("snapshot contains trailing data")
	}
	return &snapshot, nil
}

func validate(s *Snapshot) error {
	if s.Schema != 1 || strings.TrimSpace(s.Batch) == "" {
		return errors.New("snapshot schema or batch is invalid")
	}
	users := map[string]bool{}
	names := map[string]bool{}
	for _, user := range s.Users {
		name := store.NormalizeUsername(user.Username)
		if user.ID == "" || user.Username != strings.TrimSpace(user.Username) || !usernameValid(user.Username) || users[user.ID] || names[name] {
			return fmt.Errorf("duplicate or invalid user %q", user.Username)
		}
		users[user.ID], names[name] = true, true
	}
	for key, raw := range s.KV {
		if key == "" || !json.Valid(raw) {
			return fmt.Errorf("invalid KV record %q", key)
		}
		if strings.HasPrefix(key, "u_") && !users[strings.TrimPrefix(key, "u_")] {
			return fmt.Errorf("user state %q has no snapshot user", key)
		}
		sum := sha256.Sum256(raw)
		expected, ok := s.Checksums[key]
		if !ok || !strings.EqualFold(expected, hex.EncodeToString(sum[:])) {
			return fmt.Errorf("missing or mismatched checksum for %s", key)
		}
	}
	for key := range s.Checksums {
		if _, ok := s.KV[key]; !ok {
			return fmt.Errorf("checksum has no KV record %s", key)
		}
	}
	return nil
}

func usernameValid(username string) bool {
	if len(username) < 3 || len(username) > 32 {
		return false
	}
	for i, r := range username {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || i > 0 && (r == '_' || r == '.' || r == '-')) {
			return false
		}
	}
	return true
}

func importSnapshot(ctx context.Context, db *store.Postgres, s *Snapshot) error {
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM migration_batches WHERE id=$1)`, s.Batch).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return verifyImportedSnapshot(ctx, tx, s)
	}
	var accountCount, gameCount, batchCount int
	if err = tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM accounts), (SELECT count(*) FROM game_kv), (SELECT count(*) FROM migration_batches)`).Scan(&accountCount, &gameCount, &batchCount); err != nil {
		return err
	}
	if accountCount != 0 || gameCount != 0 || batchCount != 0 {
		return errors.New("target is not empty; import requires an empty database")
	}
	if _, err = tx.Exec(ctx, `INSERT INTO migration_batches(id) VALUES($1)`, s.Batch); err != nil {
		return err
	}
	for _, user := range s.Users {
		name := store.NormalizeUsername(user.Username)
		if _, err = tx.Exec(ctx, `INSERT INTO migration_claims(batch_id,mattermost_user_id,username_normalized) VALUES($1,$2,$3)`, s.Batch, user.ID, name); err != nil {
			return err
		}
	}
	keys := make([]string, 0, len(s.KV))
	for key := range s.KV {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err = tx.Exec(ctx, `INSERT INTO game_kv(pkey,pvalue) VALUES($1,$2)`, key, s.KV[key]); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func verifyImportedSnapshot(ctx context.Context, tx pgx.Tx, s *Snapshot) error {
	var claims int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM migration_claims WHERE batch_id=$1`, s.Batch).Scan(&claims); err != nil {
		return err
	}
	if claims != len(s.Users) {
		return errors.New("batch already exists with different claims")
	}
	var keys int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM game_kv`).Scan(&keys); err != nil {
		return err
	}
	if keys != len(s.KV) {
		return errors.New("batch already exists with different game data")
	}
	for _, user := range s.Users {
		var name string
		if err := tx.QueryRow(ctx, `SELECT username_normalized FROM migration_claims WHERE batch_id=$1 AND mattermost_user_id=$2`, s.Batch, user.ID).Scan(&name); err != nil || name != store.NormalizeUsername(user.Username) {
			return errors.New("batch already exists with different claims")
		}
	}
	for key, expected := range s.KV {
		var actual []byte
		if err := tx.QueryRow(ctx, `SELECT pvalue FROM game_kv WHERE pkey=$1`, key).Scan(&actual); err != nil || !jsonEqual(actual, expected) {
			return errors.New("batch already exists with different game data")
		}
	}
	return tx.Commit(ctx)
}

func jsonEqual(a, b []byte) bool {
	var left, right any
	return json.Unmarshal(a, &left) == nil && json.Unmarshal(b, &right) == nil && bytes.Equal(mustCanonical(left), mustCanonical(right))
}

func mustCanonical(v any) []byte {
	raw, _ := json.Marshal(v)
	return raw
}

func printSummary(s *Snapshot) {
	prefixes := map[string]int{}
	for key := range s.KV {
		prefix := key
		if at := strings.IndexByte(key, '_'); at >= 0 {
			prefix = key[:at+1]
		}
		prefixes[prefix]++
	}
	raw, _ := json.MarshalIndent(map[string]any{"batch": s.Batch, "users": len(s.Users), "keys": len(s.KV), "prefixes": prefixes}, "", "  ")
	fmt.Println(string(raw))
}
