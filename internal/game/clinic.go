package game

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

// The clinic. Restoratives already existed but were scattered: they dropped from bags, showed
// up in the rotating shop some days and not others, and a sick pet had no reliable place to
// go. Production data showed the cost of that — the median player holds 4 gold and cannot
// plan around a shop that changes daily. The clinic always stocks the basics at a fixed
// price and can treat a pet directly, so care is never gated behind luck.
const (
	cureCost     = 60   // treating illness outright, more than a potion because it never fails
	restCost     = 40   // a full heal without spending an item
	reviveCost   = 1200 // the clinic alternative to a phoenix feather
	clinicMarkup = 115  // percent of the base price, so bag drops stay the cheaper route
)

// clinicStock is the fixed shelf. Unlike the daily shop this never rotates: a player who
// needs a cure can always buy one.
var clinicStock = []string{"thuoc", "vitamin", "banggac", "thuocbo", "chaogiai", "tinhdau", "canhtien", "nuocthanh", "sammy"}

// Longer stays cost less per day, but every package is paid in full before care starts.
var carePackages = []carePackage{
	{Days: 1, Price: 90},
	{Days: 3, Price: 240},
	{Days: 7, Price: 490},
	{Days: 14, Price: 840},
}

type carePackage struct {
	Days  int `json:"days"`
	Price int `json:"price"`
}

type clinicOffer struct {
	Item  string `json:"item"`
	Price int    `json:"price"`
}

// clinicPrice is the catalog price plus a markup, so buying here is convenient rather than
// optimal and the market keeps its reason to exist.
func clinicPrice(id string) int {
	def := itemByID[id]
	if def == nil {
		return 0
	}
	return max(1, priceOf(*def)*clinicMarkup/100)
}

func clinicShelf() []clinicOffer {
	out := make([]clinicOffer, 0, len(clinicStock))
	for _, id := range clinicStock {
		if itemByID[id] == nil {
			continue
		}
		out = append(out, clinicOffer{Item: id, Price: clinicPrice(id)})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Price < out[b].Price })
	return out
}

// buyCare sells one restorative. There is no daily cap: the limit is gold, and gold is the
// scarce thing.
func buyCare(st *userState, id string, qty int, now int64) (string, error) {
	stocked := false
	for _, s := range clinicStock {
		if s == id {
			stocked = true
		}
	}
	if !stocked {
		return "", errors.New("bệnh viện không bán món này")
	}
	if qty < 1 || qty > 20 {
		return "", errors.New("mỗi lần mua từ 1 đến 20 món")
	}
	cost := clinicPrice(id) * qty
	if st.Coin < cost {
		return "", fmt.Errorf("cần %d vàng, bạn còn %d", cost, st.Coin)
	}
	st.Coin -= cost
	st.add(id, qty)
	return fmt.Sprintf("💊 Mua %d× %s hết %d vàng.", qty, itemByID[id].Name, cost), nil
}

// boardCare admits one living pet for a prepaid package. Admission stabilizes the pet so a
// sick or hungry animal is safe immediately; decay resumes from the exact checkout time.
// Extending an active stay starts after its current deadline, never discarding paid time.
func boardCare(st *userState, i, days int, now int64) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết, cần hồi sinh trước")
	}
	price := 0
	for _, pkg := range carePackages {
		if pkg.Days == days {
			price = pkg.Price
			break
		}
	}
	if price == 0 {
		return "", errors.New("chọn gói chăm sóc 1, 3, 7 hoặc 14 ngày")
	}
	if st.Coin < price {
		return "", fmt.Errorf("gói %d ngày giá %d vàng, bạn còn %d", days, price, st.Coin)
	}
	st.Coin -= price
	p.Sick, p.LowHours = false, 0
	p.Health = math.Max(p.Health, 70)
	p.Hunger = math.Max(p.Hunger, 80)
	p.Happy = math.Max(p.Happy, 80)
	p.WarnHungry, p.WarnSick = false, false
	base := max(now, p.CareUntil)
	p.CareUntil = base + int64(days)*24*time.Hour.Milliseconds()
	p.LastTick = now
	return fmt.Sprintf("🏥 Đã gửi %s chăm sóc %d ngày tới %s (%d vàng).", p.Name, days,
		time.UnixMilli(p.CareUntil).In(vnZone).Format("15:04 02/01/2006"), price), nil
}

// treat is the service counter: pay gold instead of carrying the right item. Each service
// refuses when it would do nothing, so no one pays for a cure they do not need.
func treat(st *userState, i int, service string, now int64) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	switch service {
	case "cure":
		if p.Dead {
			return "", errors.New(p.Name + " đã chết, cần hồi sinh trước")
		}
		if !p.Sick {
			return "", errors.New(p.Name + " không bị bệnh")
		}
		if st.Coin < cureCost {
			return "", fmt.Errorf("chữa bệnh giá %d vàng, bạn còn %d", cureCost, st.Coin)
		}
		st.Coin -= cureCost
		p.Sick, p.LowHours = false, 0
		p.Health = clamp(p.Health + healFor(p, 20))
		return fmt.Sprintf("🏥 Bác sĩ chữa khỏi bệnh cho %s (%d vàng).", p.Name, cureCost), nil
	case "rest":
		if p.Dead {
			return "", errors.New(p.Name + " đã chết, cần hồi sinh trước")
		}
		if p.Health >= 100 {
			return "", errors.New(p.Name + " đang khỏe mạnh")
		}
		if st.Coin < restCost {
			return "", fmt.Errorf("nằm viện giá %d vàng, bạn còn %d", restCost, st.Coin)
		}
		st.Coin -= restCost
		before := p.Health
		p.Health = clamp(p.Health + healFor(p, 100))
		return fmt.Sprintf("🏥 %s nằm viện, hồi %.0f sức khỏe (%d vàng).", p.Name, p.Health-before, restCost), nil
	case "revive":
		if !p.Dead {
			return "", errors.New(p.Name + " vẫn còn sống")
		}
		if st.Coin < reviveCost {
			return "", fmt.Errorf("hồi sinh giá %d vàng, bạn còn %d", reviveCost, st.Coin)
		}
		st.Coin -= reviveCost
		p.Dead, p.Cause, p.DiedAt, p.Sick, p.LowHours = false, "", 0, false, 0
		p.Health, p.Hunger, p.Happy, p.LastTick = 50, 50, 60, now
		p.setEffect("ngon", 10, now)
		return fmt.Sprintf("🏥 Bệnh viện cứu sống %s (%d vàng).", p.Name, reviveCost), nil
	}
	return "", errors.New("dịch vụ không hợp lệ")
}

// clinicView is the price list the panel draws.
type clinicView struct {
	Shelf  []clinicOffer `json:"shelf"`
	Cure   int           `json:"cure"`
	Rest   int           `json:"rest"`
	Revive int           `json:"revive"`
	Care   []carePackage `json:"care"`
}

func clinicInfo() clinicView {
	return clinicView{Shelf: clinicShelf(), Cure: cureCost, Rest: restCost, Revive: reviveCost, Care: carePackages}
}
