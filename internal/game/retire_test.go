package game

import "testing"

// 1.9.0 deleted the charm slot and renamed the ring tiers without touching saved bags, so
// 63 charms and 2 rings sat in production inventories that the bag could not draw. These
// tests pin the conversion and make a future removal fail loudly instead of silently.
func TestObsoleteItemsConvertOnLoad(t *testing.T) {
	st := &userState{Inv: map[string]int{"buamay": 3, "nhan": 2, "kiem": 1}}
	st.normalize()
	if st.Inv["buamay"] != 0 || st.Inv["nhan"] != 0 {
		t.Fatalf("obsolete ids survived: %v", st.Inv)
	}
	if st.Inv["nhandong"] != 3 || st.Inv["nhanbac"] != 2 || st.Inv["kiem"] != 1 {
		t.Fatalf("conversion lost or duplicated goods: %v", st.Inv)
	}
}

func TestObsoleteGearComesOffAndIsReplaced(t *testing.T) {
	st := &userState{
		Inv:  map[string]int{},
		Pets: []*pet{{Name: "a", Born: 1, Gear: map[string]string{"bua": "buamay", "vukhi": "kiem"}}},
	}
	st.normalize()
	g := st.Pets[0].Gear
	if g["bua"] != "" {
		t.Fatalf("a slot the game no longer has is still worn: %v", g)
	}
	if g["nhan"] != "nhandong" {
		t.Fatalf("the converted piece should move to its real slot: %v", g)
	}
	if g["vukhi"] != "kiem" {
		t.Fatalf("untouched gear changed: %v", g)
	}
}

// Nothing a player can hold may be missing from the catalog: the bag, the market and the
// shop all describe items through it, so an id without an entry is invisible.
func TestEveryReachableItemIsInTheCatalog(t *testing.T) {
	known := map[string]bool{}
	for _, it := range catalog {
		known[it.ID] = true
	}
	for id := range gearByID {
		if !known[id] {
			t.Errorf("gear %q has no catalog entry, so the bag cannot show it", id)
		}
	}
	for _, now := range retired {
		if !known[now] {
			t.Errorf("retired items convert to %q, which is not in the catalog", now)
		}
	}
	for old := range retired {
		if known[old] {
			t.Errorf("%q is retired but still in the catalog; drop it from the retired table", old)
		}
	}
}
