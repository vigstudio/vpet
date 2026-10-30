package game

import (
	"strings"
	"testing"
	"time"
)

func TestBoardCarePrepaidPackagesAndExtension(t *testing.T) {
	const day = int64(24 * time.Hour / time.Millisecond)
	now := time.Date(2026, 10, 26, 8, 0, 0, 0, vnZone).UnixMilli()
	st := newUserState()
	st.Coin = 2000
	p := &pet{Name: "Bông", Born: now - day, LastTick: now, Health: 30, Hunger: 10, Happy: 5, Sick: true}
	st.Pets = []*pet{p}

	msg, err := boardCare(st, 0, 7, now)
	if err != nil {
		t.Fatal(err)
	}
	if st.Coin != 1510 || p.CareUntil != now+7*day || p.Sick || p.Health != 70 || p.Hunger != 80 || p.Happy != 80 {
		t.Fatalf("bad admission: coin=%d pet=%+v", st.Coin, p)
	}
	if !strings.Contains(msg, "7 ngày") || !strings.Contains(msg, "490 vàng") {
		t.Fatalf("message %q", msg)
	}

	before := *p
	if p.tick(now + 6*day); p.Health != before.Health || p.Hunger != before.Hunger || p.Happy != before.Happy || p.Dead {
		t.Fatalf("care did not freeze decay: before=%+v after=%+v", before, p)
	}
	if _, err = boardCare(st, 0, 3, now+6*day); err != nil {
		t.Fatal(err)
	}
	if p.CareUntil != now+10*day || st.Coin != 1270 {
		t.Fatalf("extension discarded prepaid time: until=%d coin=%d", p.CareUntil, st.Coin)
	}
	p.tick(now + 11*day)
	if p.Hunger >= 80 || p.Happy >= 80 {
		t.Fatalf("decay did not resume after checkout: %+v", p)
	}
}

func TestBoardCareRejectsInvalidOrUnpaidStay(t *testing.T) {
	now := int64(1234)
	st := newUserState()
	st.Coin = 1000
	st.Pets = []*pet{{Name: "Sống", LastTick: now}, {Name: "Mất", Dead: true, LastTick: now}}
	if _, err := boardCare(st, 0, 2, now); err == nil {
		t.Fatal("unsupported package accepted")
	}
	if _, err := boardCare(st, 1, 1, now); err == nil {
		t.Fatal("dead pet admitted")
	}
	st.Coin = 89
	if _, err := boardCare(st, 0, 1, now); err == nil {
		t.Fatal("unpaid stay accepted")
	}
	if st.Coin != 89 || st.Pets[0].CareUntil != 0 {
		t.Fatal("refusal mutated state")
	}
}

func TestClinicInfoPublishesCarePrices(t *testing.T) {
	got := clinicInfo().Care
	if len(got) != 4 || got[0].Days != 1 || got[3].Days != 14 || got[3].Price >= got[0].Price*14 {
		t.Fatalf("bad care packages: %+v", got)
	}
}
