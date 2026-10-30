package game

import (
	"errors"
	"fmt"
	"time"
)

// Daily attendance. The calendar is derived from the month itself, never stored, so every
// player sees the same prizes on the same dates and a new month needs no migration. Only
// what the player has actually claimed is durable.
//
// The streak counts consecutive claimed days. Missing a day resets it to 1 on the next
// claim, so the milestone rewards stay a reason to come back rather than a login tax.
const (
	checkinCoin     = 12 // plain day: a little gold on top of the item
	streakBonusCoin = 60 // paid at every milestone, on top of its item
)

// checkinMilestones are the streak lengths that pay extra, longest first when several match.
var checkinMilestones = []struct {
	Days int
	Item string
}{
	{30, "tinh_the_tiem_nang"},
	{20, "the_nang_luong"},
	{10, "ngoc"},
}

// dayPrize is the item for one date of the month. The table repeats every ten days so the
// first, tenth, twentieth and thirtieth are always the memorable ones.
var dayPrizes = []string{
	"tui", "com", "thuoc", "tui", "nuoc_tang_luc",
	"banhmi", "tui_y", "vitamin", "tui", "tui_vukhi",
}

func prizeOfDay(day int) string {
	return dayPrizes[(max(day, 1)-1)%len(dayPrizes)]
}

// checkinDay is one square of the calendar.
type checkinDay struct {
	Day     int    `json:"day"`
	Item    string `json:"item"`
	Coin    int    `json:"coin"`
	Claimed bool   `json:"claimed"`
	Today   bool   `json:"today"`
	Past    bool   `json:"past"`
}

type checkinMilestone struct {
	Days    int    `json:"days"`
	Item    string `json:"item"`
	Coin    int    `json:"coin"`
	Reached bool   `json:"reached"`
}

type checkinView struct {
	Month      string             `json:"month"`
	Days       []checkinDay       `json:"days"`
	Streak     int                `json:"streak"`
	Best       int                `json:"best"`
	Claimed    bool               `json:"claimed_today"`
	Milestones []checkinMilestone `json:"milestones"`
}

func monthOf(now int64) string { return time.UnixMilli(now).In(vnZone).Format("2006-01") }

// daysInMonth answers how many squares the calendar needs, leap years included.
func daysInMonth(now int64) int {
	t := time.UnixMilli(now).In(vnZone)
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, vnZone).AddDate(0, 1, -1).Day()
}

// streakAfter is what the streak becomes when today is claimed: consecutive days only.
func (st *userState) streakAfter(now int64) int {
	if st.CheckDay == "" {
		return 1
	}
	last, err := time.ParseInLocation("2006-01-02", st.CheckDay, vnZone)
	if err != nil {
		return 1
	}
	today := time.UnixMilli(now).In(vnZone)
	yesterday := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, vnZone).AddDate(0, 0, -1)
	if last.Equal(yesterday) {
		return st.Streak + 1
	}
	return 1
}

func (st *userState) checkinView(now int64) checkinView {
	today := time.UnixMilli(now).In(vnZone)
	claimedToday := st.CheckDay == dayOf(now)
	v := checkinView{Month: monthOf(now), Streak: st.Streak, Best: st.BestStreak, Claimed: claimedToday}
	if !claimedToday {
		// show what today's claim would be worth, not a stale streak from last week
		v.Streak = st.streakAfter(now) - 1
	}
	for day := 1; day <= daysInMonth(now); day++ {
		v.Days = append(v.Days, checkinDay{
			Day: day, Item: prizeOfDay(day), Coin: checkinCoin,
			Today:   day == today.Day(),
			Past:    day < today.Day(),
			Claimed: day == today.Day() && claimedToday,
		})
	}
	for _, m := range checkinMilestones {
		v.Milestones = append(v.Milestones, checkinMilestone{Days: m.Days, Item: m.Item, Coin: streakBonusCoin,
			Reached: st.Streak >= m.Days})
	}
	// longest milestone first reads badly on a progress row; show the nearest goal first
	for i, j := 0, len(v.Milestones)-1; i < j; i, j = i+1, j-1 {
		v.Milestones[i], v.Milestones[j] = v.Milestones[j], v.Milestones[i]
	}
	return v
}

// claimCheckin pays today's square once. Milestones pay on the exact day the streak reaches
// them, so a 30-day run collects the 10th, 20th and 30th rewards along the way.
func claimCheckin(st *userState, now int64) (string, error) {
	if st.CheckDay == dayOf(now) {
		return "", errors.New("hôm nay đã điểm danh rồi, mai quay lại nhé")
	}
	streak := st.streakAfter(now)
	st.CheckDay = dayOf(now)
	st.Streak = streak
	st.BestStreak = max(st.BestStreak, streak)

	day := time.UnixMilli(now).In(vnZone).Day()
	item := prizeOfDay(day)
	st.add(item, 1)
	st.Coin += checkinCoin
	msg := fmt.Sprintf("📅 Điểm danh ngày %d: nhận %s %s và %d vàng. Chuỗi %d ngày.",
		day, itemByID[item].Emoji, itemByID[item].Name, checkinCoin, streak)
	for _, m := range checkinMilestones {
		if streak == m.Days {
			st.add(m.Item, 1)
			st.Coin += streakBonusCoin
			msg += fmt.Sprintf(" 🎉 Mốc %d ngày: thêm %s %s và %d vàng!",
				m.Days, itemByID[m.Item].Emoji, itemByID[m.Item].Name, streakBonusCoin)
		}
	}
	return msg, nil
}
