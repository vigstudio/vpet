package game

import (
	"fmt"
	"testing"
)

// Do the stats actually decide fights? Each case runs many duels between a baseline pet and
// one that differs in a single stat, and prints the win rate. A stat that moves the rate
// barely at all is decoration, not a choice.
func winRate(t *testing.T, mutate func(*pet), runs int) float64 {
	t.Helper()
	wins := 0
	for seed := 0; seed < runs; seed++ {
		a := huntPet(t).Pets[0]
		b := huntPet(t).Pets[0]
		a.Lv, b.Lv = 20, 20
		a.Skills = []string{"lua", "cuoc"}
		b.Skills = []string{"lua", "cuoc"}
		a.SkillLv = map[string]int{"lua": 3, "cuoc": 3}
		b.SkillLv = map[string]int{"lua": 3, "cuoc": 3}
		mutate(a)
		res := fight(sideOf("a", a), sideOf("b", b), seeded(int64(seed)))
		if res.Winner == 0 {
			wins++
		}
	}
	return float64(wins) / float64(runs) * 100
}

func TestStatsActuallyDecideFights(t *testing.T) {
	const runs = 400
	base := winRate(t, func(p *pet) {}, runs)
	t.Logf("identical pets: %.1f%% (sanity: should sit near 50)", base)
	if base < 40 || base > 60 {
		t.Fatalf("identical pets are not an even match: %.1f%%", base)
	}

	cases := []struct {
		name   string
		mutate func(*pet)
		min    float64 // the stat must be worth at least this win rate to count as real
	}{
		{"20 điểm Công", func(p *pet) { p.Build.Atk = 20 }, 60},
		{"20 điểm Thủ", func(p *pet) { p.Build.Def = 20 }, 60},
		{"20 điểm Sinh lực", func(p *pet) { p.Build.HP = 20 }, 60},
		{"trang bị hoàng kim toàn thân", func(p *pet) {
			p.Gear = map[string]string{"vukhi": "gaythan", "giap": "longbao", "mu": "vuongmien",
				"gang": "gangrong", "giay": "hiabay", "nhan": "nhanrong", "khien": "khienbac", "aokhoac": "aosao"}
		}, 75},
		{"ba viên ngọc công cấp 4", func(p *pet) {
			p.Gear = map[string]string{"vukhi": "kiem"}
			p.Sockets = map[string][]string{"vukhi": {"gem_atk_4", "gem_atk_4", "gem_atk_4"}}
		}, 58},
		{"hơn 10 cấp", func(p *pet) { p.Lv = 30 }, 55},
	}
	for _, c := range cases {
		got := winRate(t, c.mutate, runs)
		t.Logf("%-32s %.1f%% thắng", c.name, got)
		if got < c.min {
			t.Errorf("%s chỉ thắng %.1f%%, dưới mức %.0f%% — chỉ số này gần như vô dụng", c.name, got, c.min)
		}
	}
}

// A title is worn, not stacked, so the worst case is one rank-5 title on an already fully
// invested pet. It has to be worth wearing without turning Hard or the boss into a formality.
func TestTopTitleIsWorthWearingWithoutTrivialisingContent(t *testing.T) {
	const runs = 400
	full := func(p *pet) {
		p.Build = buildDef{Atk: 20, Def: 12, HP: 12}
		p.Gear = map[string]string{"vukhi": "gaythan", "giap": "longbao", "mu": "vuongmien",
			"gang": "gangrong", "giay": "hiabay", "nhan": "nhanrong"}
	}
	// Both sides are equipped identically, so the title is the only difference between them.
	// winRate only mutates one side, which would measure the gear instead.
	titleDuel := func(id string) float64 {
		wins := 0
		for seed := 0; seed < runs; seed++ {
			a := huntPet(t).Pets[0]
			b := huntPet(t).Pets[0]
			for _, p := range []*pet{a, b} {
				p.Lv = 20
				p.Skills = []string{"lua", "cuoc"}
				p.SkillLv = map[string]int{"lua": 3, "cuoc": 3}
				full(p)
			}
			// a has the record the title asks for and wears it; b has the same record and no title
			a.Hunts, a.Wild, a.Wins, a.Gold, a.ArenaPoints = 900, 1600, 400, 150, 2500
			a.Campaign = map[string][3]int{"vuon_1": {162, 0, 0}}
			a.Lv = 400
			b.Lv = 400
			a.Title = id
			if titleOf(a) == (gearDef{}) {
				t.Fatalf("%s did not unlock for a maxed record", id)
			}
			if fight(sideOf("a", a), sideOf("b", b), seeded(int64(seed))).Winner == 0 {
				wins++
			}
		}
		return float64(wins) / float64(runs) * 100
	}
	for _, track := range []string{"duel", "hunt", "wild", "lv", "star", "gold", "arena"} {
		id := fmt.Sprintf("t_%s%d", track, titleTiers)
		d := titleByID[id]
		got := titleDuel(id)
		t.Logf("%-12s %.1f%% thắng (⚔️%.1f 🛡️%.1f ❤️%.0f)", id, got, d.Atk, d.Def, d.HP)
		if got <= 52 {
			t.Errorf("%s only wins %.1f%%: nobody would bother equipping it", id, got)
		}
		if got >= 75 {
			t.Errorf("%s wins %.1f%%: one title should not decide a duel on its own", id, got)
		}
	}

	// and it must not flatten PvE either: a bare pet wearing the best title it could not
	// possibly have earned is the ceiling check that matters for Hard.
	rate := func(mutate func(*pet), difficulty int) float64 {
		wins := 0
		for seed := 0; seed < 200; seed++ {
			st := huntPet(t)
			q := st.Pets[0]
			q.Lv, q.Health, q.Hunger = 15, 100, 100
			q.Skills = []string{"lua", "cuoc"}
			q.SkillLv = map[string]int{"lua": 2, "cuoc": 2}
			q.Energy, q.EnergyAt = 999, 1
			q.Campaign = map[string][3]int{"vuon_1": {1, 1, 1}, "vuon_2": {1, 1, 1}, "vuon_3": {1, 1, 1}, "vuon_4": {1, 1, 0}}
			mutate(q)
			if r, err := campaign(st, 0, "vuon", 4, difficulty, 1, seeded(int64(seed))); err == nil && r.Won {
				wins++
			}
		}
		return float64(wins) / 200 * 100
	}
	bareHard := rate(func(p *pet) {}, 2)
	titledHard := rate(func(p *pet) {
		p.Wins = 400
		p.Title = "t_duel5"
	}, 2)
	t.Logf("ải Khó: trắng tay %.0f%% · chỉ thêm danh hiệu %.0f%%", bareHard, titledHard)
	if titledHard > 35 {
		t.Errorf("a title alone clears Hard %.0f%% of the time; investment stops mattering", titledHard)
	}
}

// Speed is the newest stat and the easiest to get wrong, so it gets its own measurement:
// how often does the faster pet take an extra turn, and does that show up in wins?
func TestSpeedProducesExtraTurns(t *testing.T) {
	fast := huntPet(t).Pets[0]
	slow := huntPet(t).Pets[0]
	fast.Lv, slow.Lv = 30, 30
	fast.Build.Atk = 30
	fast.Gear = map[string]string{"gang": "gangrong", "giay": "hiabay"}
	fs, ss := sideOf("f", fast), sideOf("s", slow)
	chance := extraTurnChance(fs.Speed, ss.Speed)
	t.Logf("tốc đánh %.1f vs %.1f → %.0f%% cơ hội đánh thêm lượt", fs.Speed, ss.Speed, chance*100)
	if chance <= 0.05 {
		t.Fatalf("a dedicated speed build only reaches %.1f%%, which no player would notice", chance*100)
	}

	// and it must show up as real extra turns in a real fight
	doubles, total := 0, 0
	for seed := 0; seed < 200; seed++ {
		res := fight(fs, ss, seeded(int64(seed)))
		for i := 1; i < len(res.Turns); i++ {
			total++
			if res.Turns[i].By == res.Turns[i-1].By {
				doubles++
			}
		}
	}
	rate := float64(doubles) / float64(total) * 100
	t.Logf("lượt đánh liên tiếp thực tế: %.1f%% tổng số lượt", rate)
	if rate < 3 {
		t.Fatalf("speed never translates into extra turns: %.1f%%", rate)
	}
}

// A stat sheet that disagrees with the fight is worse than no sheet, so every number the
// panel prints is checked against what sideOf hands the duel.
func TestStatSheetMatchesTheDuel(t *testing.T) {
	p := huntPet(t).Pets[0]
	p.Lv = 25
	p.Build = buildDef{Atk: 7, Def: 4, HP: 6}
	p.Gear = map[string]string{"vukhi": "kiem", "giap": "giapda", "gang": "gangthep"}
	p.Sockets = map[string][]string{"vukhi": {"gem_atk_2"}}
	sheet := p.statsOf()
	side := sideOf("", p)
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"sát thương", sheet.Atk, side.Atk},
		{"giáp", sheet.Def, side.Def},
		{"máu đấu", sheet.HP, side.HP},
	} {
		if fmt.Sprintf("%.4f", c.got) != fmt.Sprintf("%.4f", c.want) {
			t.Fatalf("bảng chỉ số lệch với trận đấu ở %s: %v vs %v", c.name, c.got, c.want)
		}
	}
}

// Duels proved the stats matter. The complaint was about the game as played, which is mostly
// PvE, so this measures whether investment changes campaign and wild outcomes too: if the
// opponent scales with the player, the investment cancels out and the stats feel inert.
func TestInvestmentChangesPvEOutcomes(t *testing.T) {
	run := func(mutate func(*pet)) (float64, float64) {
		campWins, wildWins := 0, 0
		const runs = 150
		for seed := 0; seed < runs; seed++ {
			st := huntPet(t)
			p := st.Pets[0]
			p.Lv, p.Health, p.Hunger = 20, 100, 100
			p.Skills = []string{"lua", "cuoc"}
			p.SkillLv = map[string]int{"lua": 3, "cuoc": 3}
			p.Energy, p.EnergyAt = 999, 1
			mutate(p)
			if r, err := campaign(st, 0, "vuon", 1, 0, 1, seeded(int64(seed))); err == nil && r.Won {
				campWins++
			}
			st2 := huntPet(t)
			q := st2.Pets[0]
			q.Lv, q.Health, q.Hunger = 20, 100, 100
			q.Skills = []string{"lua", "cuoc"}
			q.SkillLv = map[string]int{"lua": 3, "cuoc": 3}
			q.Energy, q.EnergyAt = 999, 1
			mutate(q)
			if r, err := encounter(st2, 0, "rung", 1, seeded(int64(seed))); err == nil && r.Won {
				wildWins++
			}
		}
		return float64(campWins) / runs * 100, float64(wildWins) / runs * 100
	}

	bareC, bareW := run(func(p *pet) {})
	t.Logf("thú trắng tay:        ải %.1f%% · thú hoang %.1f%%", bareC, bareW)

	builtC, builtW := run(func(p *pet) {
		p.Build = buildDef{Atk: 20, Def: 10, HP: 10}
		p.Gear = map[string]string{"vukhi": "gaythan", "giap": "longbao", "mu": "vuongmien",
			"gang": "gangrong", "giay": "hiabay", "nhan": "nhanrong"}
	})
	t.Logf("đầu tư đầy đủ:        ải %.1f%% · thú hoang %.1f%%", builtC, builtW)

	// A bare pet should struggle somewhere and an invested one should not. If both sit at
	// the same rate, the investment bought nothing and the stats are decoration.
	if builtW-bareW < 10 {
		t.Errorf("đầu tư gần như không đổi kết quả thú hoang: %+.1f điểm", builtW-bareW)
	}
	if builtC < 80 {
		t.Errorf("thú đầu tư vẫn chỉ qua ải %.1f%%", builtC)
	}
}

// The campaign has to reward investment without walling out a new player. This measures the
// same stage at three investment levels across the three difficulties, so a regression that
// makes Easy impossible or Hard trivial fails the build instead of reaching players.
func TestCampaignRewardsInvestment(t *testing.T) {
	build := func(kind string) func(*pet) {
		return func(p *pet) {
			switch kind {
			case "bare":
			case "some":
				p.Build = buildDef{Atk: 8, Def: 4, HP: 5}
				p.Gear = map[string]string{"vukhi": "kiem", "giap": "giapda", "mu": "muda"}
			case "full":
				p.Build = buildDef{Atk: 20, Def: 12, HP: 12}
				p.Gear = map[string]string{"vukhi": "gaythan", "giap": "longbao", "mu": "vuongmien",
					"gang": "gangrong", "giay": "hiabay", "nhan": "nhanrong"}
				p.Sockets = map[string][]string{"vukhi": {"gem_atk_3", "gem_atk_3"}}
			}
		}
	}
	rate := func(mutate func(*pet), difficulty int) float64 {
		const runs = 200
		wins := 0
		for seed := 0; seed < runs; seed++ {
			st := huntPet(t)
			q := st.Pets[0]
			q.Lv, q.Health, q.Hunger = 15, 100, 100
			q.Skills = []string{"lua", "cuoc"}
			q.SkillLv = map[string]int{"lua": 2, "cuoc": 2}
			q.Energy, q.EnergyAt = 999, 1
			q.Campaign = map[string][3]int{
				"vuon_1": {1, 1, 1}, "vuon_2": {1, 1, 1}, "vuon_3": {1, 1, 1},
				"vuon_4": {1, 1, 0}, // cleared Easy and Medium on this stage, so Hard is open
			}
			mutate(q)
			if r, err := campaign(st, 0, "vuon", 4, difficulty, 1, seeded(int64(seed))); err == nil && r.Won {
				wins++
			}
		}
		return float64(wins) / runs * 100
	}

	names := []string{"Dễ", "Vừa", "Khó"}
	got := map[string][]float64{}
	for _, kind := range []string{"bare", "some", "full"} {
		for d := 0; d < 3; d++ {
			got[kind] = append(got[kind], rate(build(kind), d))
		}
		t.Logf("%-5s → Dễ %.0f%% · Vừa %.0f%% · Khó %.0f%%", kind, got[kind][0], got[kind][1], got[kind][2])
	}

	// a new player must be able to start
	if got["bare"][0] < 40 {
		t.Errorf("thú trắng tay chỉ qua ải Dễ %.0f%%, người mới sẽ bị chặn", got["bare"][0])
	}
	// but bare must not clear the hardest tier
	if got["bare"][2] > 35 {
		t.Errorf("thú trắng tay qua ải Khó %.0f%%, đầu tư thành vô nghĩa", got["bare"][2])
	}
	// investment must be the thing that opens Hard
	if got["full"][2]-got["bare"][2] < 35 {
		t.Errorf("đầu tư chỉ thêm %.0f điểm ở độ Khó", got["full"][2]-got["bare"][2])
	}
	// the three tiers must differ for a mid-game pet
	for d := 1; d < 3; d++ {
		if got["some"][d-1]-got["some"][d] < 10 {
			t.Errorf("độ %s và %s gần như giống nhau với thú tầm trung: %.0f%% vs %.0f%%",
				names[d-1], names[d], got["some"][d-1], got["some"][d])
		}
	}
}
