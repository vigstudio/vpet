package game

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func halloweenFixture(t *testing.T) (*userState, int64) {
	t.Helper()
	now := time.Date(2026, time.October, 31, 12, 0, 0, 0, vnZone).UnixMilli()
	st := hatch(t)
	old := st.Pets[0]
	old.Born, old.DiedAt, old.Dead, old.Lv = 11, now-1000, true, 12
	old.Gear = map[string]string{"vukhi": "kiem"}
	old.GearLevel = map[string]int{"vukhi": 3}
	old.Sockets = map[string][]string{"vukhi": {"gem_atk_2"}}
	old.Acc = "no"
	if err := bury(st, 0); err != nil {
		t.Fatal(err)
	}
	st.Pets = []*pet{{Name: "Hiến tế", Born: 22, Health: 100, Hunger: 70, Happy: 70, LastTick: now, Lv: 1, Gear: map[string]string{"giap": "giapda"}}}
	return st, now
}

func TestHalloweenExchangeConservesEquipment(t *testing.T) {
	st, now := halloweenFixture(t)
	if _, err := resurrect(st, 11, now-1000, 22, now); err != nil {
		t.Fatal(err)
	}
	p := st.Pets[0]
	if p.Born != 11 || p.Dead || p.Lv != 12 || p.GearLevel["vukhi"] != 3 || p.Acc != "no" || len(st.Graves) != 0 {
		t.Fatalf("bad resurrection: %+v", p)
	}
	if st.enhancedCount("kiem", 3) != 0 || st.Inv["gem_atk_2"] != 0 || st.Inv["no"] != 0 || st.Inv["giapda"] != 1 {
		t.Fatalf("duplicated/lost equipment: %v %v", st.Inv, st.Enhanced)
	}
	if _, err := resurrect(st, 11, now-1000, 22, now); err == nil {
		t.Fatal("repeated exchange accepted")
	}
}

func TestHalloweenRejectsWithoutMutation(t *testing.T) {
	for _, scenario := range []string{"outside", "dead", "stale", "missing", "duel", "care"} {
		t.Run(scenario, func(t *testing.T) {
			st, now := halloweenFixture(t)
			when, donor := now, int64(22)
			switch scenario {
			case "outside":
				when = now + 24*hour
			case "dead":
				st.Pets[0].Dead = true
			case "stale":
				donor = 23
			case "missing":
				st.Inv["gem_atk_2"] = 0
			case "duel":
				st.DOut = []string{"pending"}
			case "care":
				st.Pets[0].CareUntil = now + hour
			}
			before, _ := json.Marshal(st)
			if _, err := resurrect(st, 11, now-1000, donor, when); err == nil {
				t.Fatal("invalid exchange accepted")
			}
			after, _ := json.Marshal(st)
			if string(before) != string(after) {
				t.Fatal("failed exchange changed state")
			}
		})
	}
}

func TestHalloweenCommitFailurePreservesStoredState(t *testing.T) {
	st, now := halloweenFixture(t)
	backend := newAllianceTestBackend()
	raw, _ := json.Marshal(st)
	backend.kv[userPrefix+"player"] = append([]byte(nil), raw...)
	backend.applyErr = errors.New("storage unavailable")
	engine := NewEngine(backend)
	if _, err := engine.resurrectPet("player", st, 11, now-1000, 22, now); err == nil {
		t.Fatal("failed commit accepted")
	}
	if string(backend.kv[userPrefix+"player"]) != string(raw) {
		t.Fatal("failed commit changed persisted state")
	}
}

func TestHalloweenLegacyAndTimezone(t *testing.T) {
	st, now := halloweenFixture(t)
	st.Graves[0].Snapshot = nil
	if _, err := resurrect(st, 11, now-1000, 22, now); err != nil {
		t.Fatal(err)
	}
	if p := st.Pets[0]; p.Lv != 1 || len(p.Gear) != 0 || len(p.Skills) != 0 {
		t.Fatal("invented legacy progress")
	}
	if st.enhancedCount("kiem", 3) != 1 {
		t.Fatal("legacy revival consumed unrecorded gear")
	}
	if !halloweenOpen(time.Date(2026, 10, 30, 17, 0, 0, 0, time.UTC).UnixMilli()) || halloweenOpen(time.Date(2026, 10, 31, 17, 0, 0, 0, time.UTC).UnixMilli()) {
		t.Fatal("event must use Vietnam midnight")
	}
}
