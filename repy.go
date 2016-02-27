package repy

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"golang.org/x/text/encoding/charmap"
)

// ReadFile reads filename, parses it as REPY, and returns a Catalog.
// TODO(lutzky): Determine if encoding can be auto-detected here.
func ReadFile(filename string) (*Catalog, error) {
	d := charmap.CodePage862.NewDecoder()
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Couldn't open %q: %v", filename, err)
	}

	p := parser{
		file:    filename,
		course:  &Course{},
		scanner: bufio.NewScanner(d.Reader(f)),
	}

	return p.parseFile()
}

// Catalog represents all of the information in a REPY file
type Catalog []Faculty

// Faculty represents a set of courses offered by a faculty.
type Faculty struct {
	name    string
	courses []Course
}

// Course represents information about a technion course.
type Course struct {
	id               uint
	name             string
	academicPoints   float32 // ...even though 2*points is always a uint :/
	lecturerInCharge string
	weeklyHours      WeeklyHours
	testDates        []Date
	groups           []Group
}

func (c Course) String() string {
	return fmt.Sprintf(
		"{Course[%d] (%q) AP:%.1f Hours:%v lecturer:%q testDates:%v groups:%v}",
		c.id,
		c.name,
		c.academicPoints,
		c.weeklyHours,
		c.lecturerInCharge,
		c.testDates,
		c.groups,
	)
}

// Date is a timezone-free representation of a date
type Date struct {
	Year, Month, Day uint
}

// TODO(lutzky): Why isn't this used when print("%v")ing a course?
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// WeeklyHours represents the amount of weekly hours, by type, that a course
// has.
type WeeklyHours struct {
	lecture  uint
	tutorial uint
	lab      uint
	project  uint
}

func (wh WeeklyHours) String() string {
	return fmt.Sprintf("Lec:%d,Tut:%d,Lab:%d", wh.lecture, wh.tutorial, wh.lab)
}

// GroupType is the type of events in a group (applies to all events within a
// group).
type GroupType int

const (
	// Lecture groups indicate frontal lectures by professors (הרצאה)
	Lecture GroupType = iota

	// Tutorial groups indicate frontal tutorials by TAs (תרגול, תרגיל)
	Tutorial

	// Lab groups indicate laboratory experiments (מעבדה)
	Lab
)

func (gt GroupType) String() string {
	return map[GroupType]string{
		Lecture:  "Lecture",
		Tutorial: "Tutorial",
		Lab:      "Lab",
	}[gt]
}

// Group represents a course's registration group (קבוצת רישום) and the events
// it entails.
type Group struct {
	id        uint
	teachers  []string
	events    []Event
	groupType GroupType
}

func (g Group) String() string {
	return fmt.Sprintf(
		"{group%d (%v) teachers:%q events:%v}",
		g.id,
		g.groupType,
		g.teachers,
		g.events,
	)
}

// Event represents a singular weekly event within a course.
type Event struct {
	day                time.Weekday
	location           string
	startHour, endHour TimeOfDay
}

func (e Event) String() string {
	return fmt.Sprintf(
		"{%v %v-%v at %q}",
		e.day,
		e.startHour,
		e.endHour,
		e.location,
	)
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

const (
	facultySep = "+==========================================+"
	courseSep  = "+------------------------------------------+"
	groupSep1  = "|               ++++++                  .סמ|"
	groupSep2  = "|                                     םושיר|"
	blankLine1 = "|                               -----      |"
	blankLine2 = "|                                          |"
)

var idAndNameRegex = regexp.MustCompile(`\| *(.*) +([0-9]{5,6}) \|`)

func (p *parser) parseIDAndName() error {
	m := idAndNameRegex.FindStringSubmatch(p.text())
	if m == nil {
		return p.errorf("Line %q doesn't match id-and-name regex `%s`", p.text(), idAndNameRegex)
	}

	p.course.name = hebrewFlip(m[1])
	p.course.id = p.parseUint(m[2])
	p.scan()
	return nil
}

func (p *parser) parseUint(s string) uint {
	result, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		panic(p.errorf("Couldn't ParseUint(%q, 10, 32): %v", s, err))
	}
	return uint(result)
}

func (p *parser) parseFloat(s string) float32 {
	result, err := strconv.ParseFloat(s, 32)
	if err != nil {
		panic(p.errorf("Couldn't ParseFloat(%q, 32): %v", s, err))
	}
	return float32(result)
}

func (p *parser) parseTotalHours(totalHours string) error {
	descriptors := strings.Fields(totalHours)
	for _, desc := range descriptors {
		bits := strings.Split(desc, "-")
		hours := p.parseUint(bits[0])
		switch bits[1] {
		case "ה":
			p.course.weeklyHours.lecture = hours
		case "ת":
			p.course.weeklyHours.tutorial = hours
		case "מ":
			p.course.weeklyHours.lab = hours
		case "פ":
			p.course.weeklyHours.project = hours
		default:
			return p.errorf("Invalid hour descriptor %q", bits[1])
		}
	}
	return nil
}

var hoursAndPointsRegex = regexp.MustCompile(`\| *([0-9]+\.[0-9]+) *:קנ *(([0-9]+-[התפמ] *)+):עובשב הארוה תועש *\|`)

func (p *parser) parseHoursAndPoints() error {
	m := hoursAndPointsRegex.FindStringSubmatch(p.text())
	if m == nil {
		return p.errorf("Line %q doesn't match hoursAndPointsRegex `%s`", p.text(), hoursAndPointsRegex)
	}

	p.course.academicPoints = p.parseFloat(m[1])
	if err := p.parseTotalHours(m[2]); err != nil {
		return err
	}
	p.scan()
	return nil
}

type parser struct {
	scanner *bufio.Scanner
	course  *Course
	line    uint
	file    string
	groupID uint
}

func (p *parser) errorf(format string, a ...interface{}) error {
	return fmt.Errorf("%s:%d: %s", p.file, p.line, fmt.Errorf(format, a...))
}

func (p *parser) infof(format string, a ...interface{}) {
	glog.Infof("%s:%d: %s", p.file, p.line, fmt.Sprintf(format, a...))
}
func (p *parser) warningf(format string, a ...interface{}) {
	glog.Warningf("%s:%d: %s", p.file, p.line, fmt.Sprintf(format, a...))
}

func (p *parser) scan() bool {
	p.line++
	result := p.scanner.Scan()
	glog.V(1).Infof("%s:%d: %s", p.file, p.line, p.text())
	return result
}

func (p *parser) text() string {
	return p.scanner.Text()
}

var (
	// testDateRegex currently ignores the test time present at the end of test
	// date lines.
	testDateRegex         = regexp.MustCompile(`\|.*([0-9]{2})/([0-9]{2})/([0-9]{2}) *'. +םוי *:.*דעומ +\|`)
	lecturerInChargeRegex = regexp.MustCompile(`\| *(.*) : *יארחא *הרומ *\|`)
)

var separatorLineRegex = regexp.MustCompile(`\| +-+ *\|`)

func (p *parser) parseCourseHeadInfo() error {
	for {
		if p.text() == groupSep1 {
			return nil
		}

		if separatorLineRegex.MatchString(p.text()) {
			// skip
		} else if m := testDateRegex.FindStringSubmatch(p.text()); m != nil {
			d := Date{
				2000 + p.parseUint(m[3]), // TODO(lutzky): Reverse Y2K bug :/
				p.parseUint(m[2]),
				p.parseUint(m[1]),
			}
			p.course.testDates = append(p.course.testDates, d)
		} else if m := lecturerInChargeRegex.FindStringSubmatch(p.text()); m != nil {
			p.course.lecturerInCharge = hebrewFlip(strings.TrimSpace(m[1]))
		} else {
			p.warningf("Ignored courseHeadInfo line %q", p.text())
		}

		if !p.scan() {
			return p.errorf("Reached EOF")
		}
	}
}

func newParserFromString(s string, name string) *parser {
	b := bytes.NewBufferString(s)
	p := parser{}
	p.file = name
	p.course = &Course{}
	p.scanner = bufio.NewScanner(b)
	return &p
}

var facultyNameRegexp = regexp.MustCompile(`\| *([א-ת ]+) *- *תועש תכרעמ *\|`)

func (p *parser) parseFacultyName() (string, error) {
	m := facultyNameRegexp.FindStringSubmatch(p.text())
	if m == nil {
		return "", p.errorf("Line %q doesn't match faculty name regex `%s`", p.text(), facultyNameRegexp)
	}
	p.scan()
	return strings.TrimSpace(m[1]), nil
}

func (p *parser) parseFile() (*Catalog, error) {
	p.scan()

	for strings.TrimSpace(p.text()) == "" {
		p.scan()
	}

	if err := p.expectLineAndAdvance(facultySep); err != nil {
		return nil, err
	}

	facultyName, err := p.parseFacultyName()
	if err != nil {
		return nil, err
	}

	// Throw away semester line
	p.scan()

	if err := p.expectLineAndAdvance(facultySep); err != nil {
		return nil, err
	}

	courses := []Course{}

	for p.text() != facultySep {
		course, err := p.parseCourse()
		if err != nil {
			return nil, err
		}
		courses = append(courses, *course)
	}

	return &Catalog{
		Faculty{
			name:    facultyName,
			courses: courses,
		},
	}, nil
}

func (p *parser) parseCourse() (*Course, error) {
	*p.course = Course{}
	// First groupID is usually omitted in REPY.
	p.groupID = 10

	if p.text() == "" {
		p.scan()
	}
	if p.text() == courseSep {
		p.scan()
	}

	if err := p.parseIDAndName(); err != nil {
		return nil, err
	}

	if err := p.parseHoursAndPoints(); err != nil {
		return nil, err
	}

	if err := p.expectLineAndAdvance(courseSep); err != nil {
		return nil, err
	}

	if err := p.parseCourseHeadInfo(); err != nil {
		return nil, err
	}

	// TODO(lutzky): There might be some comments about the course here

	if err := p.parseGroups(); err != nil {
		return nil, err
	}

	return p.course, nil
}

func (p *parser) expectLineAndAdvance(s string) error {
	if p.text() != s {
		return p.errorf("Expected %q, got %q", s, p.text())
	}
	p.scan()
	return nil
}

func (p *parser) weekDayFromHebrewLetter(letter string) time.Weekday {
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
		panic(p.errorf("Invalid weekday letter %q", letter))
	}

	return result
}

func (p *parser) timeOfDayFromStrings(hours, minutes string) TimeOfDay {
	h := p.parseUint(hours)
	m := p.parseUint(minutes)
	return TimeOfDay(h*60 + m)
}

func (p *parser) groupTypeFromString(s string) (GroupType, error) {
	mapping := map[string]GroupType{
		"האצרה": Lecture,
		"לוגרת": Tutorial,
		"ליגרת": Tutorial,
		"הדבעמ": Lab,
	}

	result, ok := mapping[s]
	if !ok {
		return 0, p.errorf("Invalid group type %q", s)
	}
	return result, nil
}

var standardLocationRegexp = regexp.MustCompile(`([א-ת]+) ([0-9]+)`)

func (p *parser) parseLocation(s string) string {
	m := standardLocationRegexp.FindStringSubmatch(s)
	if len(m) == 0 {
		return hebrewFlip(s)
	}
	building := hebrewFlip(m[1])
	room := p.parseUint(m[2])
	return fmt.Sprintf("%s %d", building, room)
}

func (p *parser) lastGroup() *Group {
	if len(p.course.groups) == 0 {
		p.course.groups = []Group{{}}
	}
	return &p.course.groups[len(p.course.groups)-1]
}

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

var lecturerRegexp = regexp.MustCompile(
	`\| *(.*) *: *הצרמ *\|`)

func (p *parser) parseLecturerLine() bool {
	if m := lecturerRegexp.FindStringSubmatch(p.text()); len(m) > 0 {
		lecturer := hebrewFlip(collapseSpaces(m[1]))
		teachers := &p.lastGroup().teachers
		*teachers = append(*teachers, lecturer)
		return true
	}
	return false
}

var eventRegexp = regexp.MustCompile(
	`\| *(.*) + ([0-9]{1,2})\.([0-9]{2})- *([0-9]{1,2})\.([0-9]{2})'([אבגדהוש]) :([א-ת]+) +([0-9]+)? *\|`)

func (p *parser) parseEventLine() bool {
	// TODO(lutzky): This is actually a group-and-event-at-once line
	if m := eventRegexp.FindStringSubmatch(p.text()); len(m) > 0 {
		p.infof("Parsed %s", p.text())
		ev := Event{
			day:       p.weekDayFromHebrewLetter(m[6]),
			startHour: p.timeOfDayFromStrings(m[2], m[3]),
			endHour:   p.timeOfDayFromStrings(m[4], m[5]),
			location:  p.parseLocation(m[1]),
		}

		groupType, err := p.groupTypeFromString(m[7])
		if err != nil {
			p.warningf("Failed to parse group type %q: %v", m[7], err)
			return false
		}

		group := Group{
			teachers:  []string{}, // TODO(lutzky): Fill these in
			events:    []Event{ev},
			groupType: groupType,
		}

		if m[8] != "" {
			group.id = p.parseUint(m[8])
			p.groupID = group.id + 1
		} else {
			group.id = p.groupID
			p.groupID++
		}

		p.course.groups = append(p.course.groups, group)

		p.scan()
		return true
	}
	return false
}

func (p *parser) parseGroups() error {
	if p.text() != groupSep1 {
		return p.errorf("Expected %q, got %q", groupSep1, p.text())
	}

	for {
		if p.text() == groupSep1 {
			p.scan()
			if err := p.expectLineAndAdvance(groupSep2); err != nil {
				return err
			}
		} else if p.text() == courseSep {
			p.scan()
			return nil
		} else if p.text() == blankLine1 || p.text() == blankLine2 {
			p.scan()
		} else if p.parseEventLine() {
			// TODO(lutzky): Do nothing?
		} else if p.parseLecturerLine() {
			p.scan()
		} else {
			p.warningf("Ignored group line %q", p.text())
			p.scan()
		}
	}
}
