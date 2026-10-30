package game

import (
	"encoding/json"
	"testing"
	"time"
)

func TestQuestClaimsOnceAndNeedsProgress(t *testing.T) {
	st := huntPet(t)
	now := time.Now().UnixMilli()
	if _, err := claimQuest(st, "d_feed", now); err == nil {
		t.Fatal("claimed with no progress")
	}
	if _, err := claimQuest(st, "khongco", now); err == nil {
		t.Fatal("unknown quest id")
	}
	st.bump("feed", 3)
	coin := st.Coin
	if _, err := claimQuest(st, "d_feed", now); err != nil {
		t.Fatal(err)
	}
	if st.Coin <= coin {
		t.Fatal("claim paid nothing")
	}
	if _, err := claimQuest(st, "d_feed", now); err == nil {
		t.Fatal("a quest must pay only once")
	}
}

func TestDailyResetsButAchievementsDoNot(t *testing.T) {
	st := huntPet(t)
	day1 := time.Date(2026, 3, 1, 10, 0, 0, 0, vnZone).UnixMilli()
	st.bump("hunt", 2)
	if _, err := claimQuest(st, "d_hunt", day1); err != nil {
		t.Fatal(err)
	}
	// the next day re-opens the daily and measures from a fresh snapshot
	day2 := time.Date(2026, 3, 2, 10, 0, 0, 0, vnZone).UnixMilli()
	st.rollDaily(day2)
	if st.QuestDone["d_hunt"] {
		t.Fatal("daily did not reset")
	}
	if _, err := claimQuest(st, "d_hunt", day2); err == nil {
		t.Fatal("yesterday's hunts must not pay today")
	}
	st.bump("hunt", 2)
	if _, err := claimQuest(st, "d_hunt", day2); err != nil {
		t.Fatal(err)
	}
	// lifetime totals survive the reset, so achievements keep counting
	st.bump("hunt", 21)
	if _, err := claimQuest(st, "a_hunt25", day2); err != nil {
		t.Fatalf("25 lifetime hunts should pay: %v", err)
	}
	day3 := time.Date(2026, 3, 3, 10, 0, 0, 0, vnZone).UnixMilli()
	st.rollDaily(day3)
	if !st.QuestDone["a_hunt25"] {
		t.Fatal("an achievement must stay claimed")
	}
}

func TestEveryQuestHasACounterAndValidReward(t *testing.T) {
	for i := range quests {
		q := &quests[i]
		if questCounter(q.ID) == "" {
			t.Fatalf("%s has no counter, so it can never complete", q.ID)
		}
		if q.Need < 1 || q.Gold < 1 {
			t.Fatalf("%s has no target or no reward", q.ID)
		}
		if q.Item != "" && itemByID[q.Item] == nil {
			t.Fatalf("%s pays an item that does not exist: %s", q.ID, q.Item)
		}
	}
}

// An achievement ladder has to get harder and pay better as it climbs, or a later rung is
// either unreachable or worth less than the one before it.
func TestAchievementLaddersRiseWithEveryRung(t *testing.T) {
	type rung struct{ need, gold, tier int }
	byTrack := map[string][]rung{}
	for i := range quests {
		q := &quests[i]
		if q.Daily {
			if q.Track != "" || q.Tier != 0 {
				t.Fatalf("%s is a daily and must not sit on a ladder", q.ID)
			}
			continue
		}
		if q.Track == "" || q.Tier < 1 {
			t.Fatalf("%s is an achievement with no ladder position", q.ID)
		}
		byTrack[q.Track] = append(byTrack[q.Track], rung{q.Need, q.Gold, q.Tier})
	}
	if len(byTrack) == 0 {
		t.Fatal("no achievement ladders at all")
	}
	for track, rungs := range byTrack {
		if len(rungs) != trackTiers[track] {
			t.Fatalf("%s has %d rungs but tops out at tier %d", track, len(rungs), trackTiers[track])
		}
		for i, r := range rungs {
			if r.tier != i+1 {
				t.Fatalf("%s rungs are out of order at %d: tier %d", track, i, r.tier)
			}
			if i == 0 {
				continue
			}
			if r.need <= rungs[i-1].need {
				t.Fatalf("%s tier %d needs %d, not more than %d", track, r.tier, r.need, rungs[i-1].need)
			}
			if r.gold <= rungs[i-1].gold {
				t.Fatalf("%s tier %d pays %d, not more than %d", track, r.tier, r.gold, rungs[i-1].gold)
			}
		}
	}
}

// Quest IDs are the key of QuestDone. Renaming one hands a finished achievement back out, so
// the rungs that have already shipped are pinned here.
func TestShippedQuestIDsNeverChange(t *testing.T) {
	for _, id := range []string{
		"d_feed", "d_hunt", "d_stage", "d_wild", "d_duel", "d_boss",
		"a_hunt25", "a_hunt100", "a_wild50", "a_stage10", "a_stage30", "a_duel10", "a_forge", "a_crush",
	} {
		if questByID[id] == nil {
			t.Fatalf("%s shipped already; removing or renaming it re-opens a claimed quest", id)
		}
	}
}

func TestEveryQuestIsTranslated(t *testing.T) {
	st := huntPet(t)
	src := st.questViews(0)
	for _, lang := range []string{"en", "ja"} {
		out := translateQuestViews(lang, src)
		if len(out) != len(src) {
			t.Fatalf("%s dropped rows: %d vs %d", lang, len(out), len(src))
		}
		for i, v := range out {
			if v.Name == src[i].Name || v.Desc == src[i].Desc {
				t.Errorf("%s: %s still reads Vietnamese (%q / %q)", lang, v.ID, v.Name, v.Desc)
			}
		}
	}
}

func TestQuestViewsShowCappedProgress(t *testing.T) {
	st := huntPet(t)
	now := time.Now().UnixMilli()
	st.bump("feed", 99)
	for _, v := range st.questViews(now) {
		if v.Have > v.Need {
			t.Fatalf("%s shows %d/%d", v.ID, v.Have, v.Need)
		}
	}
}

func TestDailyRollCanBePersistedBeforeAction(t *testing.T) {
	st := huntPet(t)
	day1 := time.Date(2026, 3, 1, 10, 0, 0, 0, vnZone).UnixMilli()
	st.QuestDay = dayOf(day1)
	st.Counts = map[string]int{"feed": 8}
	st.QuestBase = map[string]int{"feed": 5}
	day2 := time.Date(2026, 3, 2, 10, 0, 0, 0, vnZone).UnixMilli()
	if !st.rollDaily(day2) || st.QuestBase["feed"] != 8 {
		t.Fatalf("new baseline not established: day %s base %v", st.QuestDay, st.QuestBase)
	}
	st.bump("feed", 1)
	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	reloaded := &userState{}
	if err := json.Unmarshal(raw, reloaded); err != nil {
		t.Fatal(err)
	}
	if got := reloaded.questProgress(questByID["d_feed"]); got != 1 {
		t.Fatalf("today's completed action disappeared after reload: %d", got)
	}
	if reloaded.rollDaily(day2) {
		t.Fatal("same day must not move the baseline again")
	}
}
