package repy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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

func parseCourse(x string) (*Course, error) {
	b := bytes.NewBufferString(x)
	return parseCourseFromReader(b)
}

func hebrewFlip(s string) string {
	runes := []rune(strings.TrimSpace(s))
	for i, j := 0, len(runes)-1; i < len(runes)/2; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

const courseSep = "+------------------------------------------+"

func parseCourseFromReader(r io.Reader) (*Course, error) {
	var err error
	c := Course{}
	s := bufio.NewScanner(r)

	s.Scan()
	if s.Text() != courseSep {
		// TODO(lutzky): Line numbers?
		return nil, fmt.Errorf("FILE:LINE: Expected course separator, got %q", s.Text())
	}

	s.Scan()
	re := regexp.MustCompile(`\| *(.*) +([0-9]{5,6}) \|`)
	if m := re.FindStringSubmatch(s.Text()); m == nil {
		return nil, fmt.Errorf("FILE:LINE: Line %q doesn't match %q", s.Text(), re)
	} else {
		c.name = hebrewFlip(m[1])
		var id uint64
		id, err = strconv.ParseUint(m[2], 10, 32)
		if err != nil {
			panic(fmt.Sprintf("Couldn't ParseUint(%q, 10, 32): %v", m[2], err))
		}
		c.id = uint(id)
	}

	/*
		  TODO(lutzky): error handling
			if err := scanner.Err(); err != nil {
				return nil, err
			}
	*/

	return &c, nil
}
