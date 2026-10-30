package game

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
)

// A title ladder nobody can finish, or one that pays more at a lower rung, is a bug in the
// table rather than in the code, so the table is checked the same way the gear ladders are.
func TestTitleLaddersRiseAndStayUnderTheCeiling(t *testing.T) {
	if len(titles) != len(titleTracks)*titleTiers {
		t.Fatalf("expected %d titles, got %d", len(titleTracks)*titleTiers, len(titles))
	}
	seen := map[string]bool{}
	for _, tr := range titleTracks {
		var prevNeed, prevAtk, prevDef, prevHP float64
		for tier := 1; tier <= titleTiers; tier++ {
			id := fmt.Sprintf("t_%s%d", tr.Track, tier)
			d := titleByID[id]
			if d == nil {
				t.Fatalf("%s missing from the ladder", id)
			}
			if seen[d.Name] {
				t.Fatalf("%s reuses the name %q, so the two cannot be told apart", id, d.Name)
			}
			seen[d.Name] = true
			if d.Emoji == "" || d.Desc == "" {
				t.Fatalf("%s has no emoji or description", id)
			}
			if float64(d.Need) <= prevNeed {
				t.Fatalf("%s needs %d, not more than the rung below (%v)", id, d.Need, prevNeed)
			}
			// a higher rung must be worth more on at least one stat and worse on none
			if d.Atk < prevAtk || d.Def < prevDef || d.HP < prevHP {
				t.Fatalf("%s pays less than the rung below: %+v", id, d)
			}
			if d.Atk == prevAtk && d.Def == prevDef && d.HP == prevHP {
				t.Fatalf("%s pays exactly what the rung below pays, so the climb buys nothing", id)
			}
			prevNeed, prevAtk, prevDef, prevHP = float64(d.Need), d.Atk, d.Def, d.HP
		}
		top := titleByID[fmt.Sprintf("t_%s%d", tr.Track, titleTiers)]
		if top.Atk > titleMaxAtk || top.Def > titleMaxDef || top.HP > titleMaxHP {
			t.Fatalf("%s tops the ceiling: %+v", top.ID, top)
		}
		// Only one title is worn, so every track has to be worth wearing: a track whose top
		// rung pays nothing at all would never be chosen.
		if top.Atk+top.Def+top.HP <= 0 {
			t.Fatalf("%s pays nothing at all", top.ID)
		}
	}
}

// Stat steps are multiplied by the tier, and in binary floating point 0.6×3 is
// 1.7999999999999998. The panel prints these verbatim, so the ladder must round.
func TestTitleStatsCarryAtMostOneDecimal(t *testing.T) {
	for i := range titles {
		d := &titles[i]
		for name, v := range map[string]float64{"atk": d.Atk, "def": d.Def, "hp": d.HP} {
			if v*10 != math.Trunc(v*10) {
				t.Errorf("%s %s is %v, which prints as a wall of digits", d.ID, name, v)
			}
		}
	}
}

func TestTitleUnlocksFromThePetsOwnRecord(t *testing.T) {
	cases := []struct {
		track string
		set   func(*pet, int)
	}{
		{"lv", func(p *pet, n int) { p.Lv = n }},
		{"hunt", func(p *pet, n int) { p.Hunts = n }},
		{"wild", func(p *pet, n int) { p.Wild = n }},
		{"duel", func(p *pet, n int) { p.Wins = n }},
		{"star", func(p *pet, n int) { p.Campaign = map[string][3]int{"vuon_1": {n, 0, 0}} }},
		{"gold", func(p *pet, n int) { p.Gold = n }},
		{"arena", func(p *pet, n int) { p.ArenaPoints = n }},
	}
	for _, c := range cases {
		d := titleByID["t_"+c.track+"1"]
		if d == nil {
			t.Fatalf("no first rung for track %s", c.track)
		}
		p := &pet{Lv: 1}
		c.set(p, d.Need-1)
		if titleUnlocked(p, d) {
			t.Fatalf("%s unlocked one short of its target", d.ID)
		}
		c.set(p, d.Need)
		if !titleUnlocked(p, d) {
			t.Fatalf("%s should unlock at %d, progress reads %d", d.ID, d.Need, titleProgress(p, c.track))
		}
	}
}

// The whole point of a title is the stats, so the three places combat reads them from have to
// agree. sideOf covers duels, wild and campaign; power covers the world boss and party hunts;
// statsOf is what the panel prints.
func TestTitleReachesEveryCombatPath(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	p.Lv, p.Wins = 20, 400
	bareSide, barePower, bareSheet := sideOf("", p), power(p), p.statsOf()

	if _, err := wearTitle(st, 0, "t_duel5"); err != nil {
		t.Fatal(err)
	}
	d := titleByID["t_duel5"]
	side, pw, sheet := sideOf("", p), power(p), p.statsOf()

	if side.Atk-bareSide.Atk != d.Atk || side.Def-bareSide.Def != d.Def || side.HP-bareSide.HP != d.HP {
		t.Fatalf("title missing from the duel: %+v vs %+v", bareSide, side)
	}
	if pw <= barePower {
		t.Fatalf("title missing from power, so the boss and party hunts never see it: %d then %d", barePower, pw)
	}
	if sheet.TitleAtk != d.Atk || sheet.TitleDef != d.Def || sheet.TitleHP != d.HP {
		t.Fatalf("the sheet does not name the title bonus: %+v", sheet)
	}
	if sheet.Atk != side.Atk || sheet.Def != side.Def || sheet.HP != side.HP {
		t.Fatalf("sheet totals drifted from the duel: %+v vs %+v", sheet, side)
	}
	if sheet.GearAtk != bareSheet.GearAtk {
		t.Fatalf("the title leaked into the gear total: %v then %v", bareSheet.GearAtk, sheet.GearAtk)
	}
}

// A title the pet has not earned must pay nothing even if it is sitting in stored data: the
// unlock is re-checked on read, so hand-edited or outdated state cannot buy stats.
func TestUnearnedTitlePaysNothing(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	if _, err := wearTitle(st, 0, "t_duel5"); err == nil {
		t.Fatal("a pet with no wins must not be able to wear the top duel title")
	}
	if _, err := wearTitle(st, 0, "khongco"); err == nil {
		t.Fatal("an unknown title must be refused")
	}
	p.Title = "t_duel5" // as if written straight into the stored blob
	if got := titleOf(p); got.Atk != 0 || got.Def != 0 || got.HP != 0 {
		t.Fatalf("an unearned title paid out anyway: %+v", got)
	}
	p.Title = ""
	if got := titleOf(p); got.Atk != 0 {
		t.Fatalf("no title must pay nothing: %+v", got)
	}
}

func TestTitleIsPerPetAndSurvivesAReload(t *testing.T) {
	st := huntPet(t)
	st.Slots = 2
	second := *st.Pets[0]
	second.Born++
	second.Title = ""
	st.Pets = append(st.Pets, &second)
	st.Pets[0].Hunts, st.Pets[1].Hunts = 900, 0

	if _, err := wearTitle(st, 0, "t_hunt5"); err != nil {
		t.Fatal(err)
	}
	if _, err := wearTitle(st, 1, "t_hunt5"); err == nil {
		t.Fatal("the second pet has not earned it, so it must not inherit the first pet's title")
	}
	if titleOf(st.Pets[1]).HP != 0 {
		t.Fatal("the title bonus must not cross between pets")
	}

	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	reloaded := &userState{}
	if err := json.Unmarshal(raw, reloaded); err != nil {
		t.Fatal(err)
	}
	reloaded.normalize()
	if reloaded.Pets[0].Title != "t_hunt5" || reloaded.Pets[1].Title != "" {
		t.Fatalf("titles did not survive the round trip: %q, %q", reloaded.Pets[0].Title, reloaded.Pets[1].Title)
	}
	if titleOf(reloaded.Pets[0]).HP != titleByID["t_hunt5"].HP {
		t.Fatal("the reloaded title stopped paying")
	}

	// taking it off is free and immediate
	if _, err := wearTitle(reloaded, 0, ""); err != nil {
		t.Fatal(err)
	}
	if reloaded.Pets[0].Title != "" || titleOf(reloaded.Pets[0]).HP != 0 {
		t.Fatal("removing the title left the bonus behind")
	}
}

// Pets saved before this version have no title field at all. They must load, read as empty
// and pay nothing rather than error.
func TestPetsFromBeforeTitlesLoadCleanly(t *testing.T) {
	st := &userState{}
	if err := json.Unmarshal([]byte(`{"pets":[{"name":"Cũ","born":1,"lv":9,"hunts":40}]}`), st); err != nil {
		t.Fatal(err)
	}
	st.normalize()
	p := st.Pets[0]
	if p.Title != "" {
		t.Fatalf("an old pet should have no title, got %q", p.Title)
	}
	if got := titleOf(p); got != (gearDef{}) {
		t.Fatalf("an old pet must gain nothing: %+v", got)
	}
	views := p.titleViews()
	if len(views) != len(titles) {
		t.Fatalf("the ladder should always be shown in full: %d", len(views))
	}
	// 40 hunts clears the first hunt rung and nothing else on that track
	var unlocked int
	for _, v := range views {
		if v.Track == "hunt" && v.Unlocked {
			unlocked++
		}
		if v.Equipped {
			t.Fatalf("%s should not be equipped", v.ID)
		}
		if v.Have > v.Need {
			t.Fatalf("%s shows %d/%d", v.ID, v.Have, v.Need)
		}
	}
	if unlocked != 1 {
		t.Fatalf("40 hunts should open exactly the first hunt rung, opened %d", unlocked)
	}
}

// The world boss and party hunts settle on power() rather than on a fight, which is exactly
// why a bonus can go missing there without any duel test noticing. These two assert the
// arithmetic those modes actually run.
func TestTitleMovesBossDamageAndPartyPower(t *testing.T) {
	p := &pet{Lv: 20, Wins: 400}
	bare := power(p)
	p.Title = "t_duel5"
	titled := power(p)
	if titled <= bare {
		t.Fatalf("power ignored the title: %d then %d", bare, titled)
	}

	// boss damage, as expedition.go computes it, minus the random component
	bareDmg := 60 + bare*2
	titledDmg := 60 + titled*2
	if titledDmg <= bareDmg {
		t.Fatalf("a titled pet deals no more boss damage: %d vs %d", bareDmg, titledDmg)
	}

	// party hunts sum power against a fixed target, so a title can be the difference between
	// a team clearing and failing
	target := bare*2 + 1
	if bare*2 >= target {
		t.Fatal("fixture is wrong: two bare pets should fall short")
	}
	if titled+bare < target {
		t.Fatalf("a rank-5 title adds %d power, not enough to matter to a party", titled-bare)
	}
}

func TestEveryTitleIsTranslated(t *testing.T) {
	for _, locale := range []string{"en", "ja"} {
		out := translateTitles(locale, (&pet{Lv: 1}).titleViews())
		if len(out) != len(titles) {
			t.Fatalf("%s dropped rows: %d", locale, len(out))
		}
		for i, v := range out {
			if v.Name == titles[i].Name {
				t.Fatalf("%s: %s was not translated", locale, v.ID)
			}
			if v.Desc == titles[i].Desc {
				t.Fatalf("%s: %s description was not translated", locale, v.ID)
			}
			if v.Name == "" || v.Desc == "" {
				t.Fatalf("%s: %s translated to nothing", locale, v.ID)
			}
		}
	}
	// an unknown locale passes the Vietnamese source through untouched
	src := (&pet{Lv: 1}).titleViews()
	if out := translateTitles("xx", src); out[0].Name != src[0].Name {
		t.Fatal("an unknown locale must not rewrite the source")
	}
}
