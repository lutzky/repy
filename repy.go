// Package repy parses REPY files into Catalog objects, intended for conversion
// into JSON.
package repy

import (
	"bufio"
	"fmt"
	"io"
	"path"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"golang.org/x/text/encoding/charmap"
)

// Logger is an interface for passing a logger to ReadFile
type Logger interface {
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})

	// Flush will automatically be called after parsing is complete
	Flush()
}

// GLogger is a Logger that uses glog
type GLogger struct{}

// Infof implements Logger.Infof
func (g GLogger) Infof(format string, args ...interface{}) {
	glog.InfoDepth(3, fmt.Sprintf(format, args...))
}

// Warningf implements Logger.Warningf
func (g GLogger) Warningf(format string, args ...interface{}) {
	glog.WarningDepth(3, fmt.Sprintf(format, args...))
}

// Flush implements Logger.Flush
func (g GLogger) Flush() {
	glog.Flush()
}

// ReadFile reads repyReader, parses it as REPY, and returns a Catalog. If
// logger is not nil, log messages will be sent to it.
func ReadFile(repyReader io.Reader, logger Logger) (c *Catalog, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to read REPY: %v\n%s", r, debug.Stack())
		}
	}()
	defer logger.Flush()
	d := charmap.CodePage862.NewDecoder()

	p := parser{
		course:  &Course{},
		scanner: bufio.NewScanner(d.Reader(repyReader)),
		logger:  logger,
	}

	return p.parseFile()
}

func (f Faculty) String() string {
	return fmt.Sprintf("faculty(%s, %d)", f.Name, len(f.Courses))
}

func (c Course) String() string {
	return fmt.Sprintf(
		"{Course[%d] (%q) AP:%.1f Hours:%v lecturer:%q testDates:%v groups:%v}",
		c.ID,
		c.Name,
		c.AcademicPoints,
		c.WeeklyHours,
		c.LecturerInCharge,
		c.TestDates,
		c.Groups,
	)
}

func (t MinutesSinceMidnight) String() string {
	return fmt.Sprintf("%02d:%02d", t/60, t%60)
}

func parseTimeOfDay(x string) (MinutesSinceMidnight, error) {
	sections := strings.Split(strings.TrimSpace(x), ".")

	if len(sections) != 2 {
		return 0, errors.Errorf("Invalid TimeOfDay: %q", x)
	}

	result := uint(0)

	for _, section := range sections {
		result *= 60
		n, err := strconv.ParseUint(section, 10, 32)
		if err != nil {
			return 0, errors.Wrapf(err, "invalid TimeOfDay: %q", x)
		}
		result += uint(n)
	}

	return MinutesSinceMidnight(result), nil
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

	sportsFacultySep = "+===============================================================+"
	sportsCourseSep  = "+---------------------------------------------------------------+"
	sportsBlankLine1 = "|                                             -----------       |"
	sportsBlankLine2 = "|                                                               |"
)

var idAndNameRegex = regexp.MustCompile(`\| *(.*) +([0-9]{5,6}) +\|`)

func (p *parser) parseIDAndName() error {
	m := idAndNameRegex.FindStringSubmatch(p.text())
	if m == nil {
		return p.errorf("Line %q doesn't match id-and-name regex `%s`", p.text(), idAndNameRegex)
	}

	p.course.Name = hebrewFlip(m[1])
	p.course.ID = p.parseUint(m[2])
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
			p.course.WeeklyHours.Lecture = hours
		case "ת":
			p.course.WeeklyHours.Tutorial = hours
		case "מ":
			p.course.WeeklyHours.Lab = hours
		case "פ":
			p.course.WeeklyHours.Project = hours
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

	p.course.AcademicPoints = p.parseFloat(m[1])
	if err := p.parseTotalHours(m[2]); err != nil {
		return errors.Wrapf(err, "couldn't parse total-hours %q in hours-and-points line", m[2])
	}
	p.scan()
	return nil
}

type parser struct {
	scanner *bufio.Scanner
	course  *Course
	line    uint
	groupID uint
	logger  Logger

	numEOFHits int
	eof        bool
}

func (p *parser) errorfSkip(skip int, format string, a ...interface{}) error {
	var caller string
	if _, file, line, ok := runtime.Caller(skip); ok {
		caller = fmt.Sprintf("[%s:%d] ", path.Base(file), line)
	}
	return errors.Errorf("%sLine %d: %s", caller, p.line, errors.Errorf(format, a...))
}

func (p *parser) errorf(format string, a ...interface{}) error {
	return p.errorfSkip(2, format, a...)
}

func (p *parser) infof(format string, a ...interface{}) {
	if p.logger != nil {
		p.logger.Infof("Line %d: %s", p.line, fmt.Sprintf(format, a...))
	}
}
func (p *parser) warningf(format string, a ...interface{}) {
	if p.logger != nil {
		p.logger.Warningf("Line %d: %s", p.line, fmt.Sprintf(format, a...))
	}
}

func (p *parser) scan() bool {
	result := p.scanner.Scan()
	if result {
		p.line++
	} else {
		if err := p.scanner.Err(); err != nil {
			panic(err)
		}
		p.infof("Hit EOF, numEOFHits is %d. Stack trace:\n%s", p.numEOFHits, string(debug.Stack()))
		if p.numEOFHits > 10 {
			panic("Hit EOF too many times")
		}
		p.numEOFHits++
		p.eof = true
	}
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

func fixTwoDigitYear(baseYear uint) uint {
	if baseYear < 100 {
		return 2000 + baseYear
	}

	return baseYear
}

func (p *parser) parseCourseHeadInfo() error {
	for {
		if p.text() == groupSep1 || p.text() == courseSep {
			return nil
		}

		if separatorLineRegex.MatchString(p.text()) {
			// skip
		} else if m := testDateRegex.FindStringSubmatch(p.text()); m != nil {
			d := Date{
				fixTwoDigitYear(p.parseUint(m[3])),
				p.parseUint(m[2]),
				p.parseUint(m[1]),
			}
			p.course.TestDates = append(p.course.TestDates, d)
		} else if m := lecturerInChargeRegex.FindStringSubmatch(p.text()); m != nil {
			p.course.LecturerInCharge = hebrewFlip(strings.TrimSpace(m[1]))
		} else {
			p.infof("Ignored courseHeadInfo line %q", p.text())
		}

		if !p.scan() {
			return p.errorf("Reached EOF")
		}
	}
}

var (
	facultyNameRegexp     = regexp.MustCompile(`\| *([א-ת\., ]+) *- *תועש תכרעמ *\|`)
	facultySemesterRegexp = regexp.MustCompile(`\| *([א-ת" ]+) +רטסמס *\|`)
)

func (p *parser) parseFacultyName() (string, error) {
	m := facultyNameRegexp.FindStringSubmatch(p.text())
	if m == nil {
		return "", p.errorf("Line %q doesn't match faculty name regex `%s`", p.text(), facultyNameRegexp)
	}
	p.scan()
	return Reverse(strings.TrimSpace(m[1])), nil
}

func (p *parser) parseFacultySemester() (string, error) {
	m := facultySemesterRegexp.FindStringSubmatch(p.text())
	if m == nil {
		return "", p.errorf("Line %q doesn't match faculty semester regex `%s`", p.text(), facultySemesterRegexp)
	}
	p.scan()
	return Reverse(strings.TrimSpace(m[1])), nil
}

func (p *parser) parseFile() (*Catalog, error) {
	catalog := Catalog{}

faculties:
	for {
		catalog = append(catalog, Faculty{})
		currentFaculty := &catalog[len(catalog)-1]
		switch err := p.parseFaculty(currentFaculty); err {
		case nil: // Do nothing
		case io.EOF:
			// Drop last faculty added, as it doesn't actually contain anything
			catalog = catalog[0 : len(catalog)-1]
			break faculties
		default:
			return nil, errors.Wrap(err, "failed to parse a faculty")
		}
	}

	return &catalog, nil
}

func (p *parser) parseFaculty(faculty *Faculty) error {
	for strings.TrimSpace(p.text()) == "" {
		if !p.scan() {
			return io.EOF
		}
	}

	switch p.text() {
	case sportsFacultySep:
		return p.parseSportsFaculty(faculty)
	case facultySep:
		// Ordinary faculty - keep going
	default:
		return p.errorf("Expected faculty separator, but got %q", p.text())
	}

	if err := p.expectLineAndAdvance(facultySep); err != nil {
		return errors.Wrap(err, "didn't find 1st faculty separator line in faculty")
	}

	{
		var err error
		if faculty.Name, err = p.parseFacultyName(); err != nil {
			return errors.Wrap(err, "failed to parse faculty name")
		}
		if faculty.Semester, err = p.parseFacultySemester(); err != nil {
			return errors.Wrap(err, "failed to parse faculty semester")
		}
	}

	if err := p.expectLineAndAdvance(facultySep); err != nil {
		return errors.Wrap(err, "didn't find 2nd faculty separator line in faculty")
	}

courses:
	for {
		course, err := p.parseCourse()
		switch err {
		case io.EOF:
			break courses
		case nil:
			faculty.Courses = append(faculty.Courses, *course)
			// Keep scanning
		default:
			p.errorf("failed to scan a course in faculty %s: %v", faculty.Name, err)
			p.warningf("skipping to next course")
			for p.text() != courseSep {
				p.scan()
			}
		}
	}

	return nil
}

const sportsFacultyName = "טרופס תועוצקמ"

var sportsFacultySemesterRegexp = regexp.MustCompile(`\| *([א-ת" ]+) +רטסמס *- *טרופס תועוצקמ *\|`)

func (p *parser) parseSportsFaculty(faculty *Faculty) error {
	p.infof("Started scanning sports faculty")

	if err := p.expectLineAndAdvance(sportsFacultySep); err != nil {
		return errors.Wrap(err, "didn't find 1nd faculty separate line in sports faculty")
	}

	m := sportsFacultySemesterRegexp.FindStringSubmatch(p.text())
	if m == nil {
		return p.errorf("Line %q doesn't match sports semester regex `%s`", p.text(), sportsFacultySemesterRegexp)
	}
	p.scan()
	faculty.Semester = Reverse(strings.TrimSpace(m[1]))
	faculty.Name = sportsFacultyName

	if err := p.expectLineAndAdvance(sportsFacultySep); err != nil {
		return errors.Wrap(err, "didn't find 2nd faculty separate line in sports faculty")
	}

	for {
		course, err := p.parseSportsCourse()
		if err != nil {
			return errors.Wrap(err, "failed to scan a sports course")
		}
		if course != nil {
			faculty.Courses = append(faculty.Courses, *course)
		} else {
			break
		}
	}

	return nil
}

// parseCourse will return a parsed course, or nil on end-of-faculty.
func (p *parser) parseCourse() (*Course, error) {
	*p.course = Course{}
	// First groupID is usually omitted in REPY.
	p.groupID = 10

	for p.text() == courseSep {
		if !p.scan() {
			return nil, p.errorf("Unexpected EOF while parsing course")
		}
	}

	if p.text() == "" {
		// End of faculty
		return nil, io.EOF
	}

	if err := p.parseIDAndName(); err != nil {
		return nil, errors.Wrap(err, "failed to parse ID and name in ordinary course")
	}

	if err := p.parseHoursAndPoints(); err != nil {
		p.warningf("Invalid hours and points line: %v", err)
		p.scan()
	}

	if err := p.expectLineAndAdvance(courseSep); err != nil {
		return nil, errors.Wrap(err, "didn't find expected course separator when parsing course")
	}

	if err := p.parseCourseHeadInfo(); err != nil {
		return nil, errors.Wrap(err, "failed to parse course head info")
	}

	// TODO(lutzky): There might be some comments about the course here

	if err := p.parseGroups(); err != nil {
		return nil, errors.Wrap(err, "failed to parse groups for course")
	}

	return p.course, nil
}

func (p *parser) parseSportsCourse() (*Course, error) {
	*p.course = Course{}

	p.groupID = 10

	for p.text() == sportsCourseSep {
		p.scan()
	}

	if p.text() == "" {
		// End of faculty
		return nil, nil
	}

	if err := p.parseIDAndName(); err != nil {
		return nil, errors.Wrap(err, "failed to parse ID and name in sports course")
	}

	if err := p.parseHoursAndPoints(); err != nil {
		p.warningf("Invalid hours and points line in sports course: %v", err)
		p.scan()
	}

	if err := p.expectLineAndAdvance(sportsCourseSep); err != nil {
		return nil, errors.Wrap(err, "didn't find expected course separator when parsing course")
	}

	// TODO(lutzky): Actually collect the group information

	p.warningf("Skipping sports course group information (not implemented) for %s", p.course.Name)

	for p.text() != sportsCourseSep {
		if !p.scan() {
			panic("End of file reached unexpectedly")
		}
	}

	return p.course, nil
}

func (p *parser) expectLineAndAdvance(s string) error {
	if p.text() != s {
		return p.errorfSkip(2, "Expected %q, got %q", s, p.text())
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

func (p *parser) timeOfDayFromStrings(hours, minutes string) MinutesSinceMidnight {
	h := p.parseUint(hours)
	m := p.parseUint(minutes)
	return MinutesSinceMidnight(h*60 + m)
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
	if len(p.course.Groups) == 0 {
		p.course.Groups = []Group{{}}
	}
	return &p.course.Groups[len(p.course.Groups)-1]
}

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

var lecturerRegexp = regexp.MustCompile(
	`\| *(.*) *: *(הצרמ|לגרתמ) *\|`)

func (p *parser) parseLecturerLine() bool {
	if m := lecturerRegexp.FindStringSubmatch(p.text()); len(m) > 0 {
		lecturer := hebrewFlip(collapseSpaces(m[1]))
		teachers := &p.lastGroup().Teachers
		*teachers = append(*teachers, lecturer)
		return true
	}
	return false
}

func findStringSubmatchMap(r *regexp.Regexp, s string) map[string]string {
	// It's very odd that we need to implement this function ourselves.
	// Lifted from here:
	// http://blog.kamilkisiel.net/blog/2012/07/05/using-the-go-regexp-package/

	result := map[string]string{}

	m := r.FindStringSubmatch(s)
	if m == nil {
		return result
	}

	for i, name := range r.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		result[name] = m[i]
	}

	return result
}

var eventRegexp = regexp.MustCompile(
	`\| *` +
		`(?P<location>.*) +` +
		`(?P<startHour>[0-9]{1,2})\.(?P<startMinute>[0-9]{2})- *` +
		`(?P<endHour>[0-9]{1,2})\.(?P<endMinute>[0-9]{2})'` +
		`(?P<weekday>[אבגדהוש]) ` +
		`(:(?P<groupType>[א-ת]+))?` +
		` +(?P<groupID>[0-9]+)? ` +
		`*\|`)

func (p *parser) parseEventLine() bool {
	if m := findStringSubmatchMap(eventRegexp, p.text()); len(m) > 0 {
		ev := Event{
			Day:            p.weekDayFromHebrewLetter(m["weekday"]),
			StartMinute:    p.timeOfDayFromStrings(m["startHour"], m["startMinute"]),
			EndStartMinute: p.timeOfDayFromStrings(m["endHour"], m["endMinute"]),
			Location:       p.parseLocation(m["location"]),
		}

		if m["groupType"] != "" {
			groupType, err := p.groupTypeFromString(m["groupType"])
			if err != nil {
				p.warningf("Failed to parse group type %q: %v", m["groupType"], err)
				return false
			}

			group := Group{
				Teachers: []string{}, // TODO(lutzky): Fill these in
				Events:   []Event{},
				Type:     groupType,
			}

			if m["groupID"] != "" {
				group.ID = p.parseUint(m["groupID"])
				p.groupID = group.ID + 1
			} else {
				group.ID = p.groupID
				p.groupID++
			}

			p.course.Groups = append(p.course.Groups, group)
		}

		group := &p.course.Groups[len(p.course.Groups)-1]
		group.Events = append(group.Events, ev)

		p.scan()
		return true
	}
	return false
}

func (p *parser) parseGroups() error {
	if p.text() != groupSep1 {
		p.warningf("Expected %q, got %q; skipping course", groupSep1, p.text())
		return nil
	}

	for {
		if p.text() == groupSep1 {
			p.scan()
			if err := p.expectLineAndAdvance(groupSep2); err != nil {
				return errors.Wrap(err, "didn't find 2nd expected group separator")
			}
			if p.groupID > 10 {
				p.groupID = (p.groupID/10)*10 + 10
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

// Reverse reverses a visual-Hebrew string into logical order.
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
