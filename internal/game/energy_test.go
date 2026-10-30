package game

import "testing"

func TestEnergyPoolGrowsWithLevelAndRefills(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	if p.maxEnergy() != energyBase {
		t.Fatalf("level 1 pool is %d", p.maxEnergy())
	}
	p.Lv = 10
	if p.maxEnergy() != energyBase+9*energyPerLv {
		t.Fatalf("level 10 pool is %d", p.maxEnergy())
	}
	// an untouched pet starts full rather than locked out
	if p.energyAt(1000) != p.maxEnergy() {
		t.Fatal("a new pet should start with a full pool")
	}
	now := int64(1_000_000)
	p.settleEnergy(now)
	if err := p.spendEnergy(p.maxEnergy(), now); err != nil {
		t.Fatal(err)
	}
	if err := p.spendEnergy(1, now); err == nil {
		t.Fatal("an empty pool cannot pay")
	}
	if p.energyAt(now+energyTick.Milliseconds()*3) != 3 {
		t.Fatalf("three ticks should return three points, got %d", p.energyAt(now+energyTick.Milliseconds()*3))
	}
	// the remainder of a tick survives a settle, so repeated reads never round progress away
	p.settleEnergy(now + energyTick.Milliseconds()*3/2)
	if p.Energy != 1 {
		t.Fatalf("half a tick should not round up: %d", p.Energy)
	}
	if p.energyAt(now+energyTick.Milliseconds()*2) != 2 {
		t.Fatal("the partial tick was discarded")
	}
	if p.energyAt(now+energyTick.Milliseconds()*999) != p.maxEnergy() {
		t.Fatal("regen should stop at the cap")
	}
}

func TestEnergyItemsRefillAndRaiseTheCap(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	now := int64(5_000_000)
	p.settleEnergy(now)
	if _, err := drinkEnergy(st, 0, "nuoc_tang_luc", 5, now); err == nil {
		t.Fatal("a full pet should not waste a potion")
	}
	p.Energy = 2
	st.add("nuoc_tang_luc", 1)
	if _, err := drinkEnergy(st, 0, "nuoc_tang_luc", 5, now); err != nil {
		t.Fatal(err)
	}
	if p.Energy != 7 || st.Inv["nuoc_tang_luc"] != 0 {
		t.Fatalf("potion left %d energy, %d in bag", p.Energy, st.Inv["nuoc_tang_luc"])
	}
	base := p.maxEnergy()
	st.add("the_nang_luong", 1)
	if _, err := growEnergy(st, 0, "the_nang_luong", 5, now); err != nil {
		t.Fatal(err)
	}
	if p.maxEnergy() != base+5 || p.Energy != 12 {
		t.Fatalf("card gave max %d energy %d", p.maxEnergy(), p.Energy)
	}
	p.EnergyMax = maxEnergyCard
	st.add("the_nang_luong", 1)
	if _, err := growEnergy(st, 0, "the_nang_luong", 5, now); err == nil {
		t.Fatal("the permanent bonus must stop at its ceiling")
	}
	if st.Inv["the_nang_luong"] != 1 {
		t.Fatal("a refused card must stay in the bag")
	}
}
