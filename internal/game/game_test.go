package game

import (
	"testing"
	"time"
)

func fixed(vals ...float64) rng {
	i := 0
	return func() float64 {
		v := vals[i%len(vals)]
		i++
		return v
	}
}

const hour = int64(3600 * 1000)

func hatch(t *testing.T) *userState {
	t.Helper()
	st := newUserState()
	if err := createPet(st, "Mèo Cá", map[string]int{"head": 2, "c1": 3}, 0, fixed(0.0, 0.99)); err != nil {
		t.Fatal(err)
	}
	if st.Pets[0].Likes != "com" || st.Pets[0].Hates != "mathu" {
		t.Fatalf("tastes %s/%s", st.Pets[0].Likes, st.Pets[0].Hates)
	}
	return st
}

func TestStarvesWithoutFood(t *testing.T) {
	st := hatch(t)
	p := st.Pets[0]
	p.tick(24 * hour)
	if p.Dead || p.Hunger > 35 {
		t.Fatalf("after a day: dead=%v hunger=%.1f", p.Dead, p.Hunger)
	}
	if !p.tick(10*24*hour) || !p.Dead {
		t.Fatal("should die after ten days without food")
	}
	if p.DiedAt <= 24*hour || p.DiedAt >= 10*24*hour || p.Cause == "" {
		t.Fatalf("died at %d cause %q", p.DiedAt, p.Cause)
	}
}

func TestSurvivesWeekend(t *testing.T) {
	st := hatch(t)
	st.Pets[0].Hunger, st.Pets[0].Happy = 100, 100
	st.Pets[0].tick(64 * hour)
	if st.Pets[0].Dead {
		t.Fatal("a full pet must survive Friday 18:00 to Monday 08:00")
	}
}

func TestBagAndFood(t *testing.T) {
	st := hatch(t)
	if _, err := useItem(st, 0, "tui", 0, fixed(0.0, 0.9)); err != nil {
		t.Fatal(err)
	}
	if st.Inv["tui"] != startBags-1 || st.Inv["com"] != 1 {
		t.Fatalf("inventory %v", st.Inv)
	}
	st.Pets[0].Hunger = 40
	if _, err := useItem(st, 0, "com", 0, fixed(0.9)); err != nil {
		t.Fatal(err)
	}
	if st.Pets[0].Hunger != 65 || st.Inv["com"] != 0 {
		t.Fatalf("hunger %.0f inv %v", st.Pets[0].Hunger, st.Inv)
	}
	if _, err := useItem(st, 0, "com", 0, fixed(0.9)); err == nil {
		t.Fatal("using an item you do not have must fail")
	}
}

func TestSpicyTwiceHurts(t *testing.T) {
	st := hatch(t)
	st.Inv["ot"] = 2
	useItem(st, 0, "ot", 0, fixed(0.9))
	if st.Pets[0].Effect != "cay" {
		t.Fatalf("effect %q", st.Pets[0].Effect)
	}
	before := st.Pets[0].Health
	useItem(st, 0, "ot", 60000, fixed(0.9))
	if st.Pets[0].Health > before-12 {
		t.Fatalf("second chili should hurt: %.0f -> %.0f", before, st.Pets[0].Health)
	}
}

func TestMedicine(t *testing.T) {
	st := hatch(t)
	st.Inv["thuoc"] = 2
	useItem(st, 0, "thuoc", 0, fixed(0.9))
	if st.Pets[0].Health != 92 {
		t.Fatalf("medicine without illness should hurt, health %.0f", st.Pets[0].Health)
	}
	st.Pets[0].Sick = true
	useItem(st, 0, "thuoc", 0, fixed(0.9))
	if st.Pets[0].Sick {
		t.Fatal("medicine should cure")
	}
}

func TestPoisonCanKillAndRevive(t *testing.T) {
	st := hatch(t)
	st.Pets[0].Health = 30
	st.Inv["nam"], st.Inv["long"] = 1, 1
	msg, _ := useItem(st, 0, "nam", 5, fixed(0.1))
	if !st.Pets[0].Dead {
		t.Fatalf("poison at 30 health should kill: %s", msg)
	}
	if _, err := useItem(st, 0, "com", 5, fixed(0.9)); err == nil {
		t.Fatal("cannot feed a dead pet")
	}
	if _, err := useItem(st, 0, "long", 10, fixed(0.9)); err != nil || st.Pets[0].Dead {
		t.Fatalf("revive failed: %v", err)
	}
}

func TestReplaceDeadPetMakesGrave(t *testing.T) {
	st := hatch(t)
	if err := createPet(st, "B", nil, 0, fixed(0.1)); err == nil {
		t.Fatal("only one living pet")
	}
	st.Pets[0].tick(30 * 24 * hour)
	if err := createPet(st, "B", map[string]int{}, 0, fixed(0.1)); err != nil {
		t.Fatal(err)
	}
	if len(st.Graves) != 1 || st.Graves[0].Name != "Mèo Cá" {
		t.Fatalf("graves %v", st.Graves)
	}
	if err := createPet(newUserState(), "X", map[string]int{"head": 99}, 0, fixed(0.1)); err == nil {
		t.Fatal("invalid part must be rejected")
	}
}

func TestDropLimits(t *testing.T) {
	st := newUserState()
	now := time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC).UnixMilli()
	got := 0
	for i := 0; i < 200; i++ {
		if st.tryDrop(now+int64(i)*60000, fixed(0.0)) {
			got++
		}
	}
	if got != dropsPerDay {
		t.Fatalf("drops in one day = %d", got)
	}
	if st.Air != maxAir || st.Inv["tui"] != startBags+dropsPerDay-maxAir {
		t.Fatalf("air %d bags %d", st.Air, st.Inv["tui"])
	}
	for st.Air > 0 {
		if err := st.claimAir(fixed(0.0)); err != nil {
			t.Fatal(err)
		}
	}
	if st.Inv["tui"] != startBags+dropsPerDay || st.claimAir(fixed(0.0)) == nil {
		t.Fatalf("claimed bags %d", st.Inv["tui"])
	}
}

func TestTradeEscrow(t *testing.T) {
	a, b := newUserState(), newUserState()
	a.Inv["pho"] = 1
	b.Inv["thuoc"] = 1
	o := &offer{Give: map[string]int{"pho": 1}, Want: map[string]int{"thuoc": 1}}
	a.move(o.Give, -1)
	if err := settle(a, b, o); err != nil {
		t.Fatal(err)
	}
	if a.Inv["thuoc"] != 1 || a.Inv["pho"] != 0 || b.Inv["pho"] != 1 || b.Inv["thuoc"] != 0 {
		t.Fatalf("a=%v b=%v", a.Inv, b.Inv)
	}
	if err := settle(a, b, o); err == nil {
		t.Fatal("recipient no longer has the wanted item")
	}
	if validateBundle(map[string]int{"vang": 1}) == nil || validateBundle(map[string]int{"com": 0}) == nil {
		t.Fatal("bad bundle accepted")
	}
}

func TestWarningsOnce(t *testing.T) {
	st := hatch(t)
	st.Pets[0].Hunger = 10
	if len(warnings(st.Pets[0], false)) != 1 || len(warnings(st.Pets[0], false)) != 0 {
		t.Fatal("hungry warning should be sent once")
	}
}

func TestFullPetRefusesFood(t *testing.T) {
	st := hatch(t)
	st.Inv["banhmi"] = 1
	st.Pets[0].Hunger = 96
	if _, err := useItem(st, 0, "banhmi", 0, fixed(0.9)); err == nil || st.Inv["banhmi"] != 1 {
		t.Fatal("a full pet must refuse food without using it up")
	}
	st.Pets[0].Hunger = 90
	useItem(st, 0, "banhmi", 0, fixed(0.9))
	if st.Pets[0].Health < 88 || st.Pets[0].Dead {
		t.Fatalf("overeating should sting, not kill: %.0f", st.Pets[0].Health)
	}
}

func TestMigrateSinglePet(t *testing.T) {
	st := &userState{Pet: &pet{Name: "Cũ"}, Inv: map[string]int{}}
	st.normalize()
	st.normalize()
	if st.Pet != nil || len(st.Pets) != 1 || st.Slots != 1 || st.Inv["the_skin"] != 1 || st.Inv["the_slot"] != 1 {
		t.Fatalf("pets %d slots %d inv %v", len(st.Pets), st.Slots, st.Inv)
	}
}

func TestExtraSlotAndBury(t *testing.T) {
	st := hatch(t)
	if err := createPet(st, "B", map[string]int{}, 0, fixed(0.1)); err == nil {
		t.Fatal("one slot only")
	}
	if _, err := useItem(st, 0, "the_slot", 0, fixed(0.1)); err != nil || st.Slots != 2 {
		t.Fatalf("slot card: %v", err)
	}
	if err := createPet(st, "B", map[string]int{}, 0, fixed(0.1)); err != nil || len(st.Pets) != 2 {
		t.Fatalf("second pet: %v", err)
	}
	if bury(st, 0) == nil {
		t.Fatal("cannot bury a living pet")
	}
	st.Pets[0].tick(30 * 24 * hour)
	if err := bury(st, 0); err != nil || len(st.Pets) != 1 || st.Pets[0].Name != "B" || len(st.Graves) != 1 {
		t.Fatalf("bury: %v", err)
	}
}

func TestReskinKeepsStats(t *testing.T) {
	st := hatch(t)
	st.Pets[0].Hunger = 33
	if err := reskin(st, 0, "Rồng", map[string]int{"head": 5}); err != nil {
		t.Fatal(err)
	}
	p := st.Pets[0]
	if p.Name != "Rồng" || p.Parts["head"] != 5 || p.Hunger != 33 || st.Inv["the_skin"] != 0 {
		t.Fatalf("%+v inv %v", p, st.Inv)
	}
	if reskin(st, 0, "X", nil) == nil {
		t.Fatal("no card left")
	}
}

func TestInteractions(t *testing.T) {
	st := hatch(t)
	p := st.Pets[0]
	p.Happy = 50
	interact(st, 0, "vuot", 0)
	interact(st, 0, "vuot", 1000) // cooldown: no extra happiness
	if p.Happy != 58 {
		t.Fatalf("stroke happy %.0f", p.Happy)
	}
	interact(st, 0, "dam", 2000)
	if p.Health != 94 || p.Effect != "dau" {
		t.Fatalf("punch health %.0f effect %q", p.Health, p.Effect)
	}
	before := p.Happy
	interact(st, 0, "gai", 3000)
	if p.Happy != before {
		t.Fatal("an angry pet refuses kind actions except a hug")
	}
	interact(st, 0, "om", 4000)
	if p.Effect != "" || p.Happy <= before {
		t.Fatalf("hug should calm it: %q %.0f", p.Effect, p.Happy)
	}
	p.Health = 5
	if msg, _ := interact(st, 0, "dam", 5000); !p.Dead {
		t.Fatalf("punching a weak pet kills it: %s", msg)
	}
	if _, err := interact(st, 0, "xx", 0); err == nil {
		t.Fatal("unknown action")
	}
}

func TestAccessoryAndVacation(t *testing.T) {
	st := hatch(t)
	st.Inv["kinh"], st.Inv["no"], st.Inv["the_nghi"] = 1, 1, 1
	useItem(st, 0, "kinh", 0, fixed(0.1))
	useItem(st, 0, "no", 0, fixed(0.1))
	if st.Pets[0].Acc != "no" || st.Inv["kinh"] != 1 {
		t.Fatalf("acc %q inv %v", st.Pets[0].Acc, st.Inv)
	}
	if unequip(st, 0); st.Inv["no"] != 1 || st.Pets[0].Acc != "" {
		t.Fatal("unequip returns the item")
	}
	p := st.Pets[0]
	useItem(st, 0, "the_nghi", 0, fixed(0.1))
	p.tick(72 * hour)
	if p.Hunger != 70 {
		t.Fatalf("vacation should freeze hunger: %.1f", p.Hunger)
	}
	p.tick(76 * hour)
	if p.Hunger != 64 {
		t.Fatalf("after vacation hunger drops again: %.1f", p.Hunger)
	}
}

func TestSkillsAssigned(t *testing.T) {
	st := hatch(t)
	sk := st.Pets[0].Skills
	if len(sk) != 2 || sk[0] == sk[1] || skillByID[sk[0]] == nil || skillByID[sk[1]] == nil {
		t.Fatalf("skills %v", sk)
	}
	old := &userState{Pet: &pet{Name: "Cũ", Born: 42}}
	old.normalize()
	again := &userState{Pet: &pet{Name: "Cũ", Born: 42}}
	again.normalize()
	if len(old.Pets[0].Skills) != 2 || old.Pets[0].Skills[0] != again.Pets[0].Skills[0] {
		t.Fatal("old pets get stable skills")
	}
	st.Inv["sach"] = 1
	if _, err := useItem(st, 0, "sach", 0, fixed(0.7, 0.2)); err != nil || st.Inv["sach"] != 0 {
		t.Fatalf("skill book: %v", err)
	}
}

func TestMartialManualKeepsUnlockedSlotsAndRanks(t *testing.T) {
	st := newUserState()
	p := &pet{Name: "Galaxy", Lv: 16, Skills: []string{"hut", "khien", "bom"},
		SkillLv: map[string]int{"hut": 2, "khien": 2, "bom": 3}}
	st.Pets = []*pet{p}
	st.add("sach", 1)
	old := append([]string(nil), p.Skills...)
	msg, err := useItem(st, 0, "sach", 0, fixed(0.0, 0.2, 0.4, 0.6))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Skills) != 3 || p.Skills[2] != "bom" {
		t.Fatalf("book deleted an unlocked slot: before=%v after=%v", old, p.Skills)
	}
	if p.Skills[0] == old[0] && p.Skills[1] == old[1] {
		t.Fatalf("book did not change the starting pair: %v", p.Skills)
	}
	if p.skillLv(p.Skills[0]) != 2 || p.skillLv(p.Skills[1]) != 2 || p.skillLv("bom") != 3 {
		t.Fatalf("training was lost or attached to stale ids: skills=%v ranks=%v", p.Skills, p.SkillLv)
	}
	for id := range p.SkillLv {
		found := false
		for _, active := range p.Skills {
			found = found || id == active
		}
		if !found {
			t.Fatalf("stale rank %s survived: %v (%s)", id, p.SkillLv, msg)
		}
	}
	if st.Inv["sach"] != 0 {
		t.Fatalf("successful book not spent: %v", st.Inv)
	}
}

func TestMartialManualRefusalDoesNotSpend(t *testing.T) {
	st := newUserState()
	p := &pet{Name: "New", Skills: []string{"lua"}}
	st.Pets = []*pet{p}
	st.add("sach", 1)
	if _, err := useItem(st, 0, "sach", 0, fixed(0.2)); err == nil {
		t.Fatal("book accepted a pet without the two starting moves")
	}
	if st.Inv["sach"] != 1 || len(p.Skills) != 1 {
		t.Fatalf("refusal mutated state: inv=%v skills=%v", st.Inv, p.Skills)
	}
}

func TestPickSkill(t *testing.T) {
	always := fixed(0.99) // never the plain bite, last usable skill
	if got := pickSkill([]string{"lua", "khien"}, true, false, false, 100, 100, always); got != "lua" {
		t.Fatalf("shield already up, should attack: %q", got)
	}
	if got := pickSkill([]string{"khien"}, true, false, false, 100, 100, always); got != "" {
		t.Fatalf("only a useless shield left, should bite: %q", got)
	}
	if got := pickSkill([]string{"hoi", "an"}, false, true, false, 90, 100, always); got != "" {
		t.Fatalf("hidden and healthy, nothing useful: %q", got)
	}
	if got := pickSkill([]string{"set", "hoi"}, false, false, true, 40, 100, always); got != "hoi" {
		t.Fatalf("foe hidden, heal instead of wasting thunder: %q", got)
	}
	// A only ever re-raises its shield after B has hit it
	for i := 0; i < 50; i++ {
		res := fight(sideOf("a", &pet{Name: "A", Health: 100, Skills: []string{"lua", "khien"}}), sideOf("b", &pet{Name: "B", Health: 100, Skills: []string{"an"}}), seeded(int64(i)))
		up := false
		for k, tr := range res.Turns {
			switch {
			case tr.By == 0 && tr.Skill == "khien":
				if up {
					t.Fatalf("fight %d turn %d: shield raised while still up", i, k)
				}
				up = true
			case tr.By == 1 && k > 0 && tr.HP[0] < res.Turns[k-1].HP[0]:
				up = false
			}
		}
	}
}

func TestDuel(t *testing.T) {
	a, b := hatch(t), hatch(t)
	b.Pets[0].Born = 7
	wins := 0
	r := seeded(1)
	for i := 0; i < 200; i++ {
		res := fight(sideOf("a", a.Pets[0]), sideOf("b", b.Pets[0]), r)
		last := res.Turns[len(res.Turns)-1].HP
		if len(res.Turns) == 0 || len(res.Turns) > maxDuelTurns+16 || last[1-res.Winner] != 0 || last[res.Winner] <= 0 {
			t.Fatalf("winner must be the only pet left standing: %+v", res)
		}
		wins += 1 - res.Winner
	}
	if wins < 60 || wins > 140 {
		t.Fatalf("duels should go both ways, A won %d/200", wins)
	}
	res := duelResult{A: sideOf("a", a.Pets[0]), B: sideOf("b", b.Pets[0]), Winner: 0}
	bags := a.Inv["tui"]
	for i := 0; i < duelRewards+1; i++ {
		applyDuel(a, res, 0, 0, fixed(0.5))
	}
	if a.Inv["tui"] != bags+duelRewards || a.Pets[0].Wins != duelRewards+1 || len(a.Duels) != duelRewards+1 {
		t.Fatalf("rewards %d wins %d", a.Inv["tui"]-bags, a.Pets[0].Wins)
	}
	note := applyDuel(b, res, 1, 0, fixed(0.5, 0.9))
	if b.Pets[0].Dead || b.Pets[0].Health >= 100 || b.Pets[0].Effect != "dau" {
		t.Fatalf("loser hurt but alive: %s", note)
	}
	b.Pets[0].Health = 50
	note = applyDuel(b, res, 1, 0, fixed(0.5, 0.01))
	if !b.Pets[0].Dead {
		t.Fatalf("a crushing blow on a weak loser kills: %s", note)
	}
	if b.Pets[0].fit() == nil {
		t.Fatal("dead pets cannot duel")
	}
}

func TestBuryReturnsGearGemsAndAccessory(t *testing.T) {
	st := hatch(t)
	p := st.Pets[0]
	p.Dead = true
	p.Gear = map[string]string{"vukhi": "kiem", "giap": "giapda"}
	p.Sockets = map[string][]string{"vukhi": {"gem_atk_2"}, "giap": {"gem_def_1", "gem_hp_1"}}
	p.Acc = "no"
	if err := bury(st, 0); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]int{"kiem": 1, "giapda": 1, "gem_atk_2": 1, "gem_def_1": 1, "gem_hp_1": 1, "no": 1} {
		if st.Inv[id] != want {
			t.Fatalf("recovered %s = %d, want %d; bag %v", id, st.Inv[id], want, st.Inv)
		}
	}
}

func TestReplacingDeadPetReturnsEquipment(t *testing.T) {
	st := hatch(t)
	st.Pets[0].Dead = true
	st.Pets[0].Gear = map[string]string{"vukhi": "kiem"}
	st.Pets[0].Sockets = map[string][]string{"vukhi": {"gem_atk_1"}}
	if err := createPet(st, "Mới", map[string]int{}, hour, fixed(0.1)); err != nil {
		t.Fatal(err)
	}
	if len(st.Pets) != 1 || st.Pets[0].Name != "Mới" || st.Inv["kiem"] != 1 || st.Inv["gem_atk_1"] != 1 {
		t.Fatalf("replacement lost equipment: pets %+v bag %v", st.Pets, st.Inv)
	}
}

func TestDuelUpdatesArenaPointsForBothSides(t *testing.T) {
	a, b := huntPet(t), huntPet(t)
	a.Pets[0].Born, b.Pets[0].Born = 1, 2
	a.Pets[0].ArenaPoints, b.Pets[0].ArenaPoints = 1000, 1200
	res := duelResult{A: sideOf("a", a.Pets[0]), B: sideOf("b", b.Pets[0]), Winner: 0}
	applyDuel(a, res, 0, 1, fixed(.5, .5))
	applyDuel(b, res, 1, 1, fixed(.5, .5))
	if a.Pets[0].ArenaPoints <= 1020 {
		t.Fatalf("underdog win paid too little: %d", a.Pets[0].ArenaPoints)
	}
	if b.Pets[0].ArenaPoints >= 1180 {
		t.Fatalf("favourite loss cost too little: %d", b.Pets[0].ArenaPoints)
	}
}
