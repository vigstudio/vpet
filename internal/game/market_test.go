package game

import "testing"

func TestListingRefusesWhatCannotBeSold(t *testing.T) {
	st := newUserState()
	st.Coin = 500
	st.add("kiem", 2)
	st.add("da", 1)
	cases := []struct {
		name          string
		id            string
		qty, price    int
		wantRejection bool
	}{
		{"unknown item", "khongcogi", 1, 10, true},
		{"junk nobody buys", "da", 1, 10, true},
		{"more than owned", "kiem", 3, 10, true},
		{"zero quantity", "kiem", 0, 10, true},
		{"free", "kiem", 1, 0, true},
		{"above the price ceiling", "kiem", 1, maxAsk("kiem", 1) + 1, true},
		{"at the ceiling", "kiem", 1, maxAsk("kiem", 1), false},
		{"a fair lot", "kiem", 2, 200, false},
	}
	for _, c := range cases {
		err := checkListing(st, c.id, c.qty, c.price)
		if (err != nil) != c.wantRejection {
			t.Fatalf("%s: err=%v", c.name, err)
		}
	}
}

func TestListingNeedsThePostingFee(t *testing.T) {
	st := newUserState()
	st.add("kiem", 1)
	st.Coin = marketFee - 1
	if err := checkListing(st, "kiem", 1, 50); err == nil {
		t.Fatal("a seller who cannot pay the fee may not list")
	}
	st.Coin = marketFee
	if err := checkListing(st, "kiem", 1, 50); err != nil {
		t.Fatal(err)
	}
}

func TestListingCapPerSeller(t *testing.T) {
	st := newUserState()
	st.Coin = 1000
	st.add("kiem", 20)
	for i := 0; i < marketMax; i++ {
		st.Sell = append(st.Sell, "id")
	}
	if err := checkListing(st, "kiem", 1, 50); err == nil {
		t.Fatalf("a seller may have at most %d lots up", marketMax)
	}
}

// Every sale must burn gold. Without the tax the market only moves coins around and the
// daily earning cap stops holding the economy down.
func TestEverySaleBurnsSomeGold(t *testing.T) {
	for _, price := range []int{1, 5, 50, 999} {
		if taxOn(price) < 1 {
			t.Fatalf("price %d paid no tax", price)
		}
		if taxOn(price) >= price && price > 1 {
			t.Fatalf("price %d: the tax ate the whole sale", price)
		}
	}
}

// The ceiling is what stops a second account from handing someone unlimited gold.
func TestPriceCeilingTracksTheShopValue(t *testing.T) {
	if maxAsk("kiem", 1) <= 0 {
		t.Fatal("a sellable item needs a ceiling")
	}
	if maxAsk("kiem", 3) != maxAsk("kiem", 1)*3 {
		t.Fatal("the ceiling should scale with the lot size")
	}
	if maxAsk("gaythan", 1) <= maxAsk("dao", 1) {
		t.Fatal("a rarer item should be allowed to ask more")
	}
}

func TestCheapestPerUnitComesFirst(t *testing.T) {
	rows := []listingView{
		{listing: listing{ID: "a", Item: "kiem", Qty: 1, Price: 100, Created: 10}},
		{listing: listing{ID: "b", Item: "kiem", Qty: 4, Price: 200, Created: 20}},
		{listing: listing{ID: "c", Item: "kiem", Qty: 2, Price: 60, Created: 30}},
	}
	sortListings(rows)
	if rows[0].ID != "c" || rows[1].ID != "b" || rows[2].ID != "a" {
		t.Fatalf("cheapest per unit first: %s %s %s", rows[0].ID, rows[1].ID, rows[2].ID)
	}
}
