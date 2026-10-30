package game

import (
	"strings"
	"testing"
	"time"
)

const aDay = 24 * time.Hour

func TestActivityCoinsAreCappedPerDay(t *testing.T) {
	st := newUserState()
	now := time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC).UnixMilli()
	if got := st.earn(now, false); got != coinPerAction {
		t.Fatalf("first activity paid %d", got)
	}
	if got := st.earn(now+1000, false); got != 0 {
		t.Fatalf("activity during cooldown paid %d", got)
	}
	if got := st.earn(now+coinCooldown.Milliseconds(), true); got != coinPerAction+coinBonus {
		t.Fatalf("bonus activity paid %d", got)
	}
	for i := 2; i < 200; i++ {
		st.earn(now+int64(i)*coinCooldown.Milliseconds(), true)
	}
	if st.Coin != coinsPerDay || st.CoinToday != coinsPerDay {
		t.Fatalf("daily cap broken: coin %d today %d", st.Coin, st.CoinToday)
	}
	st.earn(now+aDay.Milliseconds(), false)
	if st.Coin != coinsPerDay+coinPerAction {
		t.Fatalf("the cap should reset the next day: %d", st.Coin)
	}
}

func TestShopRotatesDailyAndStaysStable(t *testing.T) {
	a := shopStock("2026-10-23")
	if len(a) != shopSize {
		t.Fatalf("shelf size %d", len(a))
	}
	if len(shopStock("2026-10-23")) != shopSize || shopStock("2026-10-23")[0] != a[0] {
		t.Fatal("the same day must show the same shop")
	}
	same := 0
	b := shopStock("2026-10-24")
	for i := range a {
		if a[i].Item == b[i].Item {
			same++
		}
	}
	if same == shopSize {
		t.Fatal("the shop should change overnight")
	}
	seen := map[string]bool{}
	for _, o := range a {
		if seen[o.Item] || o.Price <= 0 || o.Qty < 1 || itemByID[o.Item] == nil {
			t.Fatalf("bad offer %+v", o)
		}
		if itemByID[o.Item].Kind == "food" || itemByID[o.Item].Rarity == 0 {
			t.Fatalf("rotating shelf must contain only rare non-food goods: %+v", o)
		}
		seen[o.Item] = true
	}
}

func TestBuyingSpendsCoinsAndEmptiesTheShelf(t *testing.T) {
	st := newUserState()
	now := time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC).UnixMilli()
	var o shopOffer
	for _, offer := range st.stockFor(now) {
		if !offer.Unlimited {
			o = offer
			break
		}
	}
	if _, err := buy(st, o.Item, 1, now); err == nil {
		t.Fatal("no coins, no goods")
	}
	st.Coin = o.Price * (o.Qty + 1)
	if _, err := buy(st, o.Item, o.Qty, now); err != nil {
		t.Fatal(err)
	}
	if st.Inv[o.Item] < o.Qty || st.Coin != o.Price {
		t.Fatalf("inv %d coin %d", st.Inv[o.Item], st.Coin)
	}
	if _, err := buy(st, o.Item, 1, now); err == nil {
		t.Fatal("the shelf is empty until tomorrow")
	}
	for _, offer := range st.stockFor(now) {
		if offer.Item == o.Item && offer.Left != 0 {
			t.Fatalf("left %d", offer.Left)
		}
	}
	for _, offer := range st.stockFor(now + aDay.Milliseconds()) {
		if !offer.Unlimited && offer.Left == 0 {
			t.Fatal("tomorrow's rotating goods must be stocked")
		}
	}
	if _, err := buy(st, "khongco", 1, now); err == nil {
		t.Fatal("cannot buy what is not on sale")
	}
}

func TestFoodCounterIsCompleteAndUnlimited(t *testing.T) {
	foods := foodStock()
	want := 0
	for _, it := range catalog {
		if it.Kind == "food" {
			want++
		}
	}
	if len(foods) != want || want < 10 {
		t.Fatalf("food counter has %d/%d items", len(foods), want)
	}
	for _, o := range foods {
		if !o.Unlimited || itemByID[o.Item].Kind != "food" || o.Price <= 0 {
			t.Fatalf("bad permanent food offer: %+v", o)
		}
	}

	st := newUserState()
	now := time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC).UnixMilli()
	food := foods[0]
	st.Coin = food.Price * maxBuy * 2
	before := st.Inv[food.Item]
	if _, err := buy(st, food.Item, maxBuy, now); err != nil {
		t.Fatal(err)
	}
	if _, err := buy(st, food.Item, maxBuy, now); err != nil {
		t.Fatalf("permanent food ran out: %v", err)
	}
	if st.Inv[food.Item] != before+maxBuy*2 || st.Bought[food.Item] != 0 {
		t.Fatalf("unlimited food was tracked as finite: inv=%d bought=%d", st.Inv[food.Item], st.Bought[food.Item])
	}
}

func TestGachaIsExpensiveAndMostlyPlain(t *testing.T) {
	st := newUserState()
	if _, err := gacha(st, fixed(0.5)); err == nil {
		t.Fatal("a blind box is not free")
	}
	counts := [4]int{}
	for seed := 0; seed < 400; seed++ {
		st.Coin = gachaCost
		res, err := gacha(st, seeded(int64(seed)))
		if err != nil {
			t.Fatal(err)
		}
		if st.Coin != 0 || st.Inv[res.Item] == 0 || itemByID[res.Item].Rarity != res.Rarity {
			t.Fatalf("bad pull %+v coin %d", res, st.Coin)
		}
		counts[res.Rarity]++
	}
	if counts[0] < 160 || counts[3] > 40 || counts[3] == 0 {
		t.Fatalf("odds look wrong: %v", counts)
	}
}

func TestBagsCarryDifferentThings(t *testing.T) {
	// only what the bag added matters: a fresh state already holds starter items
	added := func(id string, rnd rng) map[string]int {
		st := newUserState()
		before := map[string]int{}
		for k, v := range st.Inv {
			before[k] = v
		}
		openBag(st, id, rnd)
		out := map[string]int{}
		for k, v := range st.Inv {
			if v > before[k] {
				out[k] = v - before[k]
			}
		}
		return out
	}
	for id := range added("tui_y", seeded(3)) {
		if itemByID[id].Kind != "med" {
			t.Fatalf("a medic bag holds medicine, got %s", id)
		}
	}
	gear := 0
	for id, n := range added("tui_vukhi", seeded(4)) {
		if gearByID[id].Slot == "" {
			t.Fatalf("a weapon crate holds gear, got %s", id)
		}
		gear += n
	}
	if gear != 1 {
		t.Fatalf("a weapon crate holds exactly one piece: %d", gear)
	}
	st := newUserState()
	note := openBag(st, "tui_vang", seeded(5))
	if st.Coin < 60 || !strings.Contains(note, "🪙") {
		t.Fatalf("a gold chest pays coins: %d (%s)", st.Coin, note)
	}
	kinds := map[string]bool{}
	for seed := 0; seed < 200; seed++ {
		kinds[rollBag(seeded(int64(seed)))] = true
	}
	if len(kinds) != len(bagOdds) {
		t.Fatalf("every bag kind should turn up: %v", kinds)
	}
}
