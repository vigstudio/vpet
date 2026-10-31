package game

import (
	"encoding/json"
	"math"
	"reflect"
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
	st.add("dao", 3)
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
	st.add("kiemrong", 3)
	st.Coin = 10000
	if _, err := fuse(st, "kiemrong"); err != nil {
		t.Fatalf("three mythic pieces should forge the divine rung: %v", err)
	}
	if st.Inv["kiemthanlong"] != 1 {
		t.Fatalf("divine weapon missing: %v", st.Inv)
	}
	st.add("kiemthanlong", 3)
	if _, err := fuse(st, "kiemthanlong"); err == nil {
		t.Fatal("Supreme requires bound souls")
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

func TestEveryEnhancementLevelChangesAStat(t *testing.T) {
	for id, base := range gearByID {
		previous := enhancedGear(id, 0)
		for level := 1; level <= maxEnhanceLevel; level++ {
			got := enhancedGear(id, level)
			if got.Atk == previous.Atk && got.Def == previous.Def && got.HP == previous.HP {
				t.Fatalf("%s +%d changed no stat: %+v", id, level, got)
			}
			previous = got
		}
		if got := enhancedGear(id, maxEnhanceLevel); got.Atk <= base.Atk && got.Def <= base.Def && got.HP <= base.HP {
			t.Fatalf("%s +%d did not improve its base stats", id, maxEnhanceLevel)
		}
	}
}

func TestEnhancementCostGrowsLinearly(t *testing.T) {
	for id := range gearByID {
		first, _ := enhanceCost(id, 1)
		for level := 2; level <= maxEnhanceLevel; level++ {
			got, _ := enhanceCost(id, level)
			if got != first*level {
				t.Fatalf("%s +%d costs %d, want linear %d", id, level, got, first*level)
			}
		}
	}
}

func TestSupremeLaddersPreserveOldTiersAndStats(t *testing.T) {
	old := map[string][]string{
		"vukhi":   {"dao", "kiem", "dinhba", "gaythan", "kiemrong", "kiemthanlong"},
		"giap":    {"aovai", "giapda", "giapthep", "longbao", "giaprong", "giapthanlong"},
		"mu":      {"nonre", "muda", "muthep", "vuongmien", "muthan", "muthiende"},
		"gang":    {"gangvai", "gangda", "gangthep", "gangrong", "gangthan", "gangthienloi"},
		"giay":    {"depco", "giayda", "giaythep", "hiabay", "hiathan", "hiathienvuong"},
		"nhan":    {"nhandong", "chuoingoc", "nhanbac", "nhanrong", "nhanthan", "nhanthienmenh"},
		"khien":   {"motnap", "khiengo", "khiensat", "khienbac", "khienthan", "khienthienha"},
		"aokhoac": {"aotoi", "aogam", "aolong", "aosao", "aothan", "aothienmenh"},
	}
	for slot, previous := range old {
		ladder := forgeTiers[slot]
		if len(ladder) != 7 || !reflect.DeepEqual(ladder[:6], previous) {
			t.Fatalf("changed ladder %s: %v", slot, ladder)
		}
		for tier, id := range previous {
			if tierOf(id) != tier {
				t.Fatalf("changed old tier %s", id)
			}
			if tier < 5 {
				st := newUserState()
				st.Inv = map[string]int{id: 3}
				st.Coin = 40 * (1 << tier)
				if _, err := fuse(st, id); err != nil || st.Coin != 0 || st.Inv[ladder[tier+1]] != 1 {
					t.Fatalf("old recipe %s: %v", id, err)
				}
			}
		}
		a, b := gearByID[ladder[5]], gearByID[ladder[6]]
		for _, pair := range [][2]float64{{a.Atk, b.Atk}, {a.Def, b.Def}, {a.HP, b.HP}} {
			if pair[0] == 0 {
				if pair[1] != 0 {
					t.Fatalf("changed slot profile %s", slot)
				}
				continue
			}
			if ratio := pair[1] / pair[0]; ratio < 1.35 || ratio > 1.45 {
				t.Fatalf("bad ratio %s: %v", slot, ratio)
			}
		}
	}
}

func TestSupremeForgeRequiresAllInputsWithoutPartialMutation(t *testing.T) {
	for _, tc := range []struct {
		copies, coin, souls int
		ok                  bool
	}{{3, 5000, 3, false}, {4, 4999, 3, false}, {4, 5000, 2, false}, {4, 5000, 3, true}} {
		st := newUserState()
		st.Inv = map[string]int{"kiemthanlong": tc.copies, "tinhhon": tc.souls}
		st.Coin = tc.coin
		before, _ := json.Marshal(st)
		r := st.forgeList()[0]
		if r.Need != 4 || r.Cost != 5000 || r.Material != "tinhhon" || r.MaterialNeed != 3 || r.MaterialHave != tc.souls || r.Ready != tc.ok {
			t.Fatalf("bad recipe %+v", r)
		}
		_, err := fuse(st, "kiemthanlong")
		if (err == nil) != tc.ok {
			t.Fatalf("case %+v: %v", tc, err)
		}
		if !tc.ok {
			after, _ := json.Marshal(st)
			if string(before) != string(after) {
				t.Fatal("failed forge mutated state")
			}
		} else if st.Coin != 0 || st.Inv["tinhhon"] != 0 || st.Inv["kiemthanlong"] != 0 || st.Inv["kiemhongmong"] != 1 {
			t.Fatalf("bad consumption %+v", st)
		}
	}
}

func TestSupremeQuickForgeSkipsMissingSoulsAndKeepsLowerProgress(t *testing.T) {
	st := newUserState()
	st.Inv = map[string]int{"kiemthanlong": 8, "dao": 3}
	st.Coin = 20000
	if _, err := fuseAll(st); err != nil {
		t.Fatal(err)
	}
	if st.Inv["kiemthanlong"] != 8 || st.Inv["kiemhongmong"] != 0 || st.Inv["kiem"] != 1 || st.Coin != 19960 {
		t.Fatalf("bad quick forge %+v", st)
	}
	st.Inv["tinhhon"] = 3
	if _, err := fuseAll(st); err != nil {
		t.Fatal(err)
	}
	if st.Inv["kiemthanlong"] != 4 || st.Inv["kiemhongmong"] != 1 || st.Inv["tinhhon"] != 0 || st.Coin != 14960 {
		t.Fatalf("overspent materials %+v", st)
	}
	before, _ := json.Marshal(st)
	if _, err := fuseAll(st); err == nil {
		t.Fatal("missing souls accepted")
	}
	after, _ := json.Marshal(st)
	if string(before) != string(after) {
		t.Fatal("no-op mutated state")
	}
}

func TestSupremeSetAndAutoEquip(t *testing.T) {
	st := huntPet(t)
	for _, ladder := range forgeTiers {
		st.add(ladder[6], 1)
	}
	if _, err := equipBest(st, 0); err != nil {
		t.Fatal(err)
	}
	b := setOf(st.Pets[0].Gear)
	if b == nil || b.Tier != 6 || b.Parts != 8 || b.Name != "Bộ Tối Thượng" || math.Abs(b.Atk-33.6) > 1e-9 || math.Abs(b.Def-25.2) > 1e-9 || b.HP != 210 {
		t.Fatalf("bad Supreme set %+v", b)
	}
	for _, locale := range []string{"en", "ja"} {
		if translateSet(locale, b).Name == b.Name {
			t.Fatal("untranslated set")
		}
	}
	coin, _ := enhanceCost("kiemhongmong", 10)
	if coin != 560 {
		t.Fatalf("nonlinear enhancement %d", coin)
	}
}

func TestLegacyStateLoadsWithoutSupremeFields(t *testing.T) {
	var st userState
	if err := json.Unmarshal([]byte(`{"coin":6000,"inv":{"kiemthanlong":4},"pets":[]}`), &st); err != nil {
		t.Fatal(err)
	}
	if st.Inv["tinhhon"] != 0 || st.forgeList()[0].Ready {
		t.Fatal("legacy state conjured souls")
	}
	if _, err := fuse(&st, "kiemthanlong"); err == nil {
		t.Fatal("legacy state bypassed souls")
	}
}

func TestSoulEssenceCannotBeBoughtTradedOrFoundNormally(t *testing.T) {
	st := newUserState()
	st.Coin = 100000
	st.Inv["tinhhon"] = 99
	if checkListing(st, "tinhhon", 1, 1) == nil || checkBuyOrder(st, "tinhhon", 1, 1) == nil || validateBundle(map[string]int{"tinhhon": 1}) == nil || crushable("tinhhon") {
		t.Fatal("bound material escaped")
	}
	if _, err := buy(st, "tinhhon", 1, 1700000000000); err == nil {
		t.Fatal("shop sold bound material")
	}
	if itemByID["tinhhon"].Weight != 0 {
		t.Fatal("food bag can drop soul")
	}
	for _, it := range sellable() {
		if it.ID == "tinhhon" {
			t.Fatal("shop/gacha can drop soul")
		}
	}
	for r := 0; r < len(rarityNames); r++ {
		for _, id := range lootOf(r) {
			if id == "tinhhon" {
				t.Fatal("hunt/wild can drop soul")
			}
		}
	}
	for _, m := range huntMaps {
		for _, id := range m.Finds {
			if id == "tinhhon" {
				t.Fatal("ordinary map lists soul")
			}
		}
	}
	for _, locale := range []string{"en", "ja"} {
		for _, it := range translatedCatalog(locale) {
			if (tierOf(it.ID) == 6 || it.ID == "tinhhon") && (it.Name == itemByID[it.ID].Name || it.Desc == itemByID[it.ID].Desc) {
				t.Fatalf("untranslated %s %s", locale, it.ID)
			}
		}
		if translatedRarities(locale)[6] == rarityNames[6] {
			t.Fatal("untranslated rarity")
		}
	}
}
