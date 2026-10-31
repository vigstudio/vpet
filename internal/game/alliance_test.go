package game

import (
	"encoding/json"
	"errors"
	"sort"
	"testing"
)

type allianceTestBackend struct {
	kv        map[string][]byte
	users     map[string]*User
	applyErr  error
	lastApply map[string]any
}

func newAllianceTestBackend() *allianceTestBackend {
	return &allianceTestBackend{kv: map[string][]byte{}, users: map[string]*User{}}
}

func (b *allianceTestBackend) KVGet(key string) ([]byte, error) { return b.kv[key], nil }
func (b *allianceTestBackend) KVSet(key string, value []byte) error {
	b.kv[key] = append([]byte(nil), value...)
	return nil
}
func (b *allianceTestBackend) KVDelete(key string) error { delete(b.kv, key); return nil }
func (b *allianceTestBackend) KVList(page, perPage int) ([]string, error) {
	keys := make([]string, 0, len(b.kv))
	for key := range b.kv {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	start := page * perPage
	if start >= len(keys) {
		return nil, nil
	}
	return keys[start:min(start+perPage, len(keys))], nil
}
func (b *allianceTestBackend) Apply(rows map[string]any) error {
	b.lastApply = rows
	if b.applyErr != nil {
		return b.applyErr
	}
	encoded := map[string][]byte{}
	for key, value := range rows {
		if value == nil {
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		encoded[key] = raw
	}
	for key, value := range rows {
		if value == nil {
			delete(b.kv, key)
		} else {
			b.kv[key] = encoded[key]
		}
	}
	return nil
}
func (b *allianceTestBackend) User(id string) (*User, error) {
	if u := b.users[id]; u != nil {
		return u, nil
	}
	return nil, errors.New("missing user")
}
func (b *allianceTestBackend) UserByUsername(username string) (*User, error) {
	for _, u := range b.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, errors.New("missing user")
}
func (*allianceTestBackend) Notify(string, string, string, string, string) {}
func (*allianceTestBackend) LogError(string, ...any)                       {}

func allianceTestState(level, coin int) *userState {
	st := newUserState()
	st.Coin = coin
	st.Pets = []*pet{{Name: "Mun", Lv: level}}
	return st
}

func createTestAlliance(t *testing.T, engine *Engine, user string, st *userState, now int64) *alliance {
	t.Helper()
	if _, err := engine.allianceAction(user, st, "create", "", "Long Hội", "LH", "", true, now); err != nil {
		t.Fatal(err)
	}
	pointer, err := engine.loadAllianceMember(user)
	if err != nil {
		t.Fatal(err)
	}
	a, err := engine.loadAlliance(pointer.Alliance)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAllianceNameAndTagValidation(t *testing.T) {
	if _, _, err := cleanAllianceName("ab"); err == nil {
		t.Fatal("accepted a two-rune name")
	}
	if name, key, err := cleanAllianceName("  Long   Hội  "); err != nil || name != "Long Hội" || key != "long hội" {
		t.Fatalf("normalization = %q/%q, err=%v", name, key, err)
	}
	if tag, err := cleanAllianceTag(" lh2 "); err != nil || tag != "LH2" {
		t.Fatalf("tag = %q, err=%v", tag, err)
	}
	for _, tag := range []string{"A", "ABCDEF", "L-H", "HỘI"} {
		if _, err := cleanAllianceTag(tag); err == nil {
			t.Fatalf("accepted invalid tag %q", tag)
		}
	}
}

func TestCreateAllianceChargesAndCommitsEveryInvariantTogether(t *testing.T) {
	backend := newAllianceTestBackend()
	backend.users["leader"] = &User{ID: "leader", Username: "leader"}
	engine := NewEngine(backend)
	st := allianceTestState(10, 900)

	a := createTestAlliance(t, engine, "leader", st, 1000)
	if st.Coin != 400 || len(a.Members) != 1 || a.Members[0].Role != "leader" {
		t.Fatalf("coin=%d members=%+v", st.Coin, a.Members)
	}
	for _, key := range []string{allianceIndexKey, allianceKeyPrefix + a.ID, allianceMemberKey + "leader", userPrefix + "leader"} {
		if _, ok := backend.lastApply[key]; !ok {
			t.Fatalf("create omitted atomic row %s", key)
		}
	}
}

func TestCreateAllianceRejectsDuplicateNameAndTag(t *testing.T) {
	backend := newAllianceTestBackend()
	backend.users["a"] = &User{ID: "a", Username: "a"}
	backend.users["b"] = &User{ID: "b", Username: "b"}
	engine := NewEngine(backend)
	createTestAlliance(t, engine, "a", allianceTestState(10, 500), 1)

	if _, err := engine.allianceAction("b", allianceTestState(10, 1000), "create", "", " long hội ", "NEW", "", true, 2); err == nil {
		t.Fatal("accepted duplicate normalized name")
	}
	if _, err := engine.allianceAction("b", allianceTestState(10, 1000), "create", "", "Hội Khác", "lh", "", true, 2); err == nil {
		t.Fatal("accepted duplicate tag")
	}
}

func TestAllianceCreateRequirements(t *testing.T) {
	backend := newAllianceTestBackend()
	engine := NewEngine(backend)
	if _, err := engine.allianceAction("u", allianceTestState(9, 1000), "create", "", "Long Hội", "LH", "", true, 1); err == nil {
		t.Fatal("level nine created an alliance")
	}
	if _, err := engine.allianceAction("u", allianceTestState(10, 499), "create", "", "Long Hội", "LH", "", true, 1); err == nil {
		t.Fatal("creation without 500 gold succeeded")
	}
}

func TestOpenAllianceJoinIsAtomicAndUnique(t *testing.T) {
	backend := newAllianceTestBackend()
	backend.users["leader"] = &User{ID: "leader", Username: "leader"}
	backend.users["member"] = &User{ID: "member", Username: "member"}
	engine := NewEngine(backend)
	a := createTestAlliance(t, engine, "leader", allianceTestState(10, 500), 1)

	if _, err := engine.allianceAction("member", allianceTestState(1, 0), "join", a.ID, "", "", "", false, 2); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{allianceKeyPrefix + a.ID, allianceMemberKey + "member"} {
		if _, ok := backend.lastApply[key]; !ok {
			t.Fatalf("join omitted atomic row %s", key)
		}
	}
	if _, err := engine.allianceAction("member", allianceTestState(1, 0), "join", a.ID, "", "", "", false, 3); err == nil {
		t.Fatal("member joined twice")
	}
}

func TestClosedAllianceNeedsInvite(t *testing.T) {
	backend := newAllianceTestBackend()
	backend.users["leader"] = &User{ID: "leader", Username: "leader"}
	backend.users["member"] = &User{ID: "member", Username: "member"}
	engine := NewEngine(backend)
	a := createTestAlliance(t, engine, "leader", allianceTestState(10, 500), 1)
	if _, err := engine.allianceAction("leader", allianceTestState(10, 0), "open", "", "", "", "", false, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.allianceAction("member", allianceTestState(1, 0), "join", a.ID, "", "", "", false, 3); err == nil {
		t.Fatal("joined a closed alliance without invitation")
	}
	if _, err := engine.allianceAction("leader", allianceTestState(10, 0), "invite", "", "", "", "member", false, 4); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.allianceAction("member", allianceTestState(1, 0), "join", a.ID, "", "", "", false, 5); err != nil {
		t.Fatal(err)
	}
}

func TestAllianceCapacityTwenty(t *testing.T) {
	backend := newAllianceTestBackend()
	backend.users["leader"] = &User{ID: "leader", Username: "leader"}
	engine := NewEngine(backend)
	a := createTestAlliance(t, engine, "leader", allianceTestState(10, 500), 1)
	for i := 1; i < allianceMaxMembers; i++ {
		id := string(rune('a' + i))
		backend.users[id] = &User{ID: id, Username: id}
		if _, err := engine.allianceAction(id, allianceTestState(1, 0), "join", a.ID, "", "", "", false, int64(i+1)); err != nil {
			t.Fatal(err)
		}
	}
	backend.users["overflow"] = &User{ID: "overflow", Username: "overflow"}
	if _, err := engine.allianceAction("overflow", allianceTestState(1, 0), "join", a.ID, "", "", "", false, 100); err == nil {
		t.Fatal("twenty-first member joined")
	}
}

func TestAllianceRolePermissions(t *testing.T) {
	backend := newAllianceTestBackend()
	for _, id := range []string{"leader", "officer", "member"} {
		backend.users[id] = &User{ID: id, Username: id}
	}
	engine := NewEngine(backend)
	a := createTestAlliance(t, engine, "leader", allianceTestState(10, 500), 1)
	for _, id := range []string{"officer", "member"} {
		if _, err := engine.allianceAction(id, allianceTestState(1, 0), "join", a.ID, "", "", "", false, 2); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := engine.allianceAction("member", allianceTestState(1, 0), "promote", "officer", "", "", "", false, 3); err == nil {
		t.Fatal("ordinary member promoted someone")
	}
	if _, err := engine.allianceAction("leader", allianceTestState(10, 0), "promote", "officer", "", "", "", false, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.allianceAction("officer", allianceTestState(1, 0), "kick", "leader", "", "", "", false, 4); err == nil {
		t.Fatal("officer kicked leader")
	}
	if _, err := engine.allianceAction("officer", allianceTestState(1, 0), "kick", "member", "", "", "", false, 4); err != nil {
		t.Fatal(err)
	}
}

func TestAllianceLeaveAndKickLockJoiningForSeventyTwoHours(t *testing.T) {
	backend := newAllianceTestBackend()
	for _, id := range []string{"leader", "leaver", "kicked"} {
		backend.users[id] = &User{ID: id, Username: id}
	}
	engine := NewEngine(backend)
	a := createTestAlliance(t, engine, "leader", allianceTestState(10, 500), 1)
	for _, id := range []string{"leaver", "kicked"} {
		if _, err := engine.allianceAction(id, allianceTestState(1, 0), "join", a.ID, "", "", "", false, 2); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := engine.allianceAction("leaver", allianceTestState(1, 0), "leave", "", "", "", "", false, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.allianceAction("leader", allianceTestState(10, 0), "kick", "kicked", "", "", "", false, 10); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"leaver", "kicked"} {
		if _, err := engine.allianceAction(id, allianceTestState(1, 0), "join", a.ID, "", "", "", false, 10+allianceJoinLock.Milliseconds()-1); err == nil {
			t.Fatalf("%s bypassed the cooldown", id)
		}
		if _, err := engine.allianceAction(id, allianceTestState(1, 0), "join", a.ID, "", "", "", false, 10+allianceJoinLock.Milliseconds()); err != nil {
			t.Fatalf("%s remained locked: %v", id, err)
		}
	}
}

func TestJoinRejectsOrphanAllianceRecord(t *testing.T) {
	backend := newAllianceTestBackend()
	engine := NewEngine(backend)
	a := &alliance{ID: "orphan", Name: "Orphan", Tag: "ORP", Open: true, Members: []allianceMember{}}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	backend.kv[allianceKeyPrefix+a.ID] = raw
	if _, err := engine.allianceAction("member", allianceTestState(1, 0), "join", a.ID, "", "", "", false, 1); err == nil {
		t.Fatal("joined an alliance absent from the uniqueness index")
	}
}

func TestAllianceApplyFailureChangesNoStoredRow(t *testing.T) {
	backend := newAllianceTestBackend()
	engine := NewEngine(backend)
	st := allianceTestState(10, 500)
	backend.applyErr = errors.New("apply failed")
	if _, err := engine.allianceAction("leader", st, "create", "", "Long Hội", "LH", "", true, 1); err == nil {
		t.Fatal("create ignored apply failure")
	}
	if len(backend.kv) != 0 {
		t.Fatalf("partial rows survived: %v", backend.kv)
	}
}
