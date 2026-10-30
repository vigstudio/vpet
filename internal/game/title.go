package game

import (
	"errors"
	"fmt"
	"math"
)

// round1 keeps a stat to one decimal. Without it a tier step like 0.6×3 reaches the client as
// 1.7999999999999998 and is printed in full.
func round1(v float64) float64 { return math.Round(v*10) / 10 }

// Titles. Medals (hunt.go) already read a pet's record and show it off, but they pay nothing,
// so a long record stops meaning anything once the badges are collected. A title is the same
// record read again, except the pet wears one of them and it carries real combat stats.
//
// Seven tracks, five rungs each. A pet unlocks as many as its record earns but wears exactly
// one, so picking a title is a build decision rather than a pile of free numbers. Each track
// pays a different shape - the duellist hits harder, the explorer lives longer - which is what
// makes the choice worth making.
//
// Every track reads a counter the pet already keeps, so there is nothing new to persist and no
// migration: a pet from before this version simply has no Title and gets nothing.
const (
	titleTiers = 5
	// A rank-5 title is worth about one tier-3 item. Only one is worn, so these are the
	// absolute ceilings for the whole system.
	titleMaxAtk = 8.0
	titleMaxDef = 6.0
	titleMaxHP  = 40.0
)

type titleDef struct {
	ID    string
	Name  string
	Desc  string
	Emoji string
	Track string // lv | hunt | wild | duel | star | gold | arena
	Tier  int    // 1..titleTiers
	Need  int
	Atk   float64
	Def   float64
	HP    float64
}

// titleTracks lists the thresholds and the stat shape of each ladder. The five rungs of a
// track share a shape and scale with the tier, so a track is described once instead of five
// times and the ladder cannot drift apart by hand.
var titleTracks = []struct {
	Track string
	Emoji string
	Names [titleTiers]string
	Desc  string
	Needs [titleTiers]int
	// per-tier step; tier 5 lands on the ceilings above
	Atk float64
	Def float64
	HP  float64
}{
	{
		Track: "lv", Emoji: "🌟", Desc: "Đạt cấp %d",
		Names: [titleTiers]string{"Chồi Non", "Lữ Hành Giả", "Kiếm Khách", "Tông Sư", "Thiên Mệnh Giả"},
		Needs: [titleTiers]int{10, 40, 100, 220, 400},
		Atk:   0.6, Def: 0.6, HP: 6,
	},
	{
		Track: "hunt", Emoji: "🗺️", Desc: "Đi săn %d chuyến",
		Names: [titleTiers]string{"Chân Chạy", "Người Dẫn Đường", "Nhà Thám Hiểm", "Kẻ Vẽ Bản Đồ", "Bá Chủ Đường Xa"},
		Needs: [titleTiers]int{25, 75, 200, 450, 900},
		Atk:   0.2, Def: 0.4, HP: 8,
	},
	{
		// Campaign wins also raise p.Wild (campaign.go), so these thresholds sit high enough
		// that the track cannot quietly unlock itself off the back of the star track.
		Track: "wild", Emoji: "🐾", Desc: "Hạ %d thú hoang",
		Names: [titleTiers]string{"Tay Săn Mới", "Khắc Tinh Dã Thú", "Thợ Săn Cừ Khôi", "Hung Thần Hoang Dã", "Chúa Tể Muông Thú"},
		Needs: [titleTiers]int{60, 180, 420, 850, 1600},
		Atk:   0.4, Def: 1.2, HP: 3,
	},
	{
		Track: "duel", Emoji: "⚔️", Desc: "Thắng %d trận tỷ thí",
		Names: [titleTiers]string{"Tân Binh Đấu Trường", "Võ Sĩ", "Kiếm Sĩ Lão Luyện", "Quán Quân", "Độc Cô Cầu Bại"},
		Needs: [titleTiers]int{10, 35, 90, 200, 400},
		Atk:   1.6, Def: 0.2, HP: 2,
	},
	{
		Track: "star", Emoji: "⭐", Desc: "Đạt %d sao chiến dịch",
		Names: [titleTiers]string{"Người Mở Lối", "Kẻ Chinh Phạt", "Anh Hùng Chiến Dịch", "Huyền Thoại Sống", "Vĩ Nhân Thiên Đảo"},
		Needs: [titleTiers]int{20, 50, 90, 130, 162},
		Atk:   0.6, Def: 1.0, HP: 4,
	},
	{
		Track: "gold", Emoji: "👑", Desc: "Nhặt được %d món hoàng kim",
		Names: [titleTiers]string{"Mắt Tinh Đời", "Thợ Đãi Vàng", "Tay Săn Hoàng Kim", "Kẻ Hái Lộc Trời", "Thần Tài Hiện Thế"},
		Needs: [titleTiers]int{3, 12, 30, 70, 150},
		Atk:   1.4, Def: 0.4, HP: 2,
	},
	{
		Track: "arena", Emoji: "🏆", Desc: "Đạt %d điểm Đấu trường",
		Names: [titleTiers]string{"Hạng Đồng", "Hạng Bạc", "Hạng Vàng", "Hạng Kim Cương", "Bất Khả Chiến Bại"},
		Needs: [titleTiers]int{1100, 1300, 1600, 2000, 2500},
		Atk:   0.8, Def: 0.8, HP: 5,
	},
}

// titles is the flat ladder, built once from the track table.
var titles = func() []titleDef {
	out := make([]titleDef, 0, len(titleTracks)*titleTiers)
	for _, tr := range titleTracks {
		for i := 0; i < titleTiers; i++ {
			tier := i + 1
			out = append(out, titleDef{
				ID:    fmt.Sprintf("t_%s%d", tr.Track, tier),
				Name:  tr.Names[i],
				Desc:  fmt.Sprintf(tr.Desc, tr.Needs[i]),
				Emoji: tr.Emoji,
				Track: tr.Track,
				Tier:  tier,
				Need:  tr.Needs[i],
				// Rounded to one decimal: 0.6×3 in binary floating point is
				// 1.7999999999999998, and the panel would print every digit of it.
				Atk: round1(tr.Atk * float64(tier)),
				Def: round1(tr.Def * float64(tier)),
				HP:  round1(tr.HP * float64(tier)),
			})
		}
	}
	return out
}()

var titleByID = func() map[string]*titleDef {
	m := make(map[string]*titleDef, len(titles))
	for i := range titles {
		m[titles[i].ID] = &titles[i]
	}
	return m
}()

// titleProgress is what the pet has on a track right now. Every value comes from a field the
// pet already keeps, which is why titles need no new counter and no migration.
func titleProgress(p *pet, track string) int {
	switch track {
	case "lv":
		return max(p.Lv, 1)
	case "hunt":
		return p.Hunts
	case "wild":
		return p.Wild
	case "duel":
		return p.Wins
	case "star":
		stars := 0
		for _, row := range p.Campaign {
			for _, s := range row {
				stars += s
			}
		}
		return stars
	case "gold":
		return p.Gold
	case "arena":
		return arenaPoints(p)
	}
	return 0
}

func titleUnlocked(p *pet, t *titleDef) bool {
	return t != nil && titleProgress(p, t.Track) >= t.Need
}

// titleOf is the one place combat reads a title from. It re-checks the unlock condition on
// every read, so a title that was equipped and later became invalid - or one written straight
// into stored data - pays nothing.
func titleOf(p *pet) gearDef {
	if p == nil || p.Title == "" {
		return gearDef{}
	}
	t := titleByID[p.Title]
	if !titleUnlocked(p, t) {
		return gearDef{}
	}
	return gearDef{Atk: t.Atk, Def: t.Def, HP: t.HP}
}

type titleView struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Desc     string  `json:"desc"`
	Emoji    string  `json:"emoji"`
	Track    string  `json:"track"`
	Tier     int     `json:"tier"`
	Need     int     `json:"need"`
	Have     int     `json:"have"`
	Unlocked bool    `json:"unlocked"`
	Equipped bool    `json:"equipped"`
	Atk      float64 `json:"atk,omitempty"`
	Def      float64 `json:"def,omitempty"`
	HP       float64 `json:"hp,omitempty"`
}

func (p *pet) titleViews() []titleView {
	out := make([]titleView, 0, len(titles))
	for i := range titles {
		t := &titles[i]
		have := titleProgress(p, t.Track)
		out = append(out, titleView{ID: t.ID, Name: t.Name, Desc: t.Desc, Emoji: t.Emoji,
			Track: t.Track, Tier: t.Tier, Need: t.Need, Have: min(have, t.Need),
			Unlocked: have >= t.Need, Equipped: p.Title == t.ID,
			Atk: t.Atk, Def: t.Def, HP: t.HP})
	}
	return out
}

// wearTitle equips one title, or takes the current one off when id is empty.
func wearTitle(st *userState, i int, id string) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if id == "" {
		p.Title = ""
		return "Đã gỡ danh hiệu.", nil
	}
	t := titleByID[id]
	if t == nil {
		return "", errors.New("danh hiệu không tồn tại")
	}
	if !titleUnlocked(p, t) {
		return "", fmt.Errorf("chưa đạt điều kiện: %d/%d", titleProgress(p, t.Track), t.Need)
	}
	p.Title = id
	return fmt.Sprintf("%s %s đã mang danh hiệu %s.", t.Emoji, p.Name, t.Name), nil
}
