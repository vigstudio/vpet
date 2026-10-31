package game

import "testing"

func TestExpandedCampaignPreservesOldUnlocks(t *testing.T) {
	st := towerPet(t)
	p := st.Pets[0]
	p.Campaign = map[string][3]int{"vuon_6": {3, 2, 1}}
	rows := p.campaignViews(st, towerDay(1))
	if len(rows) != len(huntMaps)*12 {
		t.Fatalf("expected 12 stages per region, got %d", len(rows))
	}
	for _, row := range rows {
		if row.Name == "" {
			t.Fatal("unnamed stage")
		}
		if row.Map == "vuon" && row.Stage == 6 && (!row.Boss || row.Stars != [3]int{3, 2, 1}) {
			t.Fatal("old boss/progress changed")
		}
		if row.Map == "vuon" && row.Stage == 7 && !row.Open {
			t.Fatal("extension should open after old boss")
		}
		if row.Map == "rung" && row.Stage == 1 && !row.Open {
			t.Fatal("old chapter unlock now requires new stages")
		}
		if row.Map == "vuon" && row.Stage == 8 && row.Open {
			t.Fatal("extension skipped a stage")
		}
	}
}

func TestTowerGoldSharesActivityCeiling(t *testing.T) {
	st := towerPet(t)
	now := towerDay(1)
	st.CoinDay, st.CoinToday = dayOf(now), coinsPerDay-2
	coin := st.Coin
	clearFloor(t, st, 1, now)
	if st.Coin-coin != 2 || st.CoinToday != coinsPerDay {
		t.Fatal("tower exceeded daily ceiling")
	}
	clearFloor(t, st, 2, now)
	if st.Coin-coin != 2 {
		t.Fatal("tower paid after cap")
	}
	if st.earn(now, false) != 0 {
		t.Fatal("activity paid after cap")
	}
	if st.towerView(towerDay(2)).Reward.Coin == 0 {
		t.Fatal("new day did not restore reward")
	}
}

func TestTowerMilestoneManualAndSkill(t *testing.T) {
	st := towerPet(t)
	st.TowerFloor = 9
	manuals := st.Inv["sach"]
	before := len(st.Pets[0].Skills)
	clearFloor(t, st, 10, towerDay(1))
	if st.Inv["sach"] != manuals+1 {
		t.Fatal("milestone missing manual")
	}
	if before < skillSlots(st.Pets[0].Lv) && len(st.Pets[0].Skills) <= before {
		t.Fatal("empty skill slot not taught")
	}
	clearFloor(t, st, 10, towerDay(1))
	if st.Inv["sach"] != manuals+1 {
		t.Fatal("milestone paid twice")
	}
}
