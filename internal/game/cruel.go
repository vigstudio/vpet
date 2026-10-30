package game

import (
	"errors"
	"fmt"
	"time"
)

// Killing your own pet is allowed, but it marks you: the "Kẻ tàn bạo" label sticks for
// fifteen days, during which no new pet can be hatched and the ones you keep suffer.
const (
	cruelDays    = 15
	cruelWindow  = cruelDays * 24 * time.Hour
	cruelHunger  = 1.6 // the surviving pets lose hunger and cheer faster while the mark lasts
	cruelHappy   = 1.8
	cruelHeal    = 0.5 // healing items only work half as well
	cruelCoinCut = 0.5 // and the world pays you half
)

// cruelUntil is when the mark lifts, or 0 when the user is not marked.
func (st *userState) cruelUntil() int64 { return st.Cruel }

func (st *userState) isCruel(now int64) bool { return st.Cruel > now }

// cruelLeft is the whole days still to serve, rounded up, for the label in the UI.
func (st *userState) cruelLeft(now int64) int {
	if !st.isCruel(now) {
		return 0
	}
	d := st.Cruel - now
	return int((d + 24*time.Hour.Milliseconds() - 1) / (24 * time.Hour.Milliseconds()))
}

// payout cuts earnings in half while the mark lasts.
func (st *userState) payout(n int, now int64) int {
	if !st.isCruel(now) {
		return n
	}
	return max(1, int(float64(n)*cruelCoinCut))
}

// kill puts a living pet down on purpose and stamps the owner with the mark.
func kill(st *userState, i int, now int64) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	p.Dead, p.Cause, p.DiedAt = true, "bị chủ giết hại", now
	p.Health, p.Hunger, p.Happy = 0, 0, 0
	st.Kills++
	st.Cruel = now + cruelWindow.Milliseconds()
	return fmt.Sprintf("💀 %s đã bị giết. Bạn mang nhãn **Kẻ tàn bạo** trong %d ngày: không ấp được thú mới, thú còn lại mau đói và buồn, thuốc men kém tác dụng, tiền thưởng giảm nửa.",
		p.Name, cruelDays), nil
}
