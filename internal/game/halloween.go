package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func halloweenOpen(now int64) bool {
	start := time.Date(2026, time.October, 31, 0, 0, 0, 0, vnZone).UnixMilli()
	end := time.Date(2026, time.November, 2, 0, 0, 0, 0, vnZone).UnixMilli()
	return now >= start && now < end
}

func copyPet(p *pet) (*pet, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var out pet
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Work on a copy: a missing socket or enhanced item must not consume any other item.
func resurrect(st *userState, born, died, sacrifice, now int64) (string, error) {
	if !halloweenOpen(now) {
		return "", errors.New("hồi sinh Halloween mở từ 31/10 đến hết 01/11/2026 theo giờ Việt Nam")
	}
	raw, err := json.Marshal(st)
	if err != nil {
		return "", err
	}
	var next userState
	if err := json.Unmarshal(raw, &next); err != nil {
		return "", err
	}
	next.normalize()
	gi, pi := -1, -1
	for i, g := range next.Graves {
		if g.Born == born && g.Died == died {
			gi = i
			break
		}
	}
	for i, p := range next.Pets {
		if p.Born == sacrifice {
			pi = i
			break
		}
	}
	if gi < 0 || pi < 0 || born == sacrifice {
		return "", errors.New("mộ hoặc thú hiến tế đã thay đổi; hãy tải lại")
	}
	donor := next.Pets[pi]
	if donor.Dead || donor.FrozenUntil > now || donor.CareUntil > now {
		return "", errors.New("thú hiến tế phải đang sống, không nghỉ phép hoặc dưỡng bệnh")
	}
	if len(next.DIn) > 0 || len(next.DOut) > 0 {
		return "", errors.New("hãy xử lý lời mời tỷ thí trước khi hiến tế")
	}
	g := next.Graves[gi]
	var revived *pet
	if g.Snapshot != nil {
		revived, err = copyPet(g.Snapshot)
		if err != nil {
			return "", err
		}
	} else {
		// ponytail: legacy graves contain appearance only; no invented skills, levels or items.
		revived = &pet{Name: g.Name, Parts: g.Parts, Born: g.Born, Lv: 1, Known: map[string]string{}}
	}
	recoverPetItems(&next, donor)
	for slot, id := range revived.Gear {
		if _, ok := gearByID[id]; !ok || !takeBagGear(&next, id, revived.GearLevel[slot]) {
			return "", errors.New("thiếu đúng trang bị/cấp cường hóa trong kho để hồi sinh; đồ đã trả về kho khi chôn")
		}
	}
	for _, gems := range revived.Sockets {
		for _, id := range gems {
			if _, _, ok := parseGem(id); !ok || next.Inv[id] < 1 {
				return "", errors.New("thiếu ngọc trong kho để khôi phục thú")
			}
			next.add(id, -1)
		}
	}
	if revived.Acc != "" {
		if itemByID[revived.Acc] == nil || next.Inv[revived.Acc] < 1 {
			return "", errors.New("thiếu phụ kiện trong kho để khôi phục thú")
		}
		next.add(revived.Acc, -1)
	}
	revived.Dead, revived.Sick = false, false
	revived.Cause, revived.Harm, revived.Effect = "", "", ""
	revived.DiedAt, revived.EffectUntil, revived.FrozenUntil, revived.CareUntil = 0, 0, 0, 0
	revived.Health, revived.Hunger, revived.Happy, revived.LowHours = 100, 70, 70, 0
	revived.LastTick, revived.EnergyAt = now, now
	revived.Touch = nil
	revived.WarnHungry, revived.WarnSick = false, false
	next.Pets[pi] = revived
	next.Graves = append(next.Graves[:gi], next.Graves[gi+1:]...)
	// The sacrifice is final, not a new recoverable grave that could create a revival cycle.
	*st = next
	return fmt.Sprintf("Đã hiến tế %s để hồi sinh %s. Đồ của thú hiến tế còn lại trong kho.", donor.Name, revived.Name), nil
}

func (p *Engine) resurrectPet(userID string, st *userState, born, died, sacrifice, now int64) (string, error) {
	if !halloweenOpen(now) {
		return "", errors.New("hồi sinh Halloween mở từ 31/10 đến hết 01/11/2026 theo giờ Việt Nam")
	}
	rooms, err := p.parties(now)
	if err != nil {
		return "", err
	}
	for _, room := range rooms {
		if room.Done {
			continue
		}
		for _, member := range room.Members {
			if member.User == userID {
				return "", errors.New("hãy rời tổ đội đang chờ trước khi hiến tế")
			}
		}
	}
	msg, err := resurrect(st, born, died, sacrifice, now)
	if err != nil {
		return "", err
	}
	if err := p.commitRows(map[string]any{userPrefix + userID: st}); err != nil {
		return "", err
	}
	return msg, nil
}
