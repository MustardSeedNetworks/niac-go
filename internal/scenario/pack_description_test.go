package scenario_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// countWords are the number words a description could restate a count with, in
// both shipped locales. Digits are caught separately; these are the spelled-out
// half. Spanish "un" and "una" are left out on purpose: they are the indefinite
// article far more often than they are a count, and "solo" / "unico" catch the
// construction that actually restates one ("de un solo sitio").
func countWords() []string {
	return []string{
		"single", "one", "two", "three", "four", "five", "six", "seven", "eight",
		"nine", "ten", "dozen", "dual", "pair",
		"solo", "sola", "unico", "unica", "dos", "tres", "cuatro", "cinco", "seis",
		"siete", "ocho", "nueve", "diez", "docena", "par", "doble",
	}
}

var digit = regexp.MustCompile(`[0-9]`)

// statedCount reports the first count a description restates, or "".
//
// A pack description must not quote a device, site, or radio count: the pack
// picker already renders those from the manifest, so a number in the prose is
// a second copy that drifts the moment a count changes. Both drifted before
// this test existed -- enterprise-scale said 531 devices and built 543, and
// warehouse said 30 access points and built 27.
//
// "Wi-Fi 7" is a standard's name rather than a count, so it is excluded before
// the digit scan rather than exempted from it.
func statedCount(description string) string {
	scanned := strings.ReplaceAll(description, "Wi-Fi 7", "Wi-Fi")
	if found := digit.FindString(scanned); found != "" {
		return found
	}
	words := countWords()
	for word := range strings.FieldsSeq(foldAccents(strings.ToLower(scanned))) {
		trimmed := strings.Trim(word, ".,:;-")
		if slices.Contains(words, trimmed) {
			return trimmed
		}
	}

	return ""
}

// foldAccents maps the accented vowels the Spanish copy uses onto their plain
// forms, so one word list serves both locales.
func foldAccents(text string) string {
	return strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n",
	).Replace(text)
}

func TestPackDescriptionsStateNoCount(t *testing.T) {
	for _, pack := range scenario.Packs() {
		if found := statedCount(pack.Description); found != "" {
			t.Errorf("%s description restates a count (%q); the manifest is the one source: %q",
				pack.ID, found, pack.Description)
		}
	}
}

// TestLocalePackDescriptionsMatchThePacks holds the translated copies to the
// same rule and to the same pack list. The locales carry a second, independent
// description per pack -- which is what the UI actually renders -- so a pack
// added, renamed or retired here must reach both of them.
func TestLocalePackDescriptionsMatchThePacks(t *testing.T) {
	packs := scenario.Packs()
	for _, locale := range []string{"en", "es"} {
		path := filepath.Join("..", "i18n", "locales", locale, "pages.json")
		raw, err := os.ReadFile(path) // #nosec G304 -- fixed in-repo locale path
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		metadata := localePackMetadata(t, path, raw)
		if len(metadata) != len(packs) {
			t.Errorf("%s carries %d pack entries, want %d", locale, len(metadata), len(packs))
		}
		for _, pack := range packs {
			entry, found := metadata[pack.ID]
			if !found {
				t.Errorf("%s has no packMetadata entry for %q", locale, pack.ID)
				continue
			}
			if found := statedCount(entry.Description); found != "" {
				t.Errorf("%s %s description restates a count (%q): %q",
					locale, pack.ID, found, entry.Description)
			}
		}
	}
}

type localeEntry struct {
	Description string `json:"description"`
	Name        string `json:"name"`
}

func localePackMetadata(t *testing.T, path string, raw []byte) map[string]localeEntry {
	t.Helper()
	var document struct {
		NewSimWizard struct {
			Fleet struct {
				PackMetadata map[string]localeEntry `json:"packMetadata"`
			} `json:"fleet"`
		} `json:"newSimWizard"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	return document.NewSimWizard.Fleet.PackMetadata
}
