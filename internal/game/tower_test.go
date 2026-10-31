package game

import (
	"strings"
	"testing"
	"time"
)

// towerPet is a fighter strong enough to clear early floors against the fixed rng below.
func towerPet(t *testing.T) *userState {
	t.Helper()
	st := huntPet(t)
	p := st.Pets[0]
	p.Lv, p.Health, p.Hunger = 60, 100, 100
	p.Build = buildDef{Atk: 60, Def: 40, HP: 40}
	return st
}

func towerDay(d int) int64 {
	return time.Date(2026, 11, d, 9, 0, 0, 0, vnZone).UnixMilli()
}

func winRoll() rng { return fixed(0.0, 0.99, 0.0) }

// A level 1 pet sent at a floor in the forties loses on any roll, which is how these tests
// get a dependable defeat without betting on the dice.
func weakPet(t *testing.T) *userState {
	t.Helper()
	st := huntPet(t)
	p := st.Pets[0]
	p.Lv, p.Health, p.Hunger = 1, 100, 100
	st.TowerFloor = 44
	return st
}

func clearFloor(t *testing.T, st *userState, floor int, now int64) string {
	t.Helper()
	st.Pets[0].Health = 100
	duel, msg, err := tower(st, 0, floor, now, winRoll())
	if err != nil {
		t.Fatalf("floor %d: %v", floor, err)
	}
	if duel.Winner != 0 {
		t.Fatalf("floor %d was supposed to be a win: %s", floor, msg)
	}
	return msg
}

func TestTowerProgressRisesOneFloorAtATime(t *testing.T) {
	st := towerPet(t)
	now := towerDay(1)
	if st.TowerFloor != 0 {
		t.Fatalf("a new account starts below floor one: %d", st.TowerFloor)
	}
	clearFloor(t, st, 0, now) // 0 means "the next floor"
	if st.TowerFloor != 1 {
		t.Fatalf("clearing floor one should record it: %d", st.TowerFloor)
	}
	clearFloor(t, st, 2, now)
	if st.TowerFloor != 2 {
		t.Fatalf("progress stalled at %d", st.TowerFloor)
	}
	// Floors cannot be skipped.
	if _, _, err := tower(st, 0, 5, now, winRoll()); err == nil {
		t.Fatal("floor five should be locked behind floor three")
	}
}

func TestTowerReplayPaysNothing(t *testing.T) {
	st := towerPet(t)
	now := towerDay(1)
	clearFloor(t, st, 1, now)
	coin, inv, xp := st.Coin, len(st.Inv), st.Pets[0].XP
	if coin == 0 {
		t.Fatal("the first clear should pay gold")
	}

	msg := clearFloor(t, st, 1, now)
	if st.Coin != coin || st.Pets[0].XP != xp {
		t.Fatalf("a cleared floor paid again: coin %d->%d xp %d->%d", coin, st.Coin, xp, st.Pets[0].XP)
	}
	if len(st.Inv) != inv {
		t.Fatalf("a cleared floor dropped loot: %v", st.Inv)
	}
	if !strings.Contains(msg, "không có thưởng") {
		t.Fatalf("the replay should say so: %q", msg)
	}
	if st.TowerFloor != 1 {
		t.Fatalf("a replay must not move progress: %d", st.TowerFloor)
	}
}

func TestTowerDailyTriesCapAndReset(t *testing.T) {
	st := weakPet(t)
	now := towerDay(1)
	if st.towerLeft(now) != towerTriesDay {
		t.Fatalf("a fresh day starts with %d tries, got %d", towerTriesDay, st.towerLeft(now))
	}
	// Losing still burns an attempt, so ten failures exhaust the day.
	for i := 0; i < towerTriesDay; i++ {
		st.Pets[0].Health = 100
		duel, _, err := tower(st, 0, 45, now, winRoll())
		if err != nil {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
		if duel.Winner == 0 {
			t.Fatalf("attempt %d was supposed to lose", i+1)
		}
	}
	if st.towerLeft(now) != 0 {
		t.Fatalf("the day should be spent, %d left", st.towerLeft(now))
	}
	if _, _, err := tower(st, 0, 45, now, winRoll()); err == nil {
		t.Fatal("an eleventh attempt must be refused")
	}

	tomorrow := towerDay(2)
	if st.towerLeft(tomorrow) != towerTriesDay {
		t.Fatalf("the counter should reset overnight: %d", st.towerLeft(tomorrow))
	}
	st.Pets[0].Health = 100
	if _, _, err := tower(st, 0, 45, tomorrow, winRoll()); err != nil {
		t.Fatalf("the new day should allow a climb: %v", err)
	}
	if st.TowerToday != 1 || st.TowerDay != dayOf(tomorrow) {
		t.Fatalf("the counter did not roll over: today=%d day=%q", st.TowerToday, st.TowerDay)
	}
}

func TestTowerMilestoneFloorsPayASkillStone(t *testing.T) {
	st := towerPet(t)
	now := towerDay(1)
	st.TowerFloor = 9
	if towerIsBoss(9) || !towerIsBoss(10) {
		t.Fatal("floor ten is the boss, floor nine is not")
	}
	gems := st.Inv["ngoc"]
	msg := clearFloor(t, st, 10, now)
	if st.Inv["ngoc"] != gems+1 {
		t.Fatalf("floor ten should hand over one 💠: %d -> %d", gems, st.Inv["ngoc"])
	}
	if !strings.Contains(msg, "ngọc kỹ năng") {
		t.Fatalf("the message should name the stone: %q", msg)
	}
	// A plain floor gives none.
	gems = st.Inv["ngoc"]
	clearFloor(t, st, 11, now)
	if st.Inv["ngoc"] != gems {
		t.Fatalf("floor eleven paid a stone it should not: %d", st.Inv["ngoc"])
	}
	v := st.towerView(now)
	if len(v.Milestones) != 1 || v.Milestones[0] != 10 {
		t.Fatalf("the view should list the cleared milestone: %v", v.Milestones)
	}
}

func TestTowerLossHurtsButNeverKills(t *testing.T) {
	st := weakPet(t)
	now := towerDay(1)
	p := st.Pets[0]
	p.Health = 100
	duel, msg, err := tower(st, 0, 45, now, winRoll())
	if err != nil {
		t.Fatal(err)
	}
	if duel.Winner == 0 {
		t.Fatalf("this floor was supposed to be out of reach: %s", msg)
	}
	if p.Health != 100-towerLoseHurt {
		t.Fatalf("a loss costs %d health, got %.0f", towerLoseHurt, 100-p.Health)
	}
	if p.Dead {
		t.Fatal("a tower loss must never kill")
	}
	if st.TowerFloor != 44 {
		t.Fatalf("a loss must not advance the tower: %d", st.TowerFloor)
	}
	if !strings.Contains(msg, "Thua") {
		t.Fatalf("the message should report the defeat: %q", msg)
	}

	// The health floor matters most at the bottom: a pet that is just fit enough to enter
	// walks out alive rather than being zeroed by the 5-point loss.
	p.Health = 30
	if _, _, err := tower(st, 0, 45, now, winRoll()); err != nil {
		t.Fatal(err)
	}
	if p.Dead || p.Health <= 0 {
		t.Fatalf("the tower killed a pet: dead=%v health=%.0f", p.Dead, p.Health)
	}
	// Below the fitness bar the climb is refused outright, so no loss can reach zero.
	p.Health = 1
	if _, _, err := tower(st, 0, 45, now, winRoll()); err == nil {
		t.Fatal("fit() should stop a pet this weak from entering")
	}
}

func TestTowerFoeIsTheSameForEveryoneAndHarderUpTheTower(t *testing.T) {
	a, b := towerFoe(7), towerFoe(7)
	if a.Name != b.Name || a.Lv != b.Lv || a.HP != b.HP || a.Atk != b.Atk {
		t.Fatalf("floor seven is not deterministic: %+v vs %+v", a, b)
	}
	if towerFoe(20).HP <= towerFoe(10).HP || towerFoe(30).Lv <= towerFoe(20).Lv {
		t.Fatal("the tower should get harder as it rises")
	}
	// A boss floor is meaningfully stronger than its neighbour.
	if towerFoe(10).HP <= towerFoe(9).HP*1.2 {
		t.Fatalf("floor ten should be a step up: %.0f vs %.0f", towerFoe(10).HP, towerFoe(9).HP)
	}
	if towerBossNames[50] == "" || towerFoe(50).Name != towerBossNames[50] {
		t.Fatalf("the final boss should be named: %q", towerFoe(50).Name)
	}
}

func TestTowerAppliesTheSeasonBuff(t *testing.T) {
	st := towerPet(t)
	now := towerDay(1)
	plain, _, err := tower(st, 0, 1, now, winRoll())
	if err != nil {
		t.Fatal(err)
	}

	buffed := towerPet(t)
	buffed.SeasonBoostBPS, buffed.SeasonBoostUntil = seasonBoostCapBPS, now+hour
	got, _, err := tower(buffed, 0, 1, now, winRoll())
	if err != nil {
		t.Fatal(err)
	}
	if got.A.Atk <= plain.A.Atk || got.A.HP <= plain.A.HP {
		t.Fatalf("the season buff did not reach the tower: atk %.1f vs %.1f, hp %.1f vs %.1f",
			got.A.Atk, plain.A.Atk, got.A.HP, plain.A.HP)
	}
}

func TestTowerViewPreviewsTheNextFloorAndStopsAtTheTop(t *testing.T) {
	st := towerPet(t)
	now := towerDay(1)
	v := st.towerView(now)
	if v.Next != 1 || v.Foe == nil || v.Reward == nil {
		t.Fatalf("a fresh account should preview floor one: %+v", v)
	}
	if v.Foe.Name != towerFoe(1).Name || v.Foe.Power <= 0 {
		t.Fatalf("the preview must describe the real foe: %+v", v.Foe)
	}
	if v.Left != towerTriesDay || v.Max != towerFloors {
		t.Fatalf("view counters are wrong: %+v", v)
	}

	st.TowerFloor = towerFloors
	done := st.towerView(now)
	if done.Next != 0 || done.Foe != nil || done.Reward != nil {
		t.Fatalf("a finished tower has nothing left to preview: %+v", done)
	}
	if len(done.Milestones) != towerFloors/towerMilestone {
		t.Fatalf("every milestone should be listed: %v", done.Milestones)
	}
	if _, _, err := tower(st, 0, 0, now, winRoll()); err == nil {
		t.Fatal("there is no floor fifty-one")
	}
}

func TestTowerGoldObeysTheCrueltyCut(t *testing.T) {
	st := towerPet(t)
	now := towerDay(1)
	st.Cruel = now + hour
	clearFloor(t, st, 1, now)
	marked := st.Coin

	clean := towerPet(t)
	clearFloor(t, clean, 1, now)
	if marked >= clean.Coin {
		t.Fatalf("the cruelty mark should halve tower gold: %d vs %d", marked, clean.Coin)
	}
}

// ---- the manual that lets a player choose ----

func manualPet(t *testing.T) *userState {
	t.Helper()
	st := huntPet(t)
	st.add("sach", 1)
	p := st.Pets[0]
	p.Skills = []string{"lua", "hoi"}
	p.SkillLv = map[string]int{"lua": 4}
	return st
}

func TestSwapSkillTakesTheChosenMoveAndKeepsTheRank(t *testing.T) {
	st := manualPet(t)
	p := st.Pets[0]
	msg, err := swapSkill(st, 0, "lua", "set")
	if err != nil {
		t.Fatal(err)
	}
	if p.Skills[0] != "set" || p.Skills[1] != "hoi" {
		t.Fatalf("only the chosen slot should change: %v", p.Skills)
	}
	if p.skillLv("set") != 4 {
		t.Fatalf("the rank belongs to the slot, got %d", p.skillLv("set"))
	}
	if _, ok := p.SkillLv["lua"]; ok {
		t.Fatalf("the replaced skill kept a rank: %v", p.SkillLv)
	}
	if st.Inv["sach"] != 0 {
		t.Fatalf("the manual should be spent: %d left", st.Inv["sach"])
	}
	if !strings.Contains(msg, skillByID["set"].Name) {
		t.Fatalf("the message should name the new skill: %q", msg)
	}
}

func TestSwapSkillLeavesOtherSlotsAlone(t *testing.T) {
	st := manualPet(t)
	p := st.Pets[0]
	p.Skills = []string{"lua", "hoi", "khien", "an"}
	p.SkillLv = map[string]int{"lua": 2, "hoi": 3, "khien": 5, "an": 2}
	if _, err := swapSkill(st, 0, "hoi", "bom"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(p.Skills, ","); got != "lua,bom,khien,an" {
		t.Fatalf("slots moved: %q", got)
	}
	for id, want := range map[string]int{"lua": 2, "bom": 3, "khien": 5, "an": 2} {
		if p.skillLv(id) != want {
			t.Fatalf("%s rank %d, want %d", id, p.skillLv(id), want)
		}
	}
}

func TestSwapSkillRejectsBadChoicesAndSpendsNothing(t *testing.T) {
	for _, tc := range []struct{ name, from, to string }{
		{"a skill the pet does not have", "set", "bom"},
		{"a skill that does not exist", "lua", "khongco"},
		{"a skill the pet already knows", "lua", "hoi"},
		{"replacing a skill with itself", "lua", "lua"},
	} {
		st := manualPet(t)
		before := append([]string(nil), st.Pets[0].Skills...)
		if _, err := swapSkill(st, 0, tc.from, tc.to); err == nil {
			t.Fatalf("%s should be refused", tc.name)
		}
		if st.Inv["sach"] != 1 {
			t.Fatalf("%s: a refused swap must not spend the manual", tc.name)
		}
		if strings.Join(st.Pets[0].Skills, ",") != strings.Join(before, ",") {
			t.Fatalf("%s: a refused swap changed the skills: %v", tc.name, st.Pets[0].Skills)
		}
	}
}

func TestSwapSkillNeedsAManualAndALivingPet(t *testing.T) {
	st := manualPet(t)
	st.add("sach", -1)
	if _, err := swapSkill(st, 0, "lua", "set"); err == nil {
		t.Fatal("without a manual the swap must fail")
	}
	st.add("sach", 1)
	st.Pets[0].Dead = true
	if _, err := swapSkill(st, 0, "lua", "set"); err == nil {
		t.Fatal("a dead pet cannot study")
	}
	if st.Inv["sach"] != 1 {
		t.Fatalf("the manual was spent on a dead pet: %d", st.Inv["sach"])
	}
}

func TestOldRandomManualStillWorks(t *testing.T) {
	st := manualPet(t)
	// The legacy path is what a client built before the picker still calls.
	if _, err := useItem(st, 0, "sach", 0, seeded(5)); err != nil {
		t.Fatal(err)
	}
	if st.Inv["sach"] != 0 {
		t.Fatalf("the random reroll should still spend the manual: %d", st.Inv["sach"])
	}
	if len(st.Pets[0].Skills) != 2 {
		t.Fatalf("the reroll should keep the slot count: %v", st.Pets[0].Skills)
	}
}

func TestTowerAndManualTextIsLocalised(t *testing.T) {
	cases := []struct{ lang, in, contains string }{
		{"en", "hết lượt leo tháp hôm nay", "no tower attempts"},
		{"ja", "hết lượt leo tháp hôm nay", "挑戦回数"},
		{"en", "bạn đã chinh phục toàn bộ tháp", "conquered the whole tower"},
		{"ja", "tầng tháp không hợp lệ", "無効な階層"},
		{"en", "thú đã có kỹ năng này", "already knows"},
		{"ja", "không có kỹ năng này", "存在しません"},
		{"en", "Thua Lính gác tháp tầng 3 ở tầng 3, mất 5 sức khỏe.", "took 5 health damage"},
		{"ja", "Thua Lính gác tháp tầng 3 ở tầng 3, mất 5 sức khỏe.", "体力を5失った"},
		{"en", "Hạ Hộ Pháp Tầng Mười ở tầng 10, tầng đã qua nên không có thưởng.", "no reward"},
		{"en", "Hạ Lính gác tháp tầng 1, chinh phục tầng 1! +12 vàng và 🎁 Túi thức ăn.", "conquered floor 1"},
		{"ja", "Hạ Hộ Pháp Tầng Mười, chinh phục tầng 10! +30 vàng, 💠 Ngọc kỹ năng và 1 💠 ngọc kỹ năng.", "制覇"},
		{"en", "Mèo Cá đổi sang ⚡ Sấm sét, giữ nguyên cấp 4.", "keeping rank 4"},
		{"ja", "Mèo Cá đổi sang ⚡ Sấm sét, giữ nguyên cấp 4.", "ランク4を維持"},
	}
	for _, tc := range cases {
		got := localizeText(tc.lang, tc.in)
		if !strings.Contains(got, tc.contains) {
			t.Errorf("%s: %q -> %q, want %q", tc.lang, tc.in, got, tc.contains)
		}
	}
}
