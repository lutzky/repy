package repy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func newParserFromString(s string) *parser {
	return &parser{
		course:  &Course{},
		scanner: bufio.NewScanner(bytes.NewBufferString(s)),
		logger:  GLogger{},
	}
}

func TestTimeOfDayToString(t *testing.T) {
	testCases := []struct {
		x    TimeOfDay
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
		want TimeOfDay
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

func TestParseCourse(t *testing.T) {
	// TODO(lutzky): This should be a glob
	testCases, err := filepath.Glob("testdata/courses/*.repy")
	if err != nil {
		t.Fatalf("Failed to glob for course REPYs: %v", err)
	}

	for _, fullPathRepy := range testCases {
		t.Run(filepath.Base(fullPathRepy), func(t *testing.T) {
			fullPathJson := strings.TrimSuffix(fullPathRepy, ".repy") + ".json"

			repyBytes, err := ioutil.ReadFile(fullPathRepy)
			if err != nil {
				t.Fatalf("Couldn't open %q: %v", fullPathRepy, err)
			}

			jsonBytes, err := ioutil.ReadFile(fullPathJson)
			if err != nil {
				t.Fatalf("Couldn't open %q: %v", fullPathJson, err)
			}

			var want Course
			if err := json.Unmarshal(jsonBytes, &want); err != nil {
				t.Fatalf("Couldn't unmarshal %q: %v", fullPathJson, err)
			}

			cp := newParserFromString(strings.TrimSpace(string(repyBytes)))

			cp.scan() // parseCourse() expects one line to be scanned already

			got, err := cp.parseCourse()

			if err != nil {
				t.Fatalf("Error parsing course: %v", err)
			} else if got == nil {
				t.Fatalf("Got a nil course")
			} else if diff := pretty.Compare(want, *got); diff != "" {
				var gotJSON string
				b, err := json.MarshalIndent(*got, "", "  ")
				if err == nil {
					gotJSON = string(b)
				} else {
					gotJSON = fmt.Sprintf("Couldn't emit JSON: %v", err)
				}

				t.Fatalf("Mismatch parsing course. Diff -want +got:\n%s\nFull 'got' in JSON:\n%s", diff, gotJSON)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	testCases := []struct {
		s, want string
	}{
		{"hello", "olleh"},
		{"הסדנה", "הנדסה"},
	}

	for _, tc := range testCases {
		got := Reverse(tc.s)
		if got != tc.want {
			t.Errorf("Reverse(a%qa) = a%qa; want a%qa", tc.s, got, tc.want)
		}
	}
}
