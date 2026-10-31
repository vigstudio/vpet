package game

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// The player market. Anyone can put an item up for gold; anyone else can buy it on the spot.
// This replaces the old 1:1 barter offer, which needed both people online and awake.
//
// Two rules keep it from wrecking the economy, which grants at most 70 activity gold a day:
//   - every sale pays a tax, so gold leaves the world on each trade instead of only moving;
//   - a listing price is capped against the item's shop value, so very high-price transfers
//     are limited; this does not prevent alternate-account transfers.
const (
	marketTTL     = 7 * 24 * time.Hour // unsold listings go home after a week
	marketMax     = 8                  // listings one user can have up at once
	marketTax     = 10                 // percent of the sale price burned
	marketFee     = 5                  // gold to post a listing, burned, so spam costs something
	marketCeiling = 5                  // a listing may ask at most this many times the shop price

)

type listing struct {
	ID      string `json:"id"`
	Seller  string `json:"seller"`
	Item    string `json:"item"`
	Qty     int    `json:"qty"`
	Price   int    `json:"price"` // gold for the whole lot
	Created int64  `json:"created"`
}

// listingView is a listing as the client sees it, with the seller's name resolved.
type listingView struct {
	listing
	SellerName string `json:"seller_name"`
	Mine       bool   `json:"mine"`
}

// buyOrder is a public request for goods. Price is escrowed at posting, so any player with
// enough stock can fulfill it in one click without waiting for the buyer to return.
type buyOrder struct {
	ID      string `json:"id"`
	Buyer   string `json:"buyer"`
	Item    string `json:"item"`
	Qty     int    `json:"qty"`
	Price   int    `json:"price"` // total prepaid gold for the lot
	Created int64  `json:"created"`
}

type buyOrderView struct {
	buyOrder
	BuyerName string `json:"buyer_name"`
	Mine      bool   `json:"mine"`
}

// marketValue is what the shop would charge for one of these, and the base for the price cap.
func marketValue(id string) int {
	def := itemByID[id]
	if def == nil {
		return 0
	}
	return priceOf(*def)
}

// maxAsk is the highest total price allowed for a lot, to bound how much gold can move through one sale; it is not an anti-alt guarantee.
func maxAsk(id string, qty int) int { return marketValue(id) * qty * marketCeiling }

// checkListing validates a would-be listing against the seller's inventory and the price cap.
func checkListing(st *userState, id string, qty, price int) error {
	def := itemByID[id]
	switch {
	case def == nil:
		return errors.New("không có món này")
	case id == "tinhhon":
		return errors.New("Tinh Hồn không thể giao dịch")
	case def.Kind == "junk":
		return errors.New("món này không ai mua")
	case qty < 1 || qty > 99:
		return errors.New("số lượng phải từ 1 đến 99")
	case st.Inv[id] < qty:
		return fmt.Errorf("bạn chỉ có %d× %s", st.Inv[id], def.Name)
	case price < 1:
		return errors.New("giá phải ít nhất 1 vàng")
	case price > maxAsk(id, qty):
		return fmt.Errorf("giá tối đa cho %d× %s là %d vàng", qty, def.Name, maxAsk(id, qty))
	case st.Coin < marketFee:
		return fmt.Errorf("cần %d vàng phí đăng bán, bạn còn %d", marketFee, st.Coin)
	case len(st.Sell) >= marketMax:
		return fmt.Errorf("bạn đang bán %d món, gỡ bớt rồi đăng tiếp", marketMax)
	}
	return nil
}

// taxOn is the cut the market burns from a sale, at least 1 gold so every trade costs something.
func taxOn(price int) int { return max(1, price*marketTax/100) }

func checkBuyOrder(st *userState, id string, qty, price int) error {
	def := itemByID[id]
	switch {
	case def == nil:
		return errors.New("không có món này")
	case id == "tinhhon":
		return errors.New("Tinh Hồn không thể giao dịch")
	case def.Kind == "junk":
		return errors.New("không thể đăng mua đồ bỏ")
	case qty < 1 || qty > 99:
		return errors.New("số lượng phải từ 1 đến 99")
	case price < 1:
		return errors.New("giá phải ít nhất 1 vàng")
	case price > maxAsk(id, qty):
		return fmt.Errorf("giá tối đa cho %d× %s là %d vàng", qty, def.Name, maxAsk(id, qty))
	case st.Coin < price+marketFee:
		return fmt.Errorf("cần %d vàng gồm giá mua và phí đăng, bạn còn %d", price+marketFee, st.Coin)
	case len(st.Buy) >= marketMax:
		return fmt.Errorf("bạn đang có %d tin mua, hủy bớt rồi đăng tiếp", marketMax)
	}
	return nil
}

// sortListings puts the cheapest gold-per-unit first, then the newest, so the front page is
// where the bargains are rather than where the oldest junk is.
func sortListings(rows []listingView) {
	sort.Slice(rows, func(a, b int) bool {
		pa := float64(rows[a].Price) / float64(max(rows[a].Qty, 1))
		pb := float64(rows[b].Price) / float64(max(rows[b].Qty, 1))
		if pa != pb {
			return pa < pb
		}
		return rows[a].Created > rows[b].Created
	})
}

// Highest unit bid first gives sellers the most valuable request at the top.
func sortBuyOrders(rows []buyOrderView) {
	sort.Slice(rows, func(a, b int) bool {
		pa := float64(rows[a].Price) / float64(max(rows[a].Qty, 1))
		pb := float64(rows[b].Price) / float64(max(rows[b].Qty, 1))
		if pa != pb {
			return pa > pb
		}
		return rows[a].Created > rows[b].Created
	})
}
