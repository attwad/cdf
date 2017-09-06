package data

import "time"

// Course represents a lesson, colloque, symposium, etc.
// It represents a superset of what gets indexed by the search engine (anything that has a json field mapped).
type Course struct {
	// Title of the course, "What was at Stake in the India-China Opium Trade?".
	Title string `json:"title"`
	// Lecturer, "John Doe".
	Lecturer string `json:"lecturer"`
	// Function of the lecturer, "EHESS, Paris".
	Function string `json:"function"`
	// Date of the course, UTC.
	Date time.Time `json:"-"`
	// Type of the course. "Colloque", "Lesson inaugurale", etc.
	LessonType string `json:"lesson_type,omitempty"`
	// Title of the colloque / yearly lesson. "Inde-Chine : Universalités croisées".
	TypeTitle string `json:"type_title,omitempty"`
	// Video link if present.
	VideoLink string `json:"-"`
	// Audio link.
	AudioLink string `json:"-"`
	// Title of the chaire. "Histoire intellectuelle de la Chine".
	Chaire string `json:"chaire"`
	// Language of the audio. ("fr", "en", etc.)
	Language string `json:"lang,omitempty"`
	// Where this course was crawled from, "https://www.college-de-france.fr/site/anne-cheng/symposium-2017-06-23-16h15.htm".
	Source string `json:"source_url"`
	// DurationSec is how long the audio file is.
	DurationSec int `json:"-"`
	// When this course was scraped.
	Scraped time.Time `json:"-"`
}

// Entry is what gets stored in Datastore, it contains a course and special storage only fields.
type Entry struct {
	Course
	// Whether the course has been converted yet.
	Converted bool
	// The hash to "randomly" pick an entry to convert.
	Hash []byte
	// Whether it is scheduled for conversion.
	Scheduled bool
	// When it was scheduled for conversion if applicable.
	ScheduledTime time.Time
	// Transcript is the full text of this lesson.
	Transcript string `datastore:",noindex" json:"-"`
}

// ExternalCourse is what gets sent to clients, it contains formatted durations, dates etc.
type ExternalCourse struct {
	Course
	FormattedDate     string `json:"date"`
	FormattedDuration string `json:"duration"`
}

// Hints returns a list of sentences or words to help speech recognition.
func (c *Course) Hints() []string {
	// Context phrases must not be longer than 100 characters.
	s := make([]string, 0)
	if len(c.Title) < 100 {
		s = append(s, c.Title)
	}
	if len(c.Lecturer) < 100 {
		s = append(s, c.Lecturer)
	}
	if len(c.Chaire) < 100 {
		s = append(s, c.Chaire)
	}
	if c.TypeTitle != "" && len(c.TypeTitle) < 100 {
		s = append(s, c.TypeTitle)
	}
	return s
}
