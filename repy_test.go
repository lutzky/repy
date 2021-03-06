package repy

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

var update = flag.Bool("update", false, "update golden files in testdata")
var onlyTestREPY = flag.String("only_test_repy", "", "only test the given REPY file")

func TestTimeOfDayToString(t *testing.T) {
	testCases := []struct {
		x    MinutesSinceMidnight
		want string
	}{
		{0, "00:00"},
		{60, "01:00"},
		{90, "01:30"},
	}

	for _, tc := range testCases {
		got := tc.x.String()
		if got != tc.want {
			t.Errorf("%v.String() == %q; want %q", tc.x, got, tc.want)
		}
	}
}

func TestTimeOfDayFromString(t *testing.T) {
	testCases := []struct {
		x    string
		want MinutesSinceMidnight
	}{
		{"6.30", 6*60 + 30},
		{" 6.30", 6*60 + 30},
		{"16.30", 16*60 + 30},
		{"16.00", 16 * 60},
	}

	for _, tc := range testCases {
		got, err := parseTimeOfDay(tc.x)
		if err != nil {
			t.Errorf("parseTimeOfDay(%q) -> %v", tc.x, err)
		} else if got != tc.want {
			t.Errorf("parseTimeOfDay(%q) == %s; want %s", tc.x, got, tc.want)
		}
	}
}

func TestParseLocation(t *testing.T) {
	cp := parser{}

	testCases := []struct{ s, want string }{
		{"בואט 009", "טאוב 9"},
		{"ןמלוא 501", "אולמן 501"},
		{"והשלכ רחא הנבמ", "מבנה אחר כלשהו"},
	}

	for _, tc := range testCases {
		got := cp.parseLocation(tc.s)
		if got != tc.want {
			t.Errorf("cp.parseLocation(%q) == %q; want %q", tc.s, got, tc.want)
		}
	}
}

func getAllREPYs() []string {
	glob := "testdata/*.repy"
	if *onlyTestREPY != "" {
		glob = *onlyTestREPY
	}
	result, err := filepath.Glob(glob)
	if err != nil {
		panic(fmt.Sprintf("Failed to glob %q for course REPYs: %v", glob, err))
	}
	if len(result) == 0 {
		panic(fmt.Sprintf("Got 0 testcases in %q", glob))
	}

	return result
}

func BenchmarkParse(b *testing.B) {
	for _, fullPathRepy := range getAllREPYs() {
		b.Run(filepath.Base(fullPathRepy), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				repyFile, err := os.Open(fullPathRepy)
				if err != nil {
					b.Fatalf("Couldn't open %q: %v", fullPathRepy, err)
				}

				ReadFile(repyFile, GLogger{})
			}
		})
	}
}

func TestParse(t *testing.T) {
	for _, fullPathRepy := range getAllREPYs() {
		t.Run(filepath.Base(fullPathRepy), func(t *testing.T) {
			fullPathJSON := strings.TrimSuffix(fullPathRepy, ".repy") + ".json"

			repyFile, err := os.Open(fullPathRepy)
			if err != nil {
				t.Fatalf("Couldn't open %q: %v", fullPathRepy, err)
			}

			got, err := ReadFile(repyFile, GLogger{})

			if err != nil {
				t.Fatalf("Error parsing course: %v", err)
			} else if got == nil {
				t.Fatalf("Got a nil course")
			}

			jsonGot, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal golden JSON: %v", err)
			}

			if *update {
				if err := ioutil.WriteFile(fullPathJSON, jsonGot, 0644); err != nil {
					t.Fatalf("Failed to write golden JSON file %q: %v", fullPathJSON, err)
				}
			}

			jsonWant, err := ioutil.ReadFile(fullPathJSON)
			if err != nil {
				t.Fatalf("Couldn't open %q: %v", fullPathJSON, err)
			}

			if !bytes.Equal(jsonGot, jsonWant) {
				t.Fatal("Parsed JSON differs from expected. Use go test -update and git diff.")
			}
		})
	}
}

func TestSportLineParse(t *testing.T) {
	testCases := []struct {
		name string
		s    string
		want map[string]string
	}{
		{
			"FullEventWithGroupNumber",
			`|             יטלתא.צא 19.00-20.30'ב  .ברועמ -הלק.לתא 11        |`,
			map[string]string{
				"location":    "יטלתא.צא",
				"startHour":   "19",
				"startMinute": "00",
				"endHour":     "20",
				"endMinute":   "30",
				"weekday":     "ב",
				"description": ".ברועמ -הלק.לתא",
				"groupID":     "11",
			},
		},
		{
			"EventWithoutGroupNumber",
			`|             יטלתא.צא 19.00-20.30'ד                            |`,
			map[string]string{
				"location":    "יטלתא.צא",
				"startHour":   "19",
				"startMinute": "00",
				"endHour":     "20",
				"endMinute":   "30",
				"weekday":     "ד",
				"description": "",
				"groupID":     "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findStringSubmatchMap(sportLineRegexp, tc.s)
			if d := pretty.Compare(tc.want, got); d != "" {
				t.Fatalf("Diff -want +got:\n%s", d)
			}
		})
	}
}

func TestEventLineParse(t *testing.T) {
	testCases := []struct {
		name string
		s    string
		want map[string]string
	}{
		{
			"FullEventWithGroupNumber",
			`|                10.30-12.30'ב :ליגרת  11  |`,
			map[string]string{
				"groupID":     "11",
				"location":    "",
				"startHour":   "10",
				"startMinute": "30",
				"endHour":     "12",
				"endMinute":   "30",
				"groupType":   "ליגרת",
				"weekday":     "ב",
			},
		},
		{
			"FullEventWithoutGroupNumber",
			`|                12.30-14.30'א :האצרה      |`,
			map[string]string{
				"groupID":     "",
				"location":    "",
				"startHour":   "12",
				"startMinute": "30",
				"endHour":     "14",
				"endMinute":   "30",
				"groupType":   "האצרה",
				"weekday":     "א",
			},
		},
		{
			"EventWithoutGroupType",
			`|                12.30-14.30'ג             |`,
			map[string]string{
				"groupID":     "",
				"location":    "",
				"startHour":   "12",
				"startMinute": "30",
				"endHour":     "14",
				"endMinute":   "30",
				"groupType":   "",
				"weekday":     "ג",
			},
		},
		{
			"EventWithLocation",
			`|       וגס ידוא 12.30-14.30'ב :האצרה      |`,
			map[string]string{
				"groupID":     "",
				"location":    "וגס ידוא",
				"startHour":   "12",
				"startMinute": "30",
				"endHour":     "14",
				"endMinute":   "30",
				"groupType":   "האצרה",
				"weekday":     "ב",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findStringSubmatchMap(eventRegexp, tc.s)
			if d := pretty.Compare(tc.want, got); d != "" {
				t.Fatalf("Diff -want +got:\n%s", d)
			}
		})
	}
}

func TestFixTwoDigitYear(t *testing.T) {
	testCases := []struct {
		baseYear, want uint
	}{
		{2, 2002},
		{18, 2018},
		{85, 2085},
		{99, 2099},
		{105, 105},
		{1985, 1985},
	}

	for _, tc := range testCases {
		got := fixTwoDigitYear(tc.baseYear)
		if got != tc.want {
			t.Errorf("fixTwoDigitYear(%d) = %d; want %d", tc.baseYear, got, tc.want)
		}
	}
}
