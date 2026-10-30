package game

import "testing"

// Every item id the game hands out must exist in the catalog. A stale id used to crash the
// whole plugin (openBag reached for "nhan" after the ring tiers were renamed in 1.9.0), so
// this walks the real draw paths instead of trusting the literals to stay in step.
func TestEveryItemHandedOutExists(t *testing.T) {
	give := func(id string) {
		t.Helper()
		if itemByID[id] == nil {
			t.Fatalf("unknown item id %q", id)
		}
	}

	// every bag kind, over enough seeds to reach each branch
	for _, bag := range []string{"tui", "tui_y", "tui_vukhi", "tui_vang"} {
		give(bag)
		for seed := 0; seed < 120; seed++ {
			st := newUserState()
			before := map[string]int{}
			for k, v := range st.Inv {
				before[k] = v
			}
			openBag(st, bag, seeded(int64(seed)))
			for id, n := range st.Inv {
				if n > before[id] {
					give(id)
				}
			}
		}
	}

	// hunting and wild loot
	for r := 0; r < len(rarityNames); r++ {
		for _, id := range lootOf(r) {
			give(id)
		}
	}
	// gear ladders, shop shelves and the blind box
	for id := range gearByID {
		give(id)
		if to := nextTier(id); to != "" {
			give(to)
		}
	}
	for _, day := range []string{"2026-10-24", "2026-11-01", "2026-12-25"} {
		for _, o := range shopStock(day) {
			give(o.Item)
		}
	}
	for seed := 0; seed < 200; seed++ {
		st := newUserState()
		st.Coin = gachaCost
		res, err := gacha(st, seeded(int64(seed)))
		if err != nil {
			t.Fatal(err)
		}
		give(res.Item)
	}
	// the food bag table
	for seed := 0; seed < 200; seed++ {
		give(rollItem(seeded(int64(seed))))
	}
}

// The mythic rung must stay forge-only. If a map, bag, shop or blind box ever hands one
// out, the ladder everyone is grinding towards collapses in a week.
func TestMythicGearIsForgedNeverFound(t *testing.T) {
	mythic := map[string]bool{}
	for _, tiers := range forgeTiers {
		mythic[tiers[len(tiers)-1]] = true
	}
	if len(mythic) != len(gearSlots) {
		t.Fatalf("expected one mythic piece per slot, got %d", len(mythic))
	}
	found := func(where, id string) {
		t.Helper()
		if mythic[id] {
			t.Fatalf("%s handed out mythic gear %q", where, id)
		}
	}
	for r := 0; r < len(rarityNames); r++ {
		for _, id := range lootOf(r) {
			found("lootOf", id)
		}
	}
	for _, bag := range []string{"tui", "tui_y", "tui_vukhi", "tui_vang"} {
		for seed := 0; seed < 200; seed++ {
			st := newUserState()
			before := map[string]int{}
			for k, v := range st.Inv {
				before[k] = v
			}
			openBag(st, bag, seeded(int64(seed)))
			for id, n := range st.Inv {
				if n > before[id] {
					found("openBag "+bag, id)
				}
			}
		}
	}
	for _, day := range []string{"2026-10-24", "2026-11-01", "2026-12-25"} {
		for _, o := range shopStock(day) {
			found("shopStock", o.Item)
		}
	}
	for seed := 0; seed < 400; seed++ {
		st := newUserState()
		st.Coin = gachaCost
		res, err := gacha(st, seeded(int64(seed)))
		if err != nil {
			t.Fatal(err)
		}
		found("gacha", res.Item)
	}
	// the forge is the one door that must open onto it
	if nextTier("gaythan") != "kiemrong" {
		t.Fatalf("the forge should reach the mythic rung, got %q", nextTier("gaythan"))
	}
}

// Every gear item must be reachable: either it drops, or something fuses into it.
func TestEveryGearPieceCanBeObtained(t *testing.T) {
	forged := map[string]bool{}
	for id := range gearByID {
		if to := nextTier(id); to != "" {
			forged[to] = true
		}
	}
	dropped := map[string]bool{}
	for r := 0; r < len(rarityNames); r++ {
		for _, id := range lootOf(r) {
			dropped[id] = true
		}
	}
	for id := range gearByID {
		if !forged[id] && !dropped[id] {
			t.Fatalf("%q can neither be found nor forged", id)
		}
	}
}
