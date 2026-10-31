package game

import (
	"testing"
	"time"
)

func TestTerritoryMapIsFourByThreeAndSymmetric(t *testing.T) {
	if len(territoryDefs) != 12 {
		t.Fatalf("territories = %d", len(territoryDefs))
	}
	entrances := 0
	for _, land := range territoryDefs {
		if land.X < 0 || land.X > 3 || land.Y < 0 || land.Y > 2 {
			t.Fatalf("bad coordinate: %+v", land)
		}
		if land.Entrance {
			entrances++
		}
		for _, adjacent := range land.Adjacent {
			back := false
			for _, candidate := range territoryDefinition(adjacent).Adjacent {
				back = back || candidate == land.ID
			}
			if !back {
				t.Fatalf("%s -> %s is not symmetric", land.ID, adjacent)
			}
		}
	}
	if entrances != 4 {
		t.Fatalf("entrances = %d", entrances)
	}
}

func TestWarTargetRulesAndTerritoryCap(t *testing.T) {
	s := newWarSeason(time.Date(2026, 10, 31, 0, 0, 0, 0, vnZone).UnixMilli())
	if !canTargetTerritory(s, "a", "land-01") || canTargetTerritory(s, "a", "land-02") {
		t.Fatal("landless alliance must enter through a corner")
	}
	s.Territories[0].Owner = "a"
	if !canTargetTerritory(s, "a", "land-02") || !canTargetTerritory(s, "a", "land-05") || canTargetTerritory(s, "a", "land-12") {
		t.Fatal("adjacency rule failed")
	}
	s.Territories[1].Owner, s.Territories[4].Owner = "a", "a"
	if canTargetTerritory(s, "a", "land-03") {
		t.Fatal("alliance exceeded three territories")
	}
}

func TestWarSettlementIsIdempotentAndTieKeepsOwner(t *testing.T) {
	backend := newAllianceTestBackend()
	engine := NewEngine(backend)
	s := newWarSeason(time.Date(2026, 10, 31, 0, 0, 0, 0, vnZone).UnixMilli())
	s.Territories[0].Owner = "holder"
	day := warDayKey(s, 0)
	s.Influence[day] = map[string]map[string]int{"land-01": {"holder": 250, "attacker": 250}}
	if !engine.settleWarDay(s, 0) || s.Territories[0].Owner != "holder" {
		t.Fatal("tie did not preserve incumbent")
	}
	points := s.Scores["holder"].Points
	if engine.settleWarDay(s, 0) || s.Scores["holder"].Points != points {
		t.Fatal("settlement ran twice")
	}
}

func TestWarAttackCapsContributionAndWaitsForMembership(t *testing.T) {
	backend := newAllianceTestBackend()
	backend.users["leader"] = &User{ID: "leader", Username: "leader"}
	engine := NewEngine(backend)
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, vnZone).UnixMilli()
	st := allianceTestState(20, 500)
	st.Pets[0].Hunger, st.Pets[0].Health = 100, 100
	a := createTestAlliance(t, engine, "leader", st, now-int64(25*time.Hour/time.Millisecond))
	_, season, err := engine.ensureWar(now)
	if err != nil {
		t.Fatal(err)
	}
	day := warDayKey(season, 0)
	season.Targets[day] = map[string]string{a.ID: "land-01"}
	if err := engine.commitRows(map[string]any{warSeasonKeyPrefix + season.ID: season}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, _, err := engine.allianceWarAction("leader", st, "attack", "land-01", 0, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := engine.allianceWarAction("leader", st, "attack", "land-01", 0, now); err == nil {
		t.Fatal("fourth attack succeeded")
	}
	season, _ = engine.loadWarSeason(season.ID)
	if season.Contribution["leader"].Points > 30 {
		t.Fatalf("daily contribution = %d", season.Contribution["leader"].Points)
	}
}

func TestSeasonBoostCapExpiryAndStatsAgree(t *testing.T) {
	now := int64(1000)
	st := allianceTestState(20, 0)
	st.SeasonBoostBPS, st.SeasonBoostUntil = 900, now+1
	pt := st.Pets[0]
	base, side := power(pt), boostedSideOf("u", pt, st, now)
	if boostedPower(pt, st, now) != int(float64(base)*1.05+0.5) {
		t.Fatalf("boosted power = %d base=%d", boostedPower(pt, st, now), base)
	}
	sheet := boostedStatsOf(pt, st, now)
	if sheet.Power != boostedPower(pt, st, now) || side.Atk != sheet.Atk || side.Def != sheet.Def || side.HP != sheet.HP {
		t.Fatalf("side and stats differ: %+v %+v", side, sheet)
	}
	if boostedPower(pt, st, now+1) != base {
		t.Fatal("expired boost remained active")
	}
}

func TestWarViewDoesNotTreatNeutralLandAsMine(t *testing.T) {
	backend := newAllianceTestBackend()
	engine := NewEngine(backend)
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, vnZone).UnixMilli()
	view := engine.warState("visitor", &allianceMemberPointer{}, nil, now)
	for _, land := range view.Territories {
		if land.Mine {
			t.Fatalf("neutral territory %s marked as mine", land.ID)
		}
	}
}

func TestTerritoryNamesAreTranslated(t *testing.T) {
	for _, locale := range []string{"en", "ja"} {
		for _, land := range territoryDefs {
			if got := translatedTerritoryName(locale, land.ID); got == land.Name || got == "" {
				t.Fatalf("%s territory %s was not translated: %q", locale, land.ID, got)
			}
		}
	}
}

func TestWarRolloverCatchesUpDowntimeAndClosesEachSeason(t *testing.T) {
	backend := newAllianceTestBackend()
	backend.users["u"] = &User{ID: "u", Username: "u"}
	engine := NewEngine(backend)
	start := time.Date(2026, 10, 31, 0, 0, 0, 0, vnZone).UnixMilli()
	current, first, err := engine.ensureWar(start)
	if err != nil {
		t.Fatal(err)
	}
	first.Contribution["u"] = &warContribution{User: "u", Points: 80, Wins: 3, ReachedAt: start + 1}
	if err := engine.commitRows(map[string]any{warSeasonKeyPrefix + first.ID: first}); err != nil {
		t.Fatal(err)
	}

	now := time.UnixMilli(start).In(vnZone).AddDate(0, 0, 29).UnixMilli()
	engine = NewEngine(backend) // restart after missing more than one full season
	current, active, err := engine.ensureWar(now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.UnixMilli(start).In(vnZone).AddDate(0, 0, 28).UnixMilli()
	if active.Start != wantStart || current.Season != active.ID {
		t.Fatalf("active season = %s start=%d, want start=%d", active.ID, active.Start, wantStart)
	}
	closedFirst, err := engine.loadWarSeason(first.ID)
	if err != nil || closedFirst.Status != "closed" || len(closedFirst.PersonalRanks) != 1 {
		t.Fatalf("first season not finalized: status=%q ranks=%+v err=%v", closedFirst.Status, closedFirst.PersonalRanks, err)
	}
	middleID := time.UnixMilli(start).In(vnZone).AddDate(0, 0, 14).Format("20060102")
	middle, err := engine.loadWarSeason(middleID)
	if err != nil || middle.Status != "closed" || current.Previous != middleID {
		t.Fatalf("middle season not closed: previous=%q status=%q err=%v", current.Previous, middle.Status, err)
	}
	before := len(backend.kv)
	if _, same, err := engine.ensureWar(now); err != nil || same.ID != active.ID || len(backend.kv) != before {
		t.Fatalf("repeat catch-up changed state: season=%v keys=%d/%d err=%v", same, len(backend.kv), before, err)
	}
}

func TestSeasonRewardClaimIsAtomicAndOneTime(t *testing.T) {
	backend := newAllianceTestBackend()
	engine := NewEngine(backend)
	now := int64(1_000_000)
	previous := newWarSeason(0)
	previous.Status = "closed"
	previous.Entitlements["u"] = &warEntitlement{User: "u", BPS: 500, AllianceBPS: 400, PersonalBPS: 200, Until: now + 1000}
	current := newWarSeason(previous.End)
	if err := engine.commitRows(map[string]any{
		warCurrentKey:                    &warCurrent{Season: current.ID, Previous: previous.ID},
		warSeasonKeyPrefix + previous.ID: previous,
		warSeasonKeyPrefix + current.ID:  current,
	}); err != nil {
		t.Fatal(err)
	}
	st := allianceTestState(20, 0)
	if _, err := engine.claimSeasonReward("u", st, now); err != nil {
		t.Fatal(err)
	}
	if st.SeasonBoostBPS != 500 || st.SeasonBoostUntil != now+1000 || st.SeasonBoostSource != previous.ID {
		t.Fatalf("boost state = %+v", st)
	}
	for _, key := range []string{userPrefix + "u", warSeasonKeyPrefix + previous.ID} {
		if _, ok := backend.lastApply[key]; !ok {
			t.Fatalf("claim omitted atomic row %s", key)
		}
	}
	if _, err := engine.claimSeasonReward("u", st, now); err == nil {
		t.Fatal("second claim succeeded")
	}
}

func TestWarAttackUsesSelectedPetAndCommitsUserWithSeason(t *testing.T) {
	backend := newAllianceTestBackend()
	backend.users["leader"] = &User{ID: "leader", Username: "leader"}
	engine := NewEngine(backend)
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, vnZone).UnixMilli()
	st := allianceTestState(20, 500)
	st.Pets[0].Name, st.Pets[0].Hunger, st.Pets[0].Health = "Một", 100, 100
	st.Pets = append(st.Pets, &pet{Name: "Hai", Lv: 30, Hunger: 100, Health: 100})
	a := createTestAlliance(t, engine, "leader", st, now-int64(25*time.Hour/time.Millisecond))
	_, season, err := engine.ensureWar(now)
	if err != nil {
		t.Fatal(err)
	}
	day := warDayKey(season, 0)
	season.Targets[day] = map[string]string{a.ID: "land-01"}
	if err := engine.commitRows(map[string]any{warSeasonKeyPrefix + season.ID: season}); err != nil {
		t.Fatal(err)
	}
	_, replay, err := engine.allianceWarAction("leader", st, "attack", "land-01", 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if replay == nil || replay.A.Name != "Hai" || st.Pets[0].Hunger != 100 || st.Pets[1].Hunger != 90 {
		t.Fatalf("selected pet not used: replay=%+v hunger=%.0f/%.0f", replay, st.Pets[0].Hunger, st.Pets[1].Hunger)
	}
	for _, key := range []string{userPrefix + "leader", warSeasonKeyPrefix + season.ID} {
		if _, ok := backend.lastApply[key]; !ok {
			t.Fatalf("attack omitted atomic row %s", key)
		}
	}
}

func TestWarRanksUseStableFinalTieBreak(t *testing.T) {
	backend := newAllianceTestBackend()
	backend.users["a"] = &User{ID: "a", Username: "a"}
	backend.users["b"] = &User{ID: "b", Username: "b"}
	engine := NewEngine(backend)
	s := newWarSeason(0)
	s.Scores["b"] = &warAllianceScore{Alliance: "b", Points: 10, ReachedAt: 1}
	s.Scores["a"] = &warAllianceScore{Alliance: "a", Points: 10, ReachedAt: 1}
	s.Contribution["b"] = &warContribution{User: "b", Points: 10, Wins: 1, ReachedAt: 1}
	s.Contribution["a"] = &warContribution{User: "a", Points: 10, Wins: 1, ReachedAt: 1}
	if got := engine.rankAlliances(s); len(got) != 2 || got[0].Alliance != "a" {
		t.Fatalf("alliance tie order = %+v", got)
	}
	if got := engine.rankPlayers(s); len(got) != 2 || got[0].User != "a" {
		t.Fatalf("personal tie order = %+v", got)
	}
}

func TestAllianceRewardUsesContributionEarnedForCurrentAlliance(t *testing.T) {
	backend := newAllianceTestBackend()
	backend.users["u"] = &User{ID: "u", Username: "u"}
	engine := NewEngine(backend)
	s := newWarSeason(0)
	s.End = allianceJoinLock.Milliseconds() * 2
	s.Scores["old"] = &warAllianceScore{Alliance: "old", Points: 200, ReachedAt: 1}
	s.Scores["new"] = &warAllianceScore{Alliance: "new", Points: 100, ReachedAt: 2}
	s.Contribution["u"] = &warContribution{User: "u", Alliance: "new", AlliancePoints: map[string]int{"old": 60, "new": 10}, Points: 70, ReachedAt: 3}
	if err := engine.commitRows(map[string]any{allianceMemberKey + "u": &allianceMemberPointer{Alliance: "new", JoinedAt: 0}}); err != nil {
		t.Fatal(err)
	}
	engine.finalizeSeason(s)
	if reward := s.Entitlements["u"]; reward != nil && reward.AllianceBPS != 0 {
		t.Fatalf("old alliance contribution granted new alliance reward: %+v", reward)
	}
}
