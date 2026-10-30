package game

import (
	"strings"
	"testing"
)

func TestEverySlotHasAFullLadder(t *testing.T) {
	if len(gearSlots) != 8 {
		t.Fatalf("eight slots expected, got %d", len(gearSlots))
	}
	for _, s := range gearSlots {
		tiers := forgeTiers[s.ID]
		if len(tiers) != len(rarityNames) {
			t.Fatalf("slot %s has %d tiers", s.ID, len(tiers))
		}
		for r, id := range tiers {
			it := itemByID[id]
			if it == nil || it.Kind != "gear" {
				t.Fatalf("%s is not a gear item", id)
			}
			if it.Rarity != r {
				t.Fatalf("%s should be rarity %d, got %d", id, r, it.Rarity)
			}
			if gearByID[id].Slot != s.ID {
				t.Fatalf("%s belongs to %q, not %q", id, gearByID[id].Slot, s.ID)
			}
		}
		top := tiers[len(tiers)-1]
		if nextTier(top) != "" {
			t.Fatalf("%s is the top tier and fuses into nothing", top)
		}
		if tierOf(top) != len(tiers)-1 {
			t.Fatalf("%s should sit on rung %d, got %d", top, len(tiers)-1, tierOf(top))
		}
		if nextTier(tiers[0]) != tiers[1] {
			t.Fatalf("%s should fuse into %s", tiers[0], tiers[1])
		}
	}
}

func TestFuseEatsThreeAndCharges(t *testing.T) {
	st := newUserState()
	if _, err := fuse(st, "dao"); err == nil {
		t.Fatal("nothing to fuse")
	}
	st.add("dao", fuseCount)
	if _, err := fuse(st, "dao"); err == nil {
		t.Fatal("the smith is not free")
	}
	st.Coin = fuseCostOf("dao")
	msg, err := fuse(st, "dao")
	if err != nil {
		t.Fatal(err)
	}
	if st.Inv["dao"] != 0 || st.Inv["kiem"] != 1 || st.Coin != 0 {
		t.Fatalf("after fusing: %v coin %d", st.Inv, st.Coin)
	}
	if !strings.Contains(msg, "Kiếm sắt") {
		t.Fatalf("the smith says what came out: %q", msg)
	}
	// mythic is no longer the end of the ladder: three of them forge the divine rung
	st.add("kiemrong", fuseCount)
	st.Coin = 10000
	if _, err := fuse(st, "kiemrong"); err != nil {
		t.Fatalf("three mythic pieces should forge the divine rung: %v", err)
	}
	if st.Inv["kiemthanlong"] != 1 {
		t.Fatalf("divine weapon missing: %v", st.Inv)
	}
	st.add("kiemthanlong", fuseCount)
	if _, err := fuse(st, "kiemthanlong"); err == nil {
		t.Fatal("the top tier cannot be fused further")
	}
	// the fee doubles every rung, so the last step costs far more than the first
	if fuseCostOf("gaythan") <= fuseCostOf("dao") {
		t.Fatalf("fusing up should get dearer: %d then %d", fuseCostOf("dao"), fuseCostOf("gaythan"))
	}
	if _, err := fuse(st, "com"); err == nil {
		t.Fatal("food is not gear")
	}
}

func TestForgeListOnlyShowsWhatYouOwn(t *testing.T) {
	st := newUserState()
	if len(st.forgeList()) != 0 {
		t.Fatal("an empty bag has nothing to forge")
	}
	st.add("muda", 2)
	list := st.forgeList()
	if len(list) != 1 || list[0].From != "muda" || list[0].To != "muthep" || list[0].Ready {
		t.Fatalf("two of three is not ready yet: %+v", list)
	}
	st.add("muda", 1)
	st.Coin = fuseCostOf("muda")
	if list = st.forgeList(); !list[0].Ready || list[0].Have != 3 {
		t.Fatalf("three copies and the fee make it ready: %+v", list)
	}
}

func TestEverySlotCanBeWorn(t *testing.T) {
	st := huntPet(t)
	for _, s := range gearSlots {
		id := forgeTiers[s.ID][1]
		st.add(id, 1)
		if _, err := equip(st, 0, id); err != nil {
			t.Fatal(err)
		}
	}
	p := st.Pets[0]
	if len(p.Gear) != len(gearSlots) {
		t.Fatalf("worn: %v", p.Gear)
	}
	g := p.gearOf()
	if g.Atk <= 0 || g.Def <= 0 || g.HP <= 0 {
		t.Fatalf("a full set should add to everything: %+v", g)
	}
	if _, err := unequipSlot(st, 0, "nhan"); err != nil {
		t.Fatal(err)
	}
	if p.Gear["nhan"] != "" || st.Inv["chuoingoc"] != 1 {
		t.Fatalf("taking a ring off puts it back in the bag: %v %v", p.Gear, st.Inv)
	}
}

// A matched outfit must beat the same number of mismatched pieces, otherwise the set bonus
// changes nothing and the choice it is meant to create does not exist.
func TestSetBonusRewardsAMatchedOutfit(t *testing.T) {
	mixed := map[string]string{"vukhi": "dao", "giap": "giapda", "mu": "muthep"}
	if setOf(mixed) != nil {
		t.Fatalf("three different tiers are not a set: %+v", setOf(mixed))
	}
	matched := map[string]string{"vukhi": "kiem", "giap": "giapda", "mu": "muda"}
	b := setOf(matched)
	if b == nil || b.Parts != 3 || b.Tier != 1 {
		t.Fatalf("three tier-1 pieces are a set: %+v", b)
	}
	if b.Atk <= 0 || b.Def <= 0 || b.HP <= 0 {
		t.Fatalf("the bonus must actually pay: %+v", b)
	}
	matched["gang"] = "gangda"
	if four := setOf(matched); four.Parts != 4 || four.Atk <= b.Atk {
		t.Fatalf("a fourth piece should pay more: %+v then %+v", b, four)
	}
	// the higher ladder wins when two sets are complete
	both := map[string]string{"vukhi": "kiem", "giap": "giapda", "mu": "muda",
		"gang": "gangthep", "giay": "giaythep", "nhan": "nhanbac"}
	if top := setOf(both); top.Tier != 2 {
		t.Fatalf("the better set should win: %+v", top)
	}
	// a slot holding an item that belongs elsewhere is stale data, not a set piece
	stale := map[string]string{"vukhi": "kiem", "giap": "giapda", "mu": "muda", "giay": "muda"}
	if b := setOf(stale); b.Parts != 3 {
		t.Fatalf("a mis-slotted item does not count: %+v", b)
	}
}

// The set bonus must reach the duel, not just the paper doll.
func TestSetBonusReachesTheFight(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	p.Gear = map[string]string{"vukhi": "kiem", "giap": "giapda", "mu": "muda"}
	bare := gearByID["kiem"].Atk + gearByID["giapda"].Atk + gearByID["muda"].Atk
	if got := p.gearOf().Atk; got <= bare {
		t.Fatalf("gearOf should include the set bonus: %v vs %v", got, bare)
	}
}

func TestFuseAllChainsAndStopsAtGoldLimit(t *testing.T) {
	st := newUserState()
	st.Inv = map[string]int{"dao": 9, "aovai": 3}
	st.Coin = 240
	if _, err := fuseAll(st); err != nil {
		t.Fatal(err)
	}
	if st.Inv["dao"] != 0 || st.Inv["kiem"] != 0 || st.Inv["dinhba"] != 1 || st.Inv["giapda"] != 1 {
		t.Fatalf("quick forge did not chain every affordable recipe: %v", st.Inv)
	}
	if st.Coin != 0 || st.Counts["forge"] != 5 {
		t.Fatalf("coin %d, forge progress %d", st.Coin, st.Counts["forge"])
	}

	poor := newUserState()
	poor.Inv["dao"] = 3
	if _, err := fuseAll(poor); err == nil || poor.Inv["dao"] != 3 {
		t.Fatalf("no gold must preserve materials: %v, err %v", poor.Inv, err)
	}
}

func TestEquipBestUsesSetBonusWithoutLosingItems(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	p.Gear = map[string]string{"vukhi": "dinhba", "giap": "giapda", "mu": "muda"}
	st.Inv = map[string]int{"kiem": 1, "gangda": 1, "giayda": 1, "nhandong": 2}
	beforeItems := len(p.Gear)
	for _, n := range st.Inv {
		beforeItems += n
	}
	beforePower := power(p)
	if _, err := equipBest(st, 0); err != nil {
		t.Fatal(err)
	}
	if power(p) <= beforePower {
		t.Fatalf("best outfit did not improve power: %d then %d", beforePower, power(p))
	}
	if p.Gear["vukhi"] != "kiem" || setOf(p.Gear) == nil {
		t.Fatalf("auto equip ignored a stronger matched outfit: %v", p.Gear)
	}
	afterItems := len(p.Gear)
	for _, n := range st.Inv {
		afterItems += n
	}
	if afterItems != beforeItems {
		t.Fatalf("auto equip duplicated or deleted gear: %d then %d, bag %v", beforeItems, afterItems, st.Inv)
	}
	if _, err := equipBest(st, 0); err != nil || afterItems != len(p.Gear)+sumInventory(st.Inv) {
		t.Fatalf("second auto equip changed inventory: %v", err)
	}
	p.Dead = true
	if _, err := equipBest(st, 0); err == nil {
		t.Fatal("a dead pet must not change equipment")
	}
}

func TestEnhancedGearRoundTripsWithoutMovingItsLevel(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	st.Coin = 100000
	st.Inv["gem_shard"] = 1000
	st.add("kiem", 1)
	if _, err := equip(st, 0, "kiem", 0); err != nil {
		t.Fatal(err)
	}
	base := p.gearOf().Atk
	for level := 1; level <= maxEnhanceLevel; level++ {
		if _, err := enhance(st, 0, "vukhi"); err != nil {
			t.Fatalf("enhance +%d: %v", level, err)
		}
	}
	if p.GearLevel["vukhi"] != maxEnhanceLevel || p.gearOf().Atk <= base {
		t.Fatalf("enhancement did not reach combat: level=%v atk=%v base=%v", p.GearLevel, p.gearOf().Atk, base)
	}
	if _, err := unequipSlot(st, 0, "vukhi"); err != nil {
		t.Fatal(err)
	}
	if st.enhancedCount("kiem", maxEnhanceLevel) != 1 || p.GearLevel["vukhi"] != 0 {
		t.Fatalf("enhanced item lost on unequip: bag=%v levels=%v", st.Enhanced, p.GearLevel)
	}
	st.add("dinhba", 1)
	if _, err := equip(st, 0, "dinhba", 0); err != nil {
		t.Fatal(err)
	}
	if p.GearLevel["vukhi"] != 0 {
		t.Fatalf("enhancement moved to another item: %+v", p.GearLevel)
	}
	if _, err := equip(st, 0, "kiem", maxEnhanceLevel); err != nil {
		t.Fatal(err)
	}
	if p.Gear["vukhi"] != "kiem" || p.GearLevel["vukhi"] != maxEnhanceLevel || st.Inv["dinhba"] != 1 {
		t.Fatalf("enhanced item did not return intact: gear=%v levels=%v bag=%v", p.Gear, p.GearLevel, st.Inv)
	}
}

func TestEnhancedGearSurvivesAutoEquipBurialAndRestore(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	p.Gear = map[string]string{"vukhi": "kiem"}
	p.GearLevel = map[string]int{"vukhi": 10}
	st.add("gaythan", 1)
	if _, err := equipBest(st, 0); err != nil {
		t.Fatal(err)
	}
	if p.Gear["vukhi"] != "gaythan" || st.enhancedCount("kiem", 10) != 1 {
		t.Fatalf("auto equip lost the replaced enhanced item: gear=%v levels=%v enhanced=%v", p.Gear, p.GearLevel, st.Enhanced)
	}
	p.Dead, p.DiedAt = true, 1
	if err := bury(st, 0); err != nil {
		t.Fatal(err)
	}
	if st.enhancedCount("kiem", 10) != 1 || st.Inv["gaythan"] != 1 {
		t.Fatalf("burial lost gear: enhanced=%v inv=%v", st.Enhanced, st.Inv)
	}
	if _, err := restoreEnhanced(st, "kiem", 10); err != nil {
		t.Fatal(err)
	}
	if st.enhancedCount("kiem", 10) != 0 || st.Inv["kiem"] != 1 {
		t.Fatalf("restore did not produce one +0 item: enhanced=%v inv=%v", st.Enhanced, st.Inv)
	}
}

func TestEnhancementFailureDoesNotMutateState(t *testing.T) {
	st := huntPet(t)
	st.add("kiem", 1)
	if _, err := equip(st, 0, "kiem", 0); err != nil {
		t.Fatal(err)
	}
	beforeCoin, beforeShards := st.Coin, st.Inv["gem_shard"]
	if _, err := enhance(st, 0, "vukhi"); err == nil {
		t.Fatal("enhancement without materials succeeded")
	}
	if st.Coin != beforeCoin || st.Inv["gem_shard"] != beforeShards || st.Pets[0].GearLevel["vukhi"] != 0 {
		t.Fatalf("failed enhancement mutated state: coin=%d shards=%d levels=%v", st.Coin, st.Inv["gem_shard"], st.Pets[0].GearLevel)
	}
}

func sumInventory(inv map[string]int) int {
	total := 0
	for _, n := range inv {
		total += n
	}
	return total
}
