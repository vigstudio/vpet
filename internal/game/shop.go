package game

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Money, the daily shop and the blind box. Coins come from successful activity (with a cap),
// from hunting and from winning fights; they only leave through the shop and the gacha.
const (
	coinCooldown  = 3 * time.Minute
	coinPerAction = 2
	coinBonus     = 3   // retained for imported economy compatibility
	coinsPerDay   = 70  // keeps the good stuff a few days away
	bossCoinsDay  = 150 // the world boss has its own daily ceiling, separate from activity
	replyFresh    = 15 * time.Minute
	gachaCost     = 150
	shopSize      = 8 // random non-food offers; the unlimited food counter is separate
	maxBuy        = 99
)

// gachaOdds and shopPrice are the only two knobs that decide how grindy the economy feels.
var gachaOdds = [4]int{74, 21, 4, 1}

type shopOffer struct {
	Item      string `json:"item"`
	Price     int    `json:"price"`
	Qty       int    `json:"qty"`
	Left      int    `json:"left"`
	Unlimited bool   `json:"unlimited,omitempty"`
}

type gachaResult struct {
	Item   string `json:"item"`
	Rarity int    `json:"rarity"`
	Coin   int    `json:"coin"`
	Note   string `json:"note"`
}

func dayOf(now int64) string { return time.UnixMilli(now).In(vnZone).Format("2006-01-02") }

// daySeed turns a date into a stable seed so everyone sees the same shelves all day.
func daySeed(day string) int64 {
	var n int64
	for _, c := range []byte(day) {
		n = n*131 + int64(c)
	}
	return n
}

// sellable lists what can appear in the shop or the blind box, cheapest tat excluded.
// The mythic rung is left out on purpose: the forge is the only way to reach it.
func sellable() []itemDef {
	out := []itemDef{}
	for _, it := range itemByID {
		if it.Kind == "junk" || it.Kind == "bag" || it.Kind == "gem" || it.Rarity >= mythicRarity {
			continue
		}
		out = append(out, *it)
	}
	// itemByID is a map, so sort by id to keep the daily shop deterministic
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].ID < out[j-1].ID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// priceOf is rarity first, then a nudge for the items that change the game most.
func priceOf(it itemDef) int {
	base := [6]int{28, 95, 260, 600, 1500, 3600}[min(max(it.Rarity, 0), divineRarity)]
	switch it.Kind {
	case "revive":
		base = 900
	case "card":
		base += 60
		if it.ID == "phieu_hoan_diem" {
			base = 1200
		}
		if it.ID == "tinh_the_tiem_nang" {
			base = 1800
		}
	case "stone":
		base += 80
	}
	return base
}

// foodStock is the permanent pantry. Food is basic pet care, not a scarcity reward, so every
// meal is always available at its catalog price and never counts against daily stock.
func foodStock() []shopOffer {
	out := []shopOffer{}
	for _, it := range catalog {
		if it.Kind == "food" {
			out = append(out, shopOffer{Item: it.ID, Price: priceOf(it), Unlimited: true})
		}
	}
	return out
}

// shopStock is the random non-food shelf. It is derived from the date alone: no storage,
// and it rotates every midnight while the food counter remains stable.
func shopStock(day string) []shopOffer {
	rnd := seeded(daySeed(day))
	pool := []itemDef{}
	for _, it := range sellable() {
		if it.Kind != "food" && it.Rarity > 0 {
			pool = append(pool, it)
		}
	}
	out := make([]shopOffer, 0, shopSize)
	seen := map[string]bool{}
	for len(out) < shopSize && len(seen) < len(pool) {
		it := pool[int(rnd()*float64(len(pool)))%len(pool)]
		if seen[it.ID] {
			continue
		}
		seen[it.ID] = true
		price := int(float64(priceOf(it)) * (0.85 + rnd()*0.4))
		qty := 1 + int(rnd()*float64(5-min(it.Rarity, 4)))
		out = append(out, shopOffer{Item: it.ID, Price: price, Qty: qty, Left: qty})
	}
	return out
}

// stockFor is the shop as this user sees it, with what they already bought today taken off.
func (st *userState) stockFor(now int64) []shopOffer {
	day := dayOf(now)
	rare := shopStock(day)
	if st.ShopDay == day {
		for i := range rare {
			rare[i].Left = max(0, rare[i].Qty-st.Bought[rare[i].Item])
		}
	}
	return append(foodStock(), rare...)
}

// earn pays for successful activity: capped per day and per few minutes.
// quickReply remains for imported compatibility tests; standalone activity never uses it.
func (st *userState) earn(now int64, quickReply bool) int {
	day := dayOf(now)
	if st.CoinDay != day {
		st.CoinDay, st.CoinToday = day, 0
	}
	if st.CoinToday >= coinsPerDay || now-st.LastCoin < coinCooldown.Milliseconds() {
		return 0
	}
	n := coinPerAction
	if quickReply {
		n += coinBonus
	}
	n = min(st.payout(n, now), coinsPerDay-st.CoinToday)
	st.LastCoin = now
	st.CoinToday += n
	st.Coin += n
	return n
}

// earnBoss pays world-boss gold against its own daily ceiling. It is deliberately separate
// from earn(): boss gold used to be added straight to the balance, which walked around the
// economy entirely — a long day of hitting paid tens of thousands of coins against a 70-coin
// cap. Keeping a second counter means the boss stays worth fighting without letting a player
// skip the rest of the game, and without eating the coins earned by looking after a pet.
func (st *userState) earnBoss(amount int, now int64) int {
	if amount <= 0 {
		return 0
	}
	day := dayOf(now)
	if st.BossCoinDay != day {
		st.BossCoinDay, st.BossCoinToday = day, 0
	}
	paid := min(amount, max(0, bossCoinsDay-st.BossCoinToday))
	st.BossCoinToday += paid
	st.Coin += paid
	return paid
}

func buy(st *userState, id string, n int, now int64) (string, error) {
	if n < 1 || n > maxBuy {
		return "", errors.New("số lượng không hợp lệ")
	}
	day := dayOf(now)
	var offer *shopOffer
	for _, o := range st.stockFor(now) {
		if o.Item == id {
			cp := o
			offer = &cp
		}
	}
	if offer == nil {
		return "", errors.New("hôm nay cửa hàng không bán món này")
	}
	if !offer.Unlimited && offer.Left < n {
		return "", fmt.Errorf("chỉ còn %d món", offer.Left)
	}
	cost := offer.Price * n
	if st.Coin < cost {
		return "", fmt.Errorf("thiếu %d vàng", cost-st.Coin)
	}
	if st.ShopDay != day {
		st.ShopDay, st.Bought = day, map[string]int{}
	}
	if st.Bought == nil {
		st.Bought = map[string]int{}
	}
	st.Coin -= cost
	if !offer.Unlimited {
		st.Bought[id] += n
	}
	st.add(id, n)
	def := itemByID[id]
	return fmt.Sprintf("Mua %d× %s %s hết %d vàng", n, def.Emoji, def.Name, cost), nil
}

// gacha opens a blind box: the good tiers stay rare on purpose.
func gacha(st *userState, rnd rng) (*gachaResult, error) {
	if st.Coin < gachaCost {
		return nil, fmt.Errorf("hộp mù giá %d vàng, bạn còn %d", gachaCost, st.Coin)
	}
	st.Coin -= gachaCost
	total := 0
	for _, w := range gachaOdds {
		total += w
	}
	r, roll := 0, rnd()*float64(total)
	for i, w := range gachaOdds {
		roll -= float64(w)
		if roll < 0 {
			r = i
			break
		}
	}
	pool := []itemDef{}
	for _, it := range sellable() {
		if it.Rarity == r {
			pool = append(pool, it)
		}
	}
	if len(pool) == 0 {
		st.Coin += gachaCost
		return nil, errors.New("hộp mù đang trống, thử lại sau")
	}
	it := pool[int(rnd()*float64(len(pool)))%len(pool)]
	st.add(it.ID, 1)
	res := &gachaResult{Item: it.ID, Rarity: r, Coin: st.Coin}
	res.Note = fmt.Sprintf("Mở hộp mù ra %s %s (%s)!", it.Emoji, it.Name, rarityNames[r])
	if r >= 2 {
		res.Note = "🎉 " + res.Note
	}
	return res, nil
}

// ---- supply bags ----

// bagOdds: plain food most of the time, a gold chest only now and then.
var bagOdds = []struct {
	ID string
	W  int
}{{"tui", 70}, {"tui_y", 14}, {"tui_vukhi", 11}, {"tui_vang", 5}}

func rollBag(rnd rng) string {
	total := 0
	for _, b := range bagOdds {
		total += b.W
	}
	n := rnd() * float64(total)
	for _, b := range bagOdds {
		n -= float64(b.W)
		if n < 0 {
			return b.ID
		}
	}
	return "tui"
}

// openBag unpacks one bag; each kind draws from its own shelf.
func openBag(st *userState, id string, rnd rng) string {
	pick := func(pool []string) string { return pool[int(rnd()*float64(len(pool)))%len(pool)] }
	var got []string
	switch id {
	case "tui_y":
		got = []string{pick([]string{"thuoc", "vitamin", "thaomoc"}), pick([]string{"thuoc", "vitamin", "thaomoc"})}
	case "tui_vukhi":
		gear := []string{}
		for gid := range gearByID {
			if itemByID[gid].Rarity >= mythicRarity {
				continue // the mythic rung is forged, never found in a crate
			}
			gear = append(gear, gid)
		}
		for i := 1; i < len(gear); i++ { // map order is random, sort so a seed replays
			for j := i; j > 0 && gear[j] < gear[j-1]; j-- {
				gear[j], gear[j-1] = gear[j-1], gear[j]
			}
		}
		got = []string{pick(gear)}
	case "tui_vang":
		coin := 60 + int(rnd()*90)
		st.Coin += coin
		got = []string{pick([]string{"ngoc", "sach", "the_skin", "the_nghi", "nhanbac", "giapthep"})}
		for _, g := range got {
			st.add(g, 1)
		}
		return fmt.Sprintf("Rương vàng: %d 🪙 và %s %s!", coin, itemByID[got[0]].Emoji, itemByID[got[0]].Name)
	default:
		got = []string{rollItem(rnd)}
		if rnd() < 0.3 {
			got = append(got, rollItem(rnd))
		}
	}
	names := make([]string, len(got))
	for i, g := range got {
		st.add(g, 1)
		names[i] = itemByID[g].Emoji + " " + itemByID[g].Name
	}
	return "Mở túi được: " + strings.Join(names, ", ")
}
