package repy

import (
	"sort"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestReadKnownREPY(t *testing.T) {
	want := []string{
		"אנרגיה",
		"ארכיטקטורה ובינוי ערים",
		"ביולוגיה",
		"הנדסה אזרחית וסביבתית",
		"הנדסה ביורפואית",
		"הנדסה כימית",
		"הנדסת אוירונוטיקה וחלל",
		"הנדסת ביוטכנולוגיה ומזון",
		"הנדסת חשמל",
		"הנדסת מכונות",
		"הנדסת תעשיה וניהול",
		"חינוך גופני",
		"חינוך למדע וטכנולוגיה",
		"כימיה",
		"לימודים הומניסטיים ואמנויות",
		"מדע והנדסה של חומרים",
		"מדעי המחשב",
		"מתמטיקה",
		"ננומדעים וננוטכנולוגיה",
		"פיסיקה",
		"רפואה",
	}

	catalog, err := ReadFile("REPY")
	if err != nil {
		t.Fatalf("Couldn't parse REPY: %v", err)
	}
	if catalog == nil {
		t.Fatalf("Got nil catalog")
	}

	got := make([]string, len(*catalog))

	for i, f := range *catalog {
		got[i] = f.Name
	}

	sort.Strings(got)

	if diff := pretty.Compare(want, got); diff != "" {
		t.Errorf("Mismatch in faculty list when parsing REPY (-want +got):\n%s", diff)
	}
}
