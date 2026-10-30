package game

import (
	"strings"
	"testing"
)

// Vietnamese is the source of truth, so a missing translation must fall back to it rather
// than reaching the player as an empty label.
func TestEveryCatalogEntryIsTranslated(t *testing.T) {
	for _, lang := range []string{"en", "ja"} {
		items := translatedCatalog(lang)
		if len(items) != len(catalog) {
			t.Fatalf("%s: catalog lost entries: %d vs %d", lang, len(items), len(catalog))
		}
		for i, it := range items {
			if it.ID != catalog[i].ID {
				t.Fatalf("%s: item order changed at %d", lang, i)
			}
			if it.Name == "" {
				t.Fatalf("%s: %s has no name", lang, it.ID)
			}
			// Some proper nouns are identical across languages (Sushi), so the description
			// is the honest signal that an entry was actually translated.
			if catalog[i].Desc != "" && it.Desc == catalog[i].Desc && !isGemOrShard(it.ID) {
				t.Errorf("%s: %s still reads Vietnamese %q", lang, it.ID, it.Desc)
			}
		}
		for _, sk := range translatedSkills(lang) {
			if sk.Name == "" || sk.Desc == "" {
				t.Errorf("%s: skill %s is untranslated", lang, sk.ID)
			}
		}
		for _, m := range translatedMaps(lang) {
			if m.Name == "" || m.Desc == "" {
				t.Errorf("%s: map %s is untranslated", lang, m.ID)
			}
		}
		if got := translatedRarities(lang); len(got) != len(rarityNames) {
			t.Errorf("%s: rarity list changed length", lang)
		}
	}
}

// Gems and shards are generated from the gem tables, which still read Vietnamese; they are
// named by tier so the text is mostly numbers. This test records that gap deliberately.
func isGemOrShard(id string) bool {
	return id == "gem_shard" || strings.HasPrefix(id, "gem_")
}

func TestTranslationNeverMutatesTheSourceTables(t *testing.T) {
	before := catalog[0].Name
	translatedCatalog("en")
	translatedCatalog("ja")
	if catalog[0].Name != before {
		t.Fatalf("translation wrote back into the catalog: %q", catalog[0].Name)
	}
	if n := translatedSkills("en")[0].Name; n == skills[0].Name {
		t.Fatalf("skills were not translated: %q", n)
	}
	if skills[0].Name != "Phun lửa" {
		t.Fatalf("skill table was mutated: %q", skills[0].Name)
	}
}

func TestLocaleFallsBackToVietnamese(t *testing.T) {
	for raw, want := range map[string]string{
		"en": "en", "en-AU": "en", "ja": "ja", "ja_JP": "ja",
		"vi": "vi", "fr": "vi", "": "vi", "e": "vi",
	} {
		if got := localeOf(raw); got != want {
			t.Errorf("localeOf(%q) = %q, want %q", raw, got, want)
		}
	}
	if got := translatedCatalog("fr")[0].Name; got != catalog[0].Name {
		t.Fatalf("unknown locale should read Vietnamese, got %q", got)
	}
}
