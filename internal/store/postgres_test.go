package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPostgres(t *testing.T) *Postgres {
	t.Helper()
	dsn := os.Getenv("VPET_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VPET_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("vpet_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	if err = db.MigrateAll(ctx); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		admin.Close()
	})
	return db
}

func TestApplyCommitsAllRowsAndRollsBackOnMarshalError(t *testing.T) {
	db := testPostgres(t)
	if err := db.KVSet("a", []byte(`{"n":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := db.KVSet("remove", []byte(`true`)); err != nil {
		t.Fatal(err)
	}
	if err := db.Apply(map[string]any{"a": map[string]int{"n": 2}, "remove": nil, "z": []string{"ok"}}); err != nil {
		t.Fatal(err)
	}
	if raw, err := db.KVGet("a"); err != nil || string(raw) != `{"n": 2}` {
		t.Fatalf("updated row: raw=%s err=%v", raw, err)
	}
	if raw, err := db.KVGet("remove"); err != nil || raw != nil {
		t.Fatalf("deleted row: raw=%s err=%v", raw, err)
	}

	err := db.Apply(map[string]any{"a": map[string]int{"n": 3}, "z": func() {}})
	if err == nil {
		t.Fatal("unmarshalable transaction succeeded")
	}
	if raw, getErr := db.KVGet("a"); getErr != nil || string(raw) != `{"n": 2}` {
		t.Fatalf("partial update escaped rollback: raw=%s err=%v", raw, getErr)
	}
}

func TestRecoverMarketJournal(t *testing.T) {
	db := testPostgres(t)
	if err := db.KVSet("delete_me", []byte(`{"old":true}`)); err != nil {
		t.Fatal(err)
	}
	journal := `{"delete_me":null,"new_row":{"coin":12}}`
	if err := db.KVSet("market_pending", []byte(journal)); err != nil {
		t.Fatal(err)
	}
	if err := db.RecoverMarketJournal(context.Background()); err != nil {
		t.Fatal(err)
	}
	if raw, err := db.KVGet("delete_me"); err != nil || raw != nil {
		t.Fatalf("journal delete failed: raw=%s err=%v", raw, err)
	}
	if raw, err := db.KVGet("new_row"); err != nil || string(raw) != `{"coin": 12}` {
		t.Fatalf("journal write failed: raw=%s err=%v", raw, err)
	}
	if raw, err := db.KVGet("market_pending"); err != nil || raw != nil {
		t.Fatalf("journal retained after recovery: raw=%s err=%v", raw, err)
	}
}

func TestReservedMigrationUsernameCannotBeRegistered(t *testing.T) {
	db := testPostgres(t)
	ctx := context.Background()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO migration_batches(id) VALUES('batch')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO migration_claims(batch_id,mattermost_user_id,username_normalized) VALUES('batch','legacy-id','reserved')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateAccount(ctx, "new-id", "Reserved", "", "vi", "hash"); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("reserved username result: %v", err)
	}
	if _, err := db.CreateAccount(ctx, "other-id", "available", "", "vi", "hash"); err != nil {
		t.Fatal(err)
	}
}

func TestRedeemClaimSucceedsOnce(t *testing.T) {
	db := testPostgres(t)
	ctx := context.Background()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO migration_batches(id) VALUES('batch')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO migration_claims(batch_id,mattermost_user_id,username_normalized) VALUES('batch','legacy-id','legacyuser')`); err != nil {
		t.Fatal(err)
	}
	account, err := db.RedeemClaim(ctx, "batch", "legacy-id", "LegacyUser", "Legacy User", "vi", "hash")
	if err != nil || account.ID != "legacy-id" || account.Username != "LegacyUser" {
		t.Fatalf("first redeem: account=%+v err=%v", account, err)
	}
	if _, err = db.RedeemClaim(ctx, "batch", "legacy-id", "LegacyUser", "Legacy User", "vi", "hash"); err == nil {
		t.Fatal("claim replay succeeded")
	}
}

func TestConcurrentRedeemClaimHasOneWinner(t *testing.T) {
	db := testPostgres(t)
	ctx := context.Background()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO migration_batches(id) VALUES('batch')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO migration_claims(batch_id,mattermost_user_id,username_normalized) VALUES('batch','legacy-id','legacyuser')`); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, err := db.RedeemClaim(context.Background(), "batch", "legacy-id", "legacyuser", "", "vi", "hash")
			results <- err
		}()
	}
	close(start)
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("redeem winners=%d, want 1", successes)
	}
}

func TestSessionExpiryAndRevocation(t *testing.T) {
	db := testPostgres(t)
	ctx := context.Background()
	if _, err := db.CreateAccount(ctx, "user-id", "sessionuser", "", "vi", "hash"); err != nil {
		t.Fatal(err)
	}
	valid := []byte("01234567890123456789012345678901")
	if err := db.CreateSession(ctx, valid, "user-id", "csrf", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if session, err := db.Session(ctx, valid); err != nil || session.AccountID != "user-id" || session.CSRF != "csrf" {
		t.Fatalf("valid session: session=%+v err=%v", session, err)
	}
	if err := db.RevokeSession(ctx, valid); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Session(ctx, valid); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("revoked session returned %v", err)
	}

	expired := []byte("abcdefghijklmnopqrstuvwxyzABCDEF")
	if err := db.CreateSession(ctx, expired, "user-id", "csrf", time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Session(ctx, expired); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expired session returned %v", err)
	}
}
