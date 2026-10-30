package game

import (
	"testing"
	"time"
)

func TestRewardActivityUsesGroupCooldownAndDailyCap(t *testing.T) {
	now := time.Date(2026, 10, 30, 9, 0, 0, 0, vnZone).UnixMilli()
	engine := &Engine{random: fixed(0)}
	st := newUserState()

	engine.rewardActivity(st, "/interact", now)
	if st.Coin != coinPerAction || st.Drops != 1 {
		t.Fatalf("first care reward: coin=%d drops=%d", st.Coin, st.Drops)
	}
	engine.rewardActivity(st, "/use", now+coinCooldown.Milliseconds()-1)
	if st.Coin != coinPerAction {
		t.Fatalf("same group farm paid again: coin=%d", st.Coin)
	}
	engine.rewardActivity(st, "/hunt", now+coinCooldown.Milliseconds())
	if st.Coin != coinPerAction*2 {
		t.Fatalf("independent adventure group did not pay: coin=%d", st.Coin)
	}

	for i := 2; i < 100; i++ {
		at := now + int64(i)*coinCooldown.Milliseconds()
		engine.rewardActivity(st, "/hunt", at)
	}
	if st.Coin != coinsPerDay || st.CoinToday != coinsPerDay {
		t.Fatalf("daily cap exceeded or missed: coin=%d today=%d", st.Coin, st.CoinToday)
	}
	if st.Drops != dropsPerDay {
		t.Fatalf("drop cap mismatch: got=%d want=%d", st.Drops, dropsPerDay)
	}
}

func TestRewardActivityIgnoresNonRewardingActions(t *testing.T) {
	engine := &Engine{random: fixed(0)}
	st := newUserState()
	engine.rewardActivity(st, "/buy", time.Now().UnixMilli())
	if st.Coin != 0 || st.Drops != 0 || len(st.ActivityAt) != 0 {
		t.Fatalf("shop action rewarded: coin=%d drops=%d groups=%v", st.Coin, st.Drops, st.ActivityAt)
	}
}
