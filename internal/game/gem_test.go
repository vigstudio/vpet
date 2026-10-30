package game

import "testing"

func TestCrushOnlyAcceptsJunkAndPaysShards(t *testing.T) {
	st := huntPet(t)
	st.add("dao", 4)
	st.add("kiem", 1)
	st.add("da", 3)
	st.add("kiemrong", 1)
	if _, err := crush(st, "kiemrong", 1); err == nil {
		t.Fatal("a mythic weapon must never be crushable")
	}
	if _, err := crush(st, "com", 1); err == nil {
		t.Fatal("food is not gear")
	}
	if _, err := crush(st, "dao", 9); err == nil {
		t.Fatal("cannot crush more than owned")
	}
	if _, err := crush(st, "dao", 2); err != nil {
		t.Fatal(err)
	}
	if st.Inv["dao"] != 2 || st.Inv["gem_shard"] != 2 {
		t.Fatalf("ordinary gear paid %d shards, bag %v", st.Inv["gem_shard"], st.Inv)
	}
	if _, err := crush(st, "kiem", 1); err != nil {
		t.Fatal(err)
	}
	// Each rung is worth three of the one below, matching what the forge consumes.
	if st.Inv["gem_shard"] != 5 {
		t.Fatalf("rare gear should pay three times an ordinary piece: %d", st.Inv["gem_shard"])
	}
	// Plain stones pay one shard, so they cannot outperform real gear.
	if _, err := crush(st, "da", 3); err != nil {
		t.Fatal(err)
	}
	if st.Inv["gem_shard"] != 8 {
		t.Fatalf("stones should pay one each: %d", st.Inv["gem_shard"])
	}
	// Spare epic and golden pieces are crushable too, and pay the ladder rate.
	st.add("dinhba", 1)
	st.add("gaythan", 1)
	before := st.Inv["gem_shard"]
	if _, err := crush(st, "dinhba", 1); err != nil {
		t.Fatal(err)
	}
	if got := st.Inv["gem_shard"] - before; got != 9 {
		t.Fatalf("epic gear pays nine shards, got %d", got)
	}
	before = st.Inv["gem_shard"]
	if _, err := crush(st, "gaythan", 1); err != nil {
		t.Fatal(err)
	}
	if got := st.Inv["gem_shard"] - before; got != 27 {
		t.Fatalf("golden gear pays 27 shards, got %d", got)
	}
	// Crushing must never beat forging: three pieces fused then crushed returns what the
	// three were worth, so there is no shard printing press.
	if shardValue("kiem") != 3*shardValue("dao") || shardValue("dinhba") != 3*shardValue("kiem") {
		t.Fatal("crush value must mirror the three-to-one forge")
	}
}

func TestGemFusionUsesTwoForOneProgression(t *testing.T) {
	if gemUpCost != 2 {
		t.Fatalf("gem fusion costs %d copies, want 2", gemUpCost)
	}
	if total := 1; total<<(gemMaxTier-1) != 8 {
		t.Fatalf("tier four should cost 8 tier-one gems, got %d", total<<(gemMaxTier-1))
	}
}

func TestGemForgeLadder(t *testing.T) {
	st := huntPet(t)
	if _, err := forgeGem(st, "gem_atk", 1); err == nil {
		t.Fatal("no shards, no gem")
	}
	st.add("gem_shard", gemShards)
	if _, err := forgeGem(st, "gem_atk", 1); err != nil {
		t.Fatal(err)
	}
	if st.Inv["gem_atk_1"] != 1 || st.Inv["gem_shard"] != 0 {
		t.Fatalf("forge left %v", st.Inv)
	}
	if _, err := forgeGem(st, "gem_atk", 2); err == nil {
		t.Fatal("one gem cannot become a tier two")
	}
	st.add("gem_atk_1", gemUpCost-1)
	if _, err := forgeGem(st, "gem_atk", 2); err != nil {
		t.Fatal(err)
	}
	if st.Inv["gem_atk_2"] != 1 || st.Inv["gem_atk_1"] != 0 {
		t.Fatalf("upgrade left %v", st.Inv)
	}
	if _, err := forgeGem(st, "gem_atk", gemMaxTier+1); err == nil {
		t.Fatal("tier is capped")
	}
	if _, err := forgeGem(st, "gem_fake", 1); err == nil {
		t.Fatal("unknown gem kind")
	}
}

func TestSocketAddsStatsAndReturnsGemOnGearChange(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	st.Coin = 500
	st.add("gem_atk_2", 1)
	if _, err := socket(st, 0, "vukhi", "gem_atk_2", 0); err == nil {
		t.Fatal("an empty slot has nothing to socket into")
	}
	st.add("kiem", 1)
	if _, err := equip(st, 0, "kiem"); err != nil {
		t.Fatal(err)
	}
	before := p.gearOf().Atk
	if _, err := socket(st, 0, "vukhi", "gem_atk_2", 0); err != nil {
		t.Fatal(err)
	}
	if st.Coin != 475 || st.Inv["gem_atk_2"] != 0 {
		t.Fatalf("socket cost wrong: coin %d bag %v", st.Coin, st.Inv)
	}
	if got := p.gearOf().Atk; got <= before {
		t.Fatalf("gem did not reach combat stats: %v vs %v", got, before)
	}
	if _, err := unequipSlot(st, 0, "vukhi"); err != nil {
		t.Fatal(err)
	}
	if p.gearOf().Atk != 0 || len(p.Sockets["vukhi"]) != 0 || st.Inv["gem_atk_2"] != 1 {
		t.Fatalf("unequip must return the gem: gear=%+v sockets=%v bag=%v", p.gearOf(), p.Sockets, st.Inv)
	}
	st.add("dinhba", 1)
	if _, err := equip(st, 0, "kiem"); err != nil {
		t.Fatal(err)
	}
	st.add("gem_hp_1", 1)
	if _, err := socket(st, 0, "vukhi", "gem_hp_1", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := equip(st, 0, "dinhba"); err != nil {
		t.Fatal(err)
	}
	if len(p.Sockets["vukhi"]) != 0 || st.Inv["gem_hp_1"] != 1 {
		t.Fatalf("replacement silently inherited a gem: sockets=%v bag=%v", p.Sockets, st.Inv)
	}
	// the socket cap holds
	for i := 0; i < maxSockets; i++ {
		st.add("gem_hp_1", 1)
		socket(st, 0, "vukhi", "gem_hp_1", 0)
	}
	if len(p.Sockets["vukhi"]) > maxSockets {
		t.Fatalf("socket cap broken: %v", p.Sockets["vukhi"])
	}
	// prying one out returns it and charges again
	coin := st.Coin
	if _, err := unsocket(st, 0, "vukhi", 0); err != nil {
		t.Fatal(err)
	}
	if st.Coin != coin-socketCost {
		t.Fatal("unsocket did not charge")
	}
	if _, err := unsocket(st, 0, "vukhi", 99); err == nil {
		t.Fatal("index must be checked")
	}
}

func TestGemsAreNotSoldByTheShop(t *testing.T) {
	for _, it := range sellable() {
		if it.Kind == "gem" {
			t.Fatalf("%s must be player-made only", it.ID)
		}
	}
}
