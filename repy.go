package repy

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Faculty []Course

type Course struct {
	id               uint
	name             string
	academicPoints   float32 // ...even though 2*points is always a uint :/
	lecturerInCharge string
	weeklyHours      WeeklyHours
	testDates        []Date
	groups           []Group
}

// Date is a timezone-free representation of a date
type Date struct {
	Year, Month, Day uint
}

// TODO(lutzky): Why isn't this used when print("%v")ing a course?
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

type WeeklyHours struct {
	lecture  uint
	tutorial uint
	lab      uint
}

type GroupType int

const (
	gtLecture = iota
	gtTutorial
	gtLab
)

func (gt GroupType) String() string {
	return map[GroupType]string{
		gtLecture:  "Lecture",
		gtTutorial: "Tutorial",
		gtLab:      "Lab",
	}[gt]
}

type Group struct {
	id        uint
	teachers  []string
	events    []Event
	groupType GroupType
}

type Event struct {
	day                time.Weekday
	location           string
	startHour, endHour TimeOfDay
}

// TimeOfDay is represented as "minutes since midnight".
type TimeOfDay uint

func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t/60, t%60)
}

func parseTimeOfDay(x string) (TimeOfDay, error) {
	sections := strings.Split(strings.TrimSpace(x), ".")

	if len(sections) != 2 {
		return 0, fmt.Errorf("Invalid TimeOfDay: %q", x)
	}

	result := uint(0)

	for _, section := range sections {
		result *= 60
		n, err := strconv.ParseUint(section, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("Invalid TimeOfDay: %q (%v)", x, err)
		}
		result += uint(n)
	}

	return TimeOfDay(result), nil
}

func hebrewFlip(s string) string {
	runes := []rune(strings.TrimSpace(s))
	for i, j := 0, len(runes)-1; i < len(runes)/2; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

const courseSep = "+------------------------------------------+"
const groupSep1 = "|               ++++++                  .סמ|"
const groupSep2 = "|                                     םושיר|"

func (cp *courseParser) parseIdAndName() error {
	re := regexp.MustCompile(`\| *(.*) +([0-9]{5,6}) \|`)
	if m := re.FindStringSubmatch(cp.text()); m == nil {
		return cp.errorf("Line %q doesn't match id-and-name regex `%s`", cp.text(), re)
	} else {
		cp.course.name = hebrewFlip(m[1])
		cp.course.id = cp.parseUint(m[2])
		cp.scan()
		return nil
	}
}

func (cp *courseParser) parseUint(s string) uint {
	result, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		panic(cp.errorf("Couldn't ParseUint(%q, 10, 32): %v", s, err))
	}
	return uint(result)
}

func (cp *courseParser) parseFloat(s string) float32 {
	result, err := strconv.ParseFloat(s, 32)
	if err != nil {
		panic(cp.errorf("Couldn't ParseFloat(%q, 32): %v", s, err))
	}
	return float32(result)
}

func (cp *courseParser) parseTotalHours(totalHours string) error {
	descriptors := strings.Split(totalHours, " ")
	for _, desc := range descriptors {
		bits := strings.Split(desc, "-")
		hours := cp.parseUint(bits[0])
		switch bits[1] {
		case "ה":
			cp.course.weeklyHours.lecture = hours
		case "ת":
			cp.course.weeklyHours.tutorial = hours
		case "מ":
			cp.course.weeklyHours.lab = hours
		default:
			return cp.errorf("Invalid hour descriptor %q", bits[1])
		}
	}
	return nil
}

func (cp *courseParser) parseHoursAndPoints() error {
	re := regexp.MustCompile(`\| *([0-9]+\.[0-9]+) *:קנ *(([0-9]-[התמ] *)+):עובשב הארוה תועש *\|`)
	if m := re.FindStringSubmatch(cp.text()); m == nil {
		return cp.errorf("Line %q doesn't match hours-and-points regex `%s`", cp.text(), re)
	} else {
		var academicPoints float64
		cp.course.academicPoints = cp.parseFloat(m[1])
		cp.course.academicPoints = float32(academicPoints)
		if err := cp.parseTotalHours(m[2]); err != nil {
			return err
		}
		cp.scan()
		return nil
	}
}

// TODO(lutzky): The logic for courseParser should be shared with faculty
// parsers, etc.
type courseParser struct {
	scanner *bufio.Scanner
	course  *Course
	line    uint
	file    string
}

func (cp *courseParser) errorf(format string, a ...interface{}) error {
	return fmt.Errorf("%s:%d: %s", cp.file, cp.line, fmt.Errorf(format, a...))
}

func (cp *courseParser) logf(format string, a ...interface{}) {
	log.Printf("%s:%d: %s", cp.file, cp.line, fmt.Sprintf(format, a...))
}

func (cp *courseParser) scan() {
	cp.scanner.Scan()
	cp.line += 1
}

func (cp *courseParser) text() string {
	return cp.scanner.Text()
}

func (cp *courseParser) getTestDateFromLine(line string) (Date, bool) {
	// TODO(lutzky): Shouldn't be necessary to pass line here, cp.text() should do
	// it.
	testDate := regexp.MustCompile(`\| *([0-9]{2})/([0-9]{2})/([0-9]{2}) *'. +םוי *:.*דעומ +\|`)
	m := testDate.FindStringSubmatch(line)
	if m == nil {
		return Date{}, false
	}
	return Date{
		2000 + cp.parseUint(m[3]), // TODO(lutzky): Reverse Y2K bug :/
		cp.parseUint(m[2]),
		cp.parseUint(m[1]),
	}, true
}

func (cp *courseParser) parseTestDates() error {
	// TODO(lutzky): All regexes should be compiled once, ahead of time.
	separatorLine := regexp.MustCompile(`\| +-+ *\|`)
	for {
		if separatorLine.MatchString(cp.text()) {
			cp.scan()
			continue
		}
		if testDate, ok := cp.getTestDateFromLine(cp.text()); !ok {
			// Test date section has ended
			if len(cp.course.testDates) == 0 {
				cp.logf("WARNING: No tests found")
			}
			return nil
		} else {
			cp.course.testDates = append(cp.course.testDates, testDate)
			cp.scan()
		}
	}

	return nil
}

func newCourseParserFromString(s string, name string) *courseParser {
	b := bytes.NewBufferString(s)
	cp := courseParser{}
	cp.file = name
	cp.course = &Course{}
	cp.scanner = bufio.NewScanner(b)
	return &cp
}

func (cp *courseParser) parse() (*Course, error) {
	cp.scan()
	if err := cp.expectLineAndAdvance(courseSep); err != nil {
		return nil, err
	}

	if err := cp.parseIdAndName(); err != nil {
		return nil, err
	}

	if err := cp.parseHoursAndPoints(); err != nil {
		return nil, err
	}

	if err := cp.expectLineAndAdvance(courseSep); err != nil {
		return nil, err
	}

	if err := cp.parseTestDates(); err != nil {
		return nil, err
	}

	// TODO(lutzky): There might be some comments about the course here

	if err := cp.parseGroups(); err != nil {
		return nil, err
	}

	return cp.course, nil
}

func (cp *courseParser) expectLineAndAdvance(s string) error {
	if cp.text() != s {
		return cp.errorf("Expected %q, got %q", s, cp.text())
	}
	cp.scan()
	return nil
}

func (cp *courseParser) weekDayFromHebrewLetter(letter string) time.Weekday {
	mapping := map[string]time.Weekday{
		"א": time.Sunday,
		"ב": time.Monday,
		"ג": time.Tuesday,
		"ד": time.Wednesday,
		"ה": time.Thursday,
		"ו": time.Friday,
		"ש": time.Saturday,
	}

	result, ok := mapping[letter]
	if !ok {
		panic(cp.errorf("Invalid weekday letter %q", letter))
	}

	return result
}

func (cp *courseParser) timeOfDayFromStrings(hours, minutes string) TimeOfDay {
	h := cp.parseUint(hours)
	m := cp.parseUint(minutes)
	return TimeOfDay(h*60 + m)
}

func (cp *courseParser) groupTypeFromString(s string) (GroupType, error) {
	mapping := map[string]GroupType{
		"האצרה": gtLecture,
		"לוגרת": gtTutorial,
		"הדבעמ": gtLab,
	}

	result, ok := mapping[s]
	if !ok {
		return 0, cp.errorf("Invalid group type %q", s)
	}
	return result, nil
}

func (cp *courseParser) parseLocation(s string) string {
	// TODO(lutzky): This requires reversal of the Hebrew, but not of the number.
	return s
}

var eventRegexp = regexp.MustCompile(
	`\| *(.*) + ([0-9]{1,2})\.([0-9]{2})- *([0-9]{1,2})\.([0-9]{2})'([אבגדהוש]) :([א-ת]+) *\|`)

func (cp *courseParser) parseEventLine() bool {
	// TODO(lutzky): This is actually a group-and-event-at-once line
	if m := eventRegexp.FindStringSubmatch(cp.text()); len(m) > 0 {
		cp.logf("Parsed %s", cp.text())
		ev := Event{
			day:       cp.weekDayFromHebrewLetter(m[6]),
			startHour: cp.timeOfDayFromStrings(m[2], m[3]),
			endHour:   cp.timeOfDayFromStrings(m[4], m[5]),
			location:  cp.parseLocation(m[1]),
		}

		groupType, err := cp.groupTypeFromString(m[7])
		if err != nil {
			cp.logf("ERROR: %v", err)
			return false
		}

		group := Group{
			id:        42,         // TODO(lutzky): Actual group ID by state
			teachers:  []string{}, // TODO(lutzky): Fill these in
			events:    []Event{ev},
			groupType: groupType,
		}

		cp.course.groups = append(cp.course.groups, group)

		cp.scan()
		return true
	}
	return false
}

func (cp *courseParser) parseGroups() error {
	var groupId uint

	if cp.text() != groupSep1 {
		return cp.errorf("Expected %q, got %q", groupSep1, cp.text())
	}

	for {
		if cp.text() == groupSep1 {
			cp.scan()
			if err := cp.expectLineAndAdvance(groupSep2); err != nil {
				return err
			}
			groupId = (10*(groupId/10) + 1)
		} else if cp.text() == courseSep {
			cp.scan()
			return nil
		} else if cp.parseEventLine() {
			// TODO(lutzky): Do nothing?
		} else {
			cp.logf("WARNING: Ignored line %q", cp.text())
			cp.scan()
		}
	}

	return cp.errorf("Unexpected end of parseGroups()")
}
