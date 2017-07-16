// datastruct.go has exported data structure definitions for REPY. These also
// determine the JSON export format.
package repy

import "time"

// Catalog represents all of the information in a REPY file
type Catalog []Faculty

// Faculty represents a set of courses offered by a faculty.
type Faculty struct {
	Name    string
	Courses []Course
}

// WeeklyHours represents the amount of weekly hours, by type, that a course
// has.
type WeeklyHours struct {
	Lecture  uint
	Tutorial uint
	Lab      uint
	Project  uint
}

// Course represents information about a technion course.
type Course struct {
	ID               uint
	Name             string
	AcademicPoints   float32 // ...even though 2*points is always a uint :/
	LecturerInCharge string
	WeeklyHours      WeeklyHours
	TestDates        []Date
	Groups           []Group
}

// Date is a timezone-free representation of a date
type Date struct {
	Year, Month, Day uint
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

// TODO(lutzky): GroupType should be JSON-Marshaled as "lecture", "tutorial",
// etc.

// Group represents a course's registration group (קבוצת רישום) and the events
// it entails.
type Group struct {
	ID       uint
	Teachers []string
	Events   []Event
	Type     GroupType
}

// Event represents a singular weekly event within a course.
type Event struct {
	Day                time.Weekday
	Location           string
	StartHour, EndHour TimeOfDay
}

// TimeOfDay is represented as "minutes since midnight".
type TimeOfDay uint

// TODO(lutzky): TimeOfDay should be JSON-Marshaled as HH:MM
