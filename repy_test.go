package repy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

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
	testCases := []struct {
		name string
		data string
		want Course
	}{
		{"storage_systems", `
+------------------------------------------+
|                עדימ ןוסחא תוכרעמ  234322 |
|3.0 :קנ          1-ת 2-ה:עובשב הארוה תועש |
+------------------------------------------+
|             11/02/16 'ה  םוי: ןושאר דעומ |
|                              ----------- |
|             08/03/16 'ג  םוי:   ינש דעומ |
|                              ----------- |
|               ++++++                  .סמ|
|                                     םושיר|
|      בואט 009  10.30-12.30'ג :האצרה      |
|                רגדי.ג    ר"ד : הצרמ      |
|                               -----      |
|                                          |
|      בואט 005  17.30-18.30'ג :ליגרת  11  |
|                                          |
|      בואט 006  15.30-16.30'ד :ליגרת  12  |
|                                          |
|                     -        :ליגרת  13  |
+------------------------------------------+
`, Course{
			ID:             234322,
			Name:           "מערכות אחסון מידע",
			AcademicPoints: 3.0,
			WeeklyHours:    WeeklyHours{lecture: 2, tutorial: 1},
			TestDates: []Date{
				{2016, 2, 11},
				{2016, 3, 8},
			},
			Groups: []Group{
				{
					id:       10,
					teachers: []string{`ד"ר ג.ידגר`},
					events: []Event{
						{day: 2, location: "טאוב 9", startHour: 10*60 + 30, endHour: 12*60 + 30},
					},
					groupType: Lecture,
				},
				{
					id: 11,
					events: []Event{
						{day: 2, location: "טאוב 5", startHour: 17*60 + 30, endHour: 18*60 + 30},
					},
					groupType: Tutorial,
				},
				{
					id: 12,
					events: []Event{
						{day: 3, location: "טאוב 6", startHour: 15*60 + 30, endHour: 16*60 + 30},
					},
					groupType: Tutorial,
				},
			},
		}},
		{"statistics", `
+------------------------------------------+
|                        הקיטסיטטס  014003 |
|3.0 :קנ          2-ת 2-ה:עובשב הארוה תועש |
+------------------------------------------+
|           ןייבשיפ.ב        : יארחא  הרומ |
|                              ----------- |
|   9.00  העש 28/01/16 'ה  םוי: ןושאר דעומ |
|                              ----------- |
|   9.00  העש 26/02/16 'ו  םוי:   ינש דעומ |
|                              ----------- |
|         דבלב היזדואיגל תדעוימ 13 הצובק.1 |
|               ++++++                  .סמ|
|                                     םושיר|
|      ןיבר 206  14.30-16.30'ג :האצרה      |
|             ןייבשיפ.ב מ/פורפ : הצרמ      |
|                               -----      |
+------------------------------------------+
`, Course{
			ID:               14003,
			Name:             "סטטיסטיקה",
			LecturerInCharge: "ב.פישביין",
			AcademicPoints:   3.0,
			WeeklyHours: WeeklyHours{
				lecture:  2,
				tutorial: 2,
			},
			TestDates: []Date{
				{2016, 01, 28},
				{2016, 02, 26},
			},
			Groups: []Group{
				{
					id:       10,
					teachers: []string{"פרופ/מ ב.פישביין"},
					events: []Event{
						{
							day:       2,
							startHour: 14*60 + 30,
							endHour:   16*60 + 30,
							location:  "רבין 206",
						},
					},
				},
			},
		}},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cp := newParserFromString(strings.TrimSpace(tc.data),
				fmt.Sprintf("testParseCourse%d", i))
			// parseCourse() expects one line to be scanned already
			cp.scan()
			got, err := cp.parseCourse()
			if err != nil {
				t.Fatalf("Error parsing course: %v", err)
			} else if got == nil {
				t.Fatalf("Got a nil course")
			} else if diff := pretty.Compare(tc.want, *got); diff != "" {
				t.Fatalf("Mismatch parsing course. Diff -want +got:\n%s", diff)
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
