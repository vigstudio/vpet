package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func validSnapshot() *Snapshot {
	raw := json.RawMessage(`{"coin":3}`)
	sum := sha256.Sum256(raw)
	return &Snapshot{
		Schema: 1,
		Batch:  "cutover-2026-10-30",
		Users:  []SnapshotUser{{ID: "user-id", Username: "player.one"}},
		KV:     map[string]json.RawMessage{"u_user-id": raw},
		Checksums: map[string]string{
			"u_user-id": hex.EncodeToString(sum[:]),
		},
	}
}

func TestJSONEqualIgnoresJSONBFormatting(t *testing.T) {
	if !jsonEqual([]byte(`{"coin": 3, "items": ["a"]}`), []byte(`{"items":["a"],"coin":3}`)) {
		t.Fatal("equivalent JSON rejected")
	}
	if jsonEqual([]byte(`{"coin":3}`), []byte(`{"coin":4}`)) {
		t.Fatal("different JSON accepted")
	}
}

func TestValidateSnapshot(t *testing.T) {
	if err := validate(validSnapshot()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSnapshotRejectsMissingChecksumAndUnknownUserState(t *testing.T) {
	s := validSnapshot()
	delete(s.Checksums, "u_user-id")
	if err := validate(s); err == nil {
		t.Fatal("snapshot without checksum accepted")
	}

	s = validSnapshot()
	raw := json.RawMessage(`{}`)
	s.KV["u_missing"] = raw
	sum := sha256.Sum256(raw)
	s.Checksums["u_missing"] = hex.EncodeToString(sum[:])
	if err := validate(s); err == nil {
		t.Fatal("state without matching user accepted")
	}
}

func TestUsernameValid(t *testing.T) {
	for _, username := range []string{"abc", "Player.One", "a_b-c"} {
		if !usernameValid(username) {
			t.Fatalf("valid username rejected: %q", username)
		}
	}
	for _, username := range []string{"ab", "_abc", "a b", "a/../b"} {
		if usernameValid(username) {
			t.Fatalf("invalid username accepted: %q", username)
		}
	}
}
