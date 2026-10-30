package game

import (
	"strings"
	"testing"
	"time"
)

func huntPet(t *testing.T) *userState {
	t.Helper()
	st := hatch(t)
	p := st.Pets[0]
	p.Hunger, p.Health, p.Lv = 100, 100, 1
	return st
}

func TestHuntNeedsLevelAndRest(t *testing.T) {
	st := huntPet(t)
	if _, err := hunt(st, 0, "nuilua", 0, fixed(0.5)); err == nil {
		t.Fatal("a level 1 pet cannot enter the volcano")
	}
	if _, err := hunt(st, 0, "khongco", 0, fixed(0.5)); err == nil {
		t.Fatal("unknown map should fail")
	}
	if _, err := hunt(st, 0, "vuon", 1, fixed(0.5)); err != nil {
		t.Fatal(err)
	}
	if _, err := hunt(st, 0, "vuon", 2, fixed(0.5)); err == nil {
		t.Fatal("the cooldown should block a second trip")
	}
	after := huntCooldown.Milliseconds() + 10
	if _, err := hunt(st, 0, "vuon", after, fixed(0.5)); err != nil {
		t.Fatal(err)
	}
	st.Pets[0].Hunger = 2
	if _, err := hunt(st, 0, "vuon", 2*after, fixed(0.5)); err == nil {
		t.Fatal("a starving pet should stay home")
	}
}

func TestHuntBringsLootAndXP(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	r := seeded(3)
	gold, before := 0, len(st.Inv)
	for i := 0; i < 300; i++ {
		p.Hunger, p.Health, p.Sick, p.Dead = 100, 100, false, false
		res, err := hunt(st, 0, "vuon", int64(i+1)*huntCooldown.Milliseconds(), r)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Finds) == 0 || len(res.Finds) > 3 || res.XP <= 0 {
			t.Fatalf("bad trip %+v", res)
		}
		for _, f := range res.Finds {
			if f.Rarity < 0 || f.Rarity > 3 {
				t.Fatalf("bad rarity %+v", f)
			}
			if itemByID[f.Item] == nil {
				t.Fatalf("unknown loot %q", f.Item)
			}
			gold += f.Rarity / 3
		}
	}
	if p.Lv < 5 || p.Lv > maxLv {
		t.Fatalf("300 trips should level the pet up: %d", p.Lv)
	}
	if len(p.Skills) != skillSlots(p.Lv) {
		t.Fatalf("level %d gives %d skill slots, pet has %d", p.Lv, skillSlots(p.Lv), len(p.Skills))
	}
	if len(st.Inv) <= before {
		t.Fatal("nothing was brought home")
	}
	if gold != 0 {
		t.Fatalf("the back garden never drops golden loot, got %d", gold)
	}
}

func TestHuntHurtsAndCanKill(t *testing.T) {
	hurt, died := 0, 0
	for seed := 0; seed < 400; seed++ {
		st := huntPet(t)
		p := st.Pets[0]
		p.Lv, p.Health = 40, minHuntHealth
		res, err := hunt(st, 0, "nuilua", time.Hour.Milliseconds(), seeded(int64(seed)))
		if err != nil {
			t.Fatal(err)
		}
		if res.Hurt > 0 {
			hurt++
			if p.Dead {
				died++
				if !strings.Contains(res.Note, "bỏ mạng") {
					t.Fatalf("a dead pet should say so: %q", res.Note)
				}
			}
		} else if p.Health != minHuntHealth {
			t.Fatalf("an unharmed pet keeps its health: %.0f", p.Health)
		}
	}
	if hurt < 60 || hurt > 220 {
		t.Fatalf("the volcano hurts about a third of the time, got %d/400", hurt)
	}
	if died == 0 {
		t.Fatal("a weak pet in the volcano should sometimes die")
	}
}

func TestGearAndSkillRanks(t *testing.T) {
	st := huntPet(t)
	p := st.Pets[0]
	st.add("kiem", 1)
	st.add("dao", 1)
	if _, err := equip(st, 0, "com"); err == nil {
		t.Fatal("food is not gear")
	}
	if _, err := equip(st, 0, "kiem"); err != nil {
		t.Fatal(err)
	}
	if p.Gear["vukhi"] != "kiem" || p.gearOf().Atk != 5 {
		t.Fatalf("weapon not equipped: %+v", p.Gear)
	}
	if _, err := equip(st, 0, "dao"); err != nil {
		t.Fatal(err)
	}
	if p.Gear["vukhi"] != "dao" || st.Inv["kiem"] != 1 {
		t.Fatalf("swapping gear returns the old one: %+v inv %v", p.Gear, st.Inv)
	}
	if _, err := unequipSlot(st, 0, "giap"); err == nil {
		t.Fatal("an empty slot cannot be emptied")
	}
	if _, err := unequipSlot(st, 0, "vukhi"); err != nil || st.Inv["dao"] != 1 || p.Gear["vukhi"] != "" {
		t.Fatalf("unequip failed: %v %v", st.Inv, p.Gear)
	}

	if _, err := skillUp(st, 0, p.Skills[0], 0); err == nil {
		t.Fatal("no stone, no rank")
	}
	st.add("ngoc", 9)
	if _, err := skillUp(st, 0, "khongco", 0); err == nil {
		t.Fatal("cannot raise a skill the pet lacks")
	}
	id := p.Skills[0]
	for lv := 2; lv <= maxSkillLv; lv++ {
		if _, err := skillUp(st, 0, id, 0); err != nil {
			t.Fatal(err)
		}
		if p.skillLv(id) != lv {
			t.Fatalf("rank %d, want %d", p.skillLv(id), lv)
		}
	}
	if _, err := skillUp(st, 0, id, 0); err == nil {
		t.Fatal("max rank should refuse")
	}
	if st.Inv["ngoc"] != 9-(maxSkillLv-1) {
		t.Fatalf("stones spent wrong: %d", st.Inv["ngoc"])
	}
}

func TestEverySkillRankChangesCombatEffect(t *testing.T) {
	for _, sk := range skills {
		base, maxed := skillValue(sk.ID, 1), skillValue(sk.ID, maxSkillLv)
		if maxed <= base {
			t.Errorf("%s rank 1 %.1f rank %d %.1f: stone changes nothing", sk.ID, base, maxSkillLv, maxed)
		}
	}
	if skillValue("lua", 1) != 14 || skillValue("lua", 5) != 28 {
		t.Fatalf("damage scaling drifted: %.0f -> %.0f", skillValue("lua", 1), skillValue("lua", 5))
	}
	if skillValue("khien", 1) != .5 || skillValue("khien", 5) != .9 {
		t.Fatalf("shield scaling drifted: %.1f -> %.1f", skillValue("khien", 1), skillValue("khien", 5))
	}
	if skillValue("an", 1) != 0 || skillValue("an", 5) != .8 {
		t.Fatalf("hide scaling drifted: %.1f -> %.1f", skillValue("an", 1), skillValue("an", 5))
	}

	// Prove the authoritative fight consumes the rank, not only the display helper.
	foe := duelSide{Name: "B", HP: 20, Skills: nil}
	rank1 := duelSide{Name: "A", HP: 40, Skills: []string{"lua"}, SkillLv: map[string]int{"lua": 1}}
	rank5 := rank1
	rank5.SkillLv = map[string]int{"lua": 5}
	a := fight(rank1, foe, fixed(.9, .9, 0, .9, .9))
	b := fight(rank5, foe, fixed(.9, .9, 0, .9, .9))
	if len(a.Turns) == 0 || len(b.Turns) == 0 || a.Turns[0].HP[1] != 6 || b.Turns[0].HP[1] != 0 {
		t.Fatalf("fight ignored skill rank: rank1=%+v rank5=%+v", a.Turns, b.Turns)
	}
}

func TestGearMakesDuelsStronger(t *testing.T) {
	bare := &pet{Name: "A", Health: 100, Lv: 1, Skills: []string{"lua"}}
	armed := &pet{Name: "B", Health: 100, Lv: 20, Skills: []string{"lua"}, SkillLv: map[string]int{"lua": 5},
		Gear: map[string]string{"vukhi": "gaythan", "giap": "longbao", "bua": "vuongmien"}}
	wins := 0
	r := seeded(5)
	for i := 0; i < 100; i++ {
		if fight(sideOf("a", bare), sideOf("b", armed), r).Winner == 1 {
			wins++
		}
	}
	if wins < 90 {
		t.Fatalf("a fully geared level 20 pet should dominate, won %d/100", wins)
	}
}

func TestMedalsFollowTheRecord(t *testing.T) {
	p := &pet{Lv: 1}
	if len(medals(p)) != 0 {
		t.Fatalf("a fresh pet has no medals: %+v", medals(p))
	}
	p.Lv, p.Hunts, p.Wins, p.Gold = maxLv, 100, 50, 1
	if len(medals(p)) != 9 {
		t.Fatalf("a maxed pet earns every medal, got %d", len(medals(p)))
	}
}
