package game

import (
	"strings"
	"testing"
)

func TestWildSideScalesWithTheRegion(t *testing.T) {
	r := seeded(11)
	low := wildSide(mapByID["vuon"], 1, r)
	high := wildSide(mapByID["nuilua"], 40, r)
	if low.Lv < 1 || low.Lv > 4 {
		t.Fatalf("a garden foe stays near level 1: %d", low.Lv)
	}
	if high.Lv < 35 || high.HP <= low.HP || high.Atk <= low.Atk {
		t.Fatalf("a volcano foe should be far stronger: %+v vs %+v", high, low)
	}
	for _, s := range []duelSide{low, high} {
		if len(s.Skills) < 2 || len(s.Skills) > maxSkillSlots || !strings.Contains(s.Name, "hoang") {
			t.Fatalf("bad wild side %+v", s)
		}
		seen := map[string]bool{}
		for _, id := range s.Skills {
			if seen[id] || skillByID[id] == nil {
				t.Fatalf("duplicate or unknown skill in %v", s.Skills)
			}
			seen[id] = true
		}
		for k, limit := range partLimits {
			if v, ok := s.Parts[k]; !ok || v < 0 || v >= limit {
				t.Fatalf("part %s out of range: %d", k, v)
			}
		}
	}
}

func TestEncounterGatesAndUsesOnlyEnergyAndHungerForPacing(t *testing.T) {
	st := huntPet(t)
	if _, err := encounter(st, 0, "nuilua", 0, fixed(0.5)); err == nil {
		t.Fatal("a level 1 pet cannot roam the volcano")
	}
	if _, err := encounter(st, 0, "khongco", 0, fixed(0.5)); err == nil {
		t.Fatal("unknown region should fail")
	}
	p := st.Pets[0]
	p.Lv = 10
	p.Energy, p.EnergyAt, p.Hunger = p.maxEnergy(), 1, 100
	for n := 0; n < 13; n++ {
		if _, err := encounter(st, 0, "vuon", int64(n+1), seeded(int64(n))); err != nil {
			t.Fatalf("trip %d was blocked by a hidden timer or quota: %v", n+1, err)
		}
	}
	p.Energy = energyWild - 1
	if _, err := encounter(st, 0, "vuon", 20, seeded(20)); err == nil {
		t.Fatal("an empty energy pool should block the trip")
	}
	p.Energy, p.Hunger = energyWild, wildHunger-1
	if _, err := encounter(st, 0, "vuon", 21, seeded(21)); err == nil {
		t.Fatal("hunger should still block the trip")
	}
}

func TestEncounterWinsLosesAndRewards(t *testing.T) {
	wins, losses, loot := 0, 0, 0
	for seed := 0; seed < 200; seed++ {
		st := huntPet(t)
		p := st.Pets[0]
		p.Lv = 8
		before := p.Health
		res, err := encounter(st, 0, "vuon", int64(seed+1), seeded(int64(seed)))
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Duel.Turns) == 0 || res.XP <= 0 {
			t.Fatalf("every encounter is a real fight with xp: %+v", res)
		}
		if res.Won {
			wins++
			loot += len(res.Finds)
			if p.Wild != 1 || p.Health != before {
				t.Fatalf("a win costs no health: %+v", p)
			}
		} else {
			losses++
			if res.Hurt <= 0 || p.Health >= before || p.Effect != "dau" {
				t.Fatalf("a loss hurts: hurt %.0f health %.0f effect %q", res.Hurt, p.Health, p.Effect)
			}
		}
		if p.Hunger != 100-wildHunger {
			t.Fatalf("roaming costs hunger: %.0f", p.Hunger)
		}
	}
	if wins < 40 || losses < 20 {
		t.Fatalf("encounters should go both ways: %d wins, %d losses", wins, losses)
	}
	if loot < wins/2 {
		t.Fatalf("most wins drop something: %d loot for %d wins", loot, wins)
	}
}

func TestEncounterCanKillInTheVolcano(t *testing.T) {
	died := 0
	for seed := 0; seed < 300 && died == 0; seed++ {
		st := huntPet(t)
		p := st.Pets[0]
		p.Lv, p.Health = 35, minHuntHealth
		res, err := encounter(st, 0, "nuilua", int64(seed+1), seeded(int64(seed)))
		if err != nil {
			t.Fatal(err)
		}
		if p.Dead {
			died++
			if !strings.Contains(res.Note, "bỏ mạng") || res.Won {
				t.Fatalf("a dead pet lost and is reported: %+v", res)
			}
		}
	}
	if died == 0 {
		t.Fatal("a weak pet losing in the volcano should sometimes die")
	}
}

func TestWildPetsCarryTheirArt(t *testing.T) {
	seen := map[int]bool{}
	for i, m := range huntMaps {
		for seed := 0; seed < 60; seed++ {
			s := wildSide(&huntMaps[i], m.MinLv, seeded(int64(seed)))
			if s.Sprite < i*5+1 || s.Sprite > i*5+5 {
				t.Fatalf("%s drew sprite %d, outside its five", m.ID, s.Sprite)
			}
			seen[s.Sprite] = true
		}
	}
	if len(seen) != len(huntMaps)*5 {
		t.Fatalf("every wild pet should turn up: %d of %d", len(seen), len(huntMaps)*5)
	}
}
