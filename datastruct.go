package repy

// This file has exported data structure definitions for REPY. These also
// determine the JSON export format.

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

	// Semester is specified per-faculty in the REPY file. Presumably this is for
	// cases when not all faculties are up-to-date.
	Semester string `json:"semester"`
}

// WeeklyHours represents the amount of weekly hours, by type, that a course
// has.
type WeeklyHours struct {
	Lecture  uint `json:"lecture,omitempty"`
	Tutorial uint `json:"tutorial,omitempty"`
	Lab      uint `json:"lab,omitempty"`
	Project  uint `json:"project,omitempty"`
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

	// Sport groups indicate a sporting activity in the sports faculty
	Sport
)

var groupTypeName = []string{
	Lecture:  "lecture",
	Tutorial: "tutorial",
	Lab:      "lab",
	Sport:    "sport",
}

func (gt GroupType) String() string {
	return groupTypeName[gt]
}

// MarshalText implements encoding.TextMarshaler
func (gt GroupType) MarshalText() ([]byte, error) {
	return []byte(groupTypeName[gt]), nil
}

var dummyGroupTypeForStaticTypeChecks = GroupType(0)
var _ encoding.TextMarshaler = dummyGroupTypeForStaticTypeChecks
var _ encoding.TextUnmarshaler = &dummyGroupTypeForStaticTypeChecks

// UnmarshalText implements encoding.TextUnmarshaler
func (gt *GroupType) UnmarshalText(b []byte) error {
	s := string(b)

	for i, n := range groupTypeName {
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
	ID          uint      `json:"id"`
	Teachers    []string  `json:"teachers"`
	Events      []Event   `json:"events"`
	Type        GroupType `json:"type"`
	Description string    `json:"description"`
}

// Event represents a singular weekly event within a course.
type Event struct {
	Day         time.Weekday         `json:"day"`
	Location    string               `json:"location"`
	StartMinute MinutesSinceMidnight `json:"startMinute"`
	EndMinute   MinutesSinceMidnight `json:"endMinute"`
}

// MinutesSinceMidnight is a way of representing a scheduled time-of-day,
// literally represented as "minutes since midnight".
type MinutesSinceMidnight uint
