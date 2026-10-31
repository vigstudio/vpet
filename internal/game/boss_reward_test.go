package game

import "testing"

// The complaint that started this: a geared pet hits for ~4,000 and a new one for ~275, so
// paying per damage point gave the same fifty blows fifteen times the reward. Effort should
// decide the payout far more than gear does.
func TestBossPayNarrowsTheGapBetweenStrongAndNewPets(t *testing.T) {
	const pool, fighters = 800_000, 20
	weak := bossGold(275*50, pool, fighters)    // a new pet, fifty blows
	strong := bossGold(4075*50, pool, fighters) // a fully geared pet, the same fifty blows

	if weak < bossGoldFloor {
		t.Fatalf("a real contribution paid %d, below the floor %d", weak, bossGoldFloor)
	}
	if ratio := float64(strong) / float64(weak); ratio > 4 {
		t.Fatalf("strong pet earns %.1fx the new one (%d vs %d); the gap is back", ratio, strong, weak)
	}
}

func TestBossGoldNeverExceedsItsCap(t *testing.T) {
	for _, pool := range []int{1, 500_000, 3_000_000} {
		for _, mine := range []int{1, pool / 2, pool, pool * 10} {
			if got := bossGold(mine, pool, 20); got > bossGoldCap {
				t.Fatalf("pool %d, damage %d paid %d, over the cap %d", pool, mine, got, bossGoldCap)
			}
		}
	}
}

func TestBossGoldRewardsMoreDamageWithoutRunningAway(t *testing.T) {
	const pool = 1_000_000
	last := 0
	for _, mine := range []int{1_000, 10_000, 50_000, 200_000} {
		got := bossGold(mine, pool, 20)
		if got < last {
			t.Fatalf("damage %d paid %d, less than the previous tier's %d", mine, got, last)
		}
		last = got
	}
	// Ten times the damage must not be ten times the pay.
	small, large := bossGold(20_000, pool, 20), bossGold(200_000, pool, 20)
	if float64(large) > float64(small)*4 {
		t.Fatalf("10x damage paid %dx gold (%d vs %d)", large/max(small, 1), large, small)
	}
}

func TestBossGoldIgnoresNothing(t *testing.T) {
	if got := bossGold(0, 1000, 20); got != 0 {
		t.Fatalf("no damage paid %d", got)
	}
	if got := bossGold(100, 0, 20); got != 0 {
		t.Fatalf("an empty pool paid %d", got)
	}
}

// Crates are read against the pool, which grows with the number of fighters. Before this the
// thresholds were fixed damage numbers, so a busy day quietly demoted everyone.
func TestBossCratesDoNotShrinkOnABusyDay(t *testing.T) {
	quiet, busy := 800_000, 2_900_000 // 20 fighters versus 160
	if got, want := bossCrates(quiet/20, quiet, 20), bossCrates(busy/160, busy, 160); got != want {
		t.Fatalf("an equal share paid %d crates on a quiet day and %d on a busy one", got, want)
	}
}

func TestBossCratesRiseWithShare(t *testing.T) {
	const pool, fighters = 1_000_000, 20
	last := 0
	for _, share := range []float64{0.001, 0.02, 0.05, 0.2} {
		got := bossCrates(int(float64(pool)*share), pool, fighters)
		if got < last {
			t.Fatalf("share %.3f paid %d crates, fewer than %d", share, got, last)
		}
		if got < 1 || got > 4 {
			t.Fatalf("share %.3f paid %d crates, outside 1..4", share, got)
		}
		last = got
	}
	if bossCrates(0, pool, fighters) != 0 {
		t.Fatal("no damage still paid a crate")
	}
}

// Boss gold used to be added straight to the balance, walking around the economy entirely:
// a long day of hitting paid tens of thousands of coins against a 70-coin daily cap.
func TestBossGoldObeysItsOwnDailyCap(t *testing.T) {
	st := &userState{Inv: map[string]int{}}
	now := int64(1_700_000_000_000)

	total := 0
	for range 20 {
		total += st.earnBoss(bossGoldCap, now)
	}
	if total != bossCoinsDay {
		t.Fatalf("paid %d across the day, want the cap %d", total, bossCoinsDay)
	}
	if st.Coin != bossCoinsDay {
		t.Fatalf("balance %d, want %d", st.Coin, bossCoinsDay)
	}

	// Activity gold is a separate budget: fighting the boss must not consume the coins a
	// player earns by looking after their pet.
	if st.CoinToday != 0 {
		t.Fatalf("boss gold consumed %d of the activity allowance", st.CoinToday)
	}

	tomorrow := now + 24*60*60*1000
	if got := st.earnBoss(bossGoldCap, tomorrow); got == 0 {
		t.Fatal("the cap did not reset the next day")
	}
}

func TestEarnBossRefusesNothing(t *testing.T) {
	st := &userState{Inv: map[string]int{}}
	if got := st.earnBoss(0, 1); got != 0 {
		t.Fatalf("zero paid %d", got)
	}
	if got := st.earnBoss(-50, 1); got != 0 {
		t.Fatalf("a negative amount paid %d", got)
	}
	if st.Coin != 0 {
		t.Fatalf("balance moved to %d", st.Coin)
	}
}

// The whole point of the elastic pool is that turning up late still leaves a fight worth
// joining, and the reward has to agree: an equal share pays equally whatever the crowd size.
func TestEqualEffortPaysEquallyWhateverTheCrowd(t *testing.T) {
	small := bossGold(575_000/5, 575_000, 5)         // five fighters
	large := bossGold(2_900_000/160, 2_900_000, 160) // a hundred and sixty
	if small == large {
		return // identical shares, identical pay
	}
	if diff := float64(small-large) / float64(max(large, 1)); diff > 0.25 || diff < -0.25 {
		t.Fatalf("a 1/N share paid %d with five fighters and %d with 160", small, large)
	}
}

func TestSoulEssenceBossDropRequiresKillContributionAndLowRoll(t *testing.T) {
	for _, tc := range []struct {
		hp, mine int
		roll     float64
		want     int
	}{{0, 300, 0.149, 1}, {0, 300, 0.15, 0}, {1, 300, 0, 0}, {0, 299, 0, 0}, {0, 0, 0, 0}} {
		st := newUserState()
		b := &worldBoss{HP: tc.hp, Max: 1000, Damage: map[string]int{"u": tc.mine, "v": 1000 - tc.mine}}
		bossReward(st, b, "u", 1700000000000, func() float64 { return tc.roll })
		if st.Inv["tinhhon"] != tc.want {
			t.Fatalf("case %+v: %v", tc, st.Inv)
		}
	}
}
