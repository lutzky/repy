// datastruct.go has exported data structure definitions for REPY. These also
// determine the JSON export format.
package repy

import (
	"encoding"
	"time"

	"github.com/pkg/errors"
)

// Catalog represents all of the information in a REPY file
type Catalog []Faculty

// Faculty represents a set of courses offered by a faculty.
type Faculty struct {
	Name    string   `json:"name"`
	Courses []Course `json:"courses"`
}

// WeeklyHours represents the amount of weekly hours, by type, that a course
// has.
type WeeklyHours struct {
	Lecture  uint `json:"lecture"`
	Tutorial uint `json:"tutorial"`
	Lab      uint `json:"lab"`
	Project  uint `json:"project"`
}

// Course represents information about a technion course.
type Course struct {
	ID               uint        `json:"id"`
	Name             string      `json:"name"`
	AcademicPoints   float32     `json:"academicPoints"`
	LecturerInCharge string      `json:"lecturerInCharge"`
	WeeklyHours      WeeklyHours `json:"weeklyHours"`
	TestDates        []Date      `json:"testDates"`
	Groups           []Group     `json:"groups"`
}

// Date is a timezone-free representation of a date
type Date struct {
	Year  uint `json:"year"`
	Month uint `json:"month"`
	Day   uint `json:"day"`
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

var GroupTypeName = []string{
	Lecture:  "lecture",
	Tutorial: "tutorial",
	Lab:      "lab",
}

func (gt GroupType) String() string {
	return GroupTypeName[gt]
}

func (gt GroupType) MarshalText() ([]byte, error) {
	return []byte(GroupTypeName[gt]), nil
}

func typeCheckTextMarshaler() {
	var gt GroupType = GroupType(0)
	var _ encoding.TextMarshaler = gt
	var _ encoding.TextUnmarshaler = &gt
	panic("This should never be called")
}

func (gt *GroupType) UnmarshalText(b []byte) error {
	s := string(b)

	for i, n := range GroupTypeName {
		if n == s {
			*gt = GroupType(i)
			return nil
		}
	}
	return errors.Errorf("Unknown GroupType %q", s)
}

// Group represents a course's registration group (קבוצת רישום) and the events
// it entails.
type Group struct {
	ID       uint      `json:"id"`
	Teachers []string  `json:"teachers"`
	Events   []Event   `json:"events"`
	Type     GroupType `json:"type"`
}

// Event represents a singular weekly event within a course.
type Event struct {
	Day            time.Weekday         `json:"day"`
	Location       string               `json:"location"`
	StartMinute    MinutesSinceMidnight `json:"startMinute"`
	EndStartMinute MinutesSinceMidnight `json:"endMinute"`
}

// MinutesSinceMidnight is a way of representing a scheduled time-of-day,
// literally represented as "minutes since midnight".
type MinutesSinceMidnight uint
