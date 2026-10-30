package game

import "testing"

func TestPowerRisesWithLevelGearAndSkills(t *testing.T) {
	bare := &pet{Lv: 1, Health: 50}
	if power(bare) <= 0 {
		t.Fatal("even a bare pet has some power")
	}
	levelled := &pet{Lv: 20, Health: 50}
	if power(levelled) <= power(bare) {
		t.Fatal("levels should count")
	}
	geared := &pet{Lv: 1, Health: 50, Gear: map[string]string{"vukhi": "kiemrong", "giap": "giaprong"}}
	if power(geared) <= power(bare) {
		t.Fatal("worn gear should count")
	}
	trained := &pet{Lv: 1, Health: 50, Skills: []string{"set", "hut"}, SkillLv: map[string]int{"set": 5, "hut": 5}}
	if power(trained) <= power(&pet{Lv: 1, Health: 50, Skills: []string{"set", "hut"}}) {
		t.Fatal("skill ranks should count")
	}
}

// The board must field a pet that can actually fight when one is available, so a player
// with a wounded champion and a healthy rookie is not shown as unable to fight.
func TestBestPrefersAFitPet(t *testing.T) {
	strongButDead := &pet{Name: "To", Lv: 30, Health: 90, Dead: true}
	weakButFit := &pet{Name: "Nhỏ", Lv: 2, Health: 80}
	st := &userState{Pets: []*pet{strongButDead, weakButFit}}
	if got := best(st); got != weakButFit {
		t.Fatalf("a fit pet should be fielded, got %q", got.Name)
	}
	// with nothing fit, the strongest still shows so the player appears on the board
	hurt := &pet{Name: "Yếu", Lv: 9, Health: 5}
	if got := best(&userState{Pets: []*pet{strongButDead, hurt}}); got == nil {
		t.Fatal("a player with pets must still appear")
	}
	if best(&userState{}) != nil {
		t.Fatal("no pets, no entry")
	}
}

func TestBestPrefersArenaPointsBeforePower(t *testing.T) {
	highPower := &pet{Name: "Mạnh", Lv: 100, Health: 100, Hunger: 100, Happy: 100, ArenaPoints: 900}
	highRank := &pet{Name: "Có hạng", Lv: 10, Health: 100, Hunger: 100, Happy: 100, ArenaPoints: 1400}
	if got := best(&userState{Pets: []*pet{highPower, highRank}}); got != highRank {
		t.Fatalf("arena pet = %q, want highest points", got.Name)
	}
}

func TestArenaRanksByPointsThenPower(t *testing.T) {
	rows := []arenaEntry{
		{Name: "a", Wins: 20, Points: 980, Power: 900},
		{Name: "b", Wins: 5, Points: 1100, Power: 100},
		{Name: "c", Wins: 1, Points: 1100, Power: 300},
	}
	sortArena(rows)
	if rows[0].Name != "c" || rows[1].Name != "b" || rows[2].Name != "a" {
		t.Fatalf("points first, then power: %s %s %s", rows[0].Name, rows[1].Name, rows[2].Name)
	}
}

func TestArenaPointChangeRewardsHarderWins(t *testing.T) {
	if got := arenaPointChange(1000, 1000, true); got != 20 {
		t.Fatalf("even win = %d, want 20", got)
	}
	if arenaPointChange(1000, 1300, true) <= arenaPointChange(1000, 700, true) {
		t.Fatal("beating a stronger opponent must pay more")
	}
	if arenaPointChange(1000, 1300, false) <= arenaPointChange(1000, 700, false) {
		t.Fatal("losing to a weaker opponent must cost more")
	}
}
