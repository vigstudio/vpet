package game

import (
	"strings"
	"testing"
	"time"
)

func TestKillingMarksTheOwner(t *testing.T) {
	st := huntPet(t)
	now := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC).UnixMilli()
	if st.isCruel(now) || st.cruelLeft(now) != 0 {
		t.Fatal("a fresh owner is not marked")
	}
	msg, err := kill(st, 0, now)
	if err != nil {
		t.Fatal(err)
	}
	p := st.Pets[0]
	if !p.Dead || p.Cause != "bị chủ giết hại" || p.DiedAt != now || st.Kills != 1 {
		t.Fatalf("the pet should be dead by the owner's hand: %+v", p)
	}
	if !strings.Contains(msg, "Kẻ tàn bạo") {
		t.Fatalf("the message names the mark: %q", msg)
	}
	if !st.isCruel(now) || st.cruelLeft(now) != cruelDays {
		t.Fatalf("the mark lasts %d days, got %d", cruelDays, st.cruelLeft(now))
	}
	if _, err := kill(st, 0, now); err == nil {
		t.Fatal("a dead pet cannot be killed twice")
	}

	// no new pets until it lifts
	if err := createPet(st, "Thú Mới", map[string]int{"head": 1, "body": 1}, now+time.Hour.Milliseconds(), seeded(1)); err == nil {
		t.Fatal("a marked owner cannot hatch")
	}
	after := now + cruelWindow.Milliseconds() + 1000
	if st.isCruel(after) || st.cruelLeft(after) != 0 {
		t.Fatal("the mark lifts after the window")
	}
	if err := createPet(st, "Thú Mới", map[string]int{"head": 1, "body": 1}, after, seeded(1)); err != nil {
		t.Fatalf("hatching works again once it lifts: %v", err)
	}
}

func TestTheMarkBitesInEveryWay(t *testing.T) {
	now := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC).UnixMilli()

	// hunger and cheer fall faster
	plain, cruel := huntPet(t), huntPet(t)
	cruel.Cruel = now + cruelWindow.Milliseconds()
	for _, st := range []*userState{plain, cruel} {
		st.Pets[0].LastTick = now
		st.markPets(now)
	}
	later := now + 6*time.Hour.Milliseconds()
	plain.tickAll(later)
	cruel.tickAll(later)
	if cruel.Pets[0].Hunger >= plain.Pets[0].Hunger || cruel.Pets[0].Happy >= plain.Pets[0].Happy {
		t.Fatalf("a marked owner's pet suffers more: %+v vs %+v", cruel.Pets[0], plain.Pets[0])
	}

	// healing is halved
	p := cruel.Pets[0]
	p.Health = 20
	full := healFor(&pet{}, 40)
	half := healFor(p, 40)
	if half >= full {
		t.Fatalf("healing should be weaker under the mark: %v vs %v", half, full)
	}

	// and the world pays half
	if got := cruel.payout(40, now); got != 20 {
		t.Fatalf("payout halves: %d", got)
	}
	if got := plain.payout(40, now); got != 40 {
		t.Fatalf("a clean owner is paid in full: %d", got)
	}
}

func TestNewRestorativesHeal(t *testing.T) {
	for _, id := range []string{"banggac", "thuocbo", "canhtien", "nuocthanh"} {
		def := itemByID[id]
		if def == nil || def.Kind != "med" || def.Health <= 0 {
			t.Fatalf("%s should be a healing item: %+v", id, def)
		}
		st := huntPet(t)
		p := st.Pets[0]
		p.Health, p.Sick, p.Effect = 30, true, "dau"
		st.add(id, 1)
		if _, err := useItem(st, 0, id, 0, seeded(2)); err != nil {
			t.Fatal(err)
		}
		if p.Health <= 30 {
			t.Fatalf("%s did not heal: %v", id, p.Health)
		}
		if id == "canhtien" && p.Sick {
			t.Fatal("the broth cures illness")
		}
		if id == "nuocthanh" && (p.Sick || p.Effect != "") {
			t.Fatalf("holy water clears everything: sick %v effect %q", p.Sick, p.Effect)
		}
	}
}
