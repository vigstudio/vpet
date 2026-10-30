package game

import (
	"testing"
	"time"
)

func atDay(y int, m time.Month, d int) int64 {
	return time.Date(y, m, d, 9, 0, 0, 0, vnZone).UnixMilli()
}

func TestCheckinPaysOncePerDay(t *testing.T) {
	st := newUserState()
	now := atDay(2026, time.October, 7)
	bags := st.Inv["tui"]
	if _, err := claimCheckin(st, now); err != nil {
		t.Fatal(err)
	}
	if st.Coin != checkinCoin || st.Streak != 1 || st.Inv["tui_y"] != 1 {
		t.Fatalf("first claim wrong: coin=%d streak=%d inv=%v", st.Coin, st.Streak, st.Inv)
	}
	if st.Inv["tui"] != bags {
		t.Fatalf("day 7 should pay its own prize, not a bag: %d", st.Inv["tui"])
	}
	if _, err := claimCheckin(st, now+3600_000); err == nil {
		t.Fatal("claimed twice in one day")
	}
	if st.Coin != checkinCoin {
		t.Fatal("refused claim still paid")
	}
}

func TestStreakBreaksOnAMissedDay(t *testing.T) {
	st := newUserState()
	for d := 1; d <= 3; d++ {
		if _, err := claimCheckin(st, atDay(2026, time.October, d)); err != nil {
			t.Fatal(err)
		}
	}
	if st.Streak != 3 || st.BestStreak != 3 {
		t.Fatalf("streak %d best %d", st.Streak, st.BestStreak)
	}
	if _, err := claimCheckin(st, atDay(2026, time.October, 5)); err != nil {
		t.Fatal(err)
	}
	if st.Streak != 1 {
		t.Fatalf("a missed day must reset the streak: %d", st.Streak)
	}
	if st.BestStreak != 3 {
		t.Fatalf("best streak should stay: %d", st.BestStreak)
	}
}

func TestMilestonesPayExactlyAtTenTwentyThirty(t *testing.T) {
	st := newUserState()
	paid := map[int]int{}
	for d := 1; d <= 30; d++ {
		before := st.Coin
		if _, err := claimCheckin(st, atDay(2026, time.October, d)); err != nil {
			t.Fatal(err)
		}
		paid[d] = st.Coin - before
	}
	for _, d := range []int{10, 20, 30} {
		if paid[d] != checkinCoin+streakBonusCoin {
			t.Fatalf("day %d paid %d, want %d", d, paid[d], checkinCoin+streakBonusCoin)
		}
	}
	for _, d := range []int{9, 11, 19, 21, 29} {
		if paid[d] != checkinCoin {
			t.Fatalf("day %d paid a milestone bonus it should not: %d", d, paid[d])
		}
	}
	if st.Inv["ngoc"] != 1 || st.Inv["the_nang_luong"] != 1 || st.Inv["tinh_the_tiem_nang"] != 1 {
		t.Fatalf("milestone items missing: %v", st.Inv)
	}
}

func TestCalendarCoversTheMonthAndMarksToday(t *testing.T) {
	st := newUserState()
	now := atDay(2026, time.February, 9)
	v := st.checkinView(now)
	if len(v.Days) != 28 || v.Month != "2026-02" {
		t.Fatalf("february calendar: %d days, month %q", len(v.Days), v.Month)
	}
	if v.Claimed {
		t.Fatal("nothing claimed yet")
	}
	for _, d := range v.Days {
		if itemByID[d.Item] == nil {
			t.Fatalf("day %d offers an unknown item %q", d.Day, d.Item)
		}
		if d.Today != (d.Day == 9) || d.Past != (d.Day < 9) {
			t.Fatalf("day %d flagged wrong: %+v", d.Day, d)
		}
	}
	if _, err := claimCheckin(st, now); err != nil {
		t.Fatal(err)
	}
	v = st.checkinView(now)
	if !v.Claimed || v.Streak != 1 || !v.Days[8].Claimed {
		t.Fatalf("claimed calendar: %+v", v)
	}
	if len(v.Milestones) != 3 || v.Milestones[0].Days != 10 {
		t.Fatalf("milestones should count up from the nearest: %+v", v.Milestones)
	}
	leap := st.checkinView(atDay(2024, time.February, 1))
	if len(leap.Days) != 29 {
		t.Fatalf("leap february: %d days", len(leap.Days))
	}
}
