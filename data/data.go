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
	Function string `json:"function,omitempty"`
	// Date of the course, UTC.
	Date time.Time
	// Type of the course. "Colloque", "Lesson inaugurale", etc.
	LessonType string `json:"lesson_type"`
	// Title of the colloque / yearly lesson. "Inde-Chine : Universalités croisées".
	TypeTitle string `json:"type_title,omitempty"`
	// Video link if present.
	VideoLink string `json:"-"`
	// Audio link.
	AudioLink string `json:"audio_link"`
	// Title of the chaire. "Histoire intellectuelle de la Chine".
	Chaire string `json:"chaire"`
	// Language of the audio. ("fr", "en", etc.)
	Language string `json:"lang"`
	// Where this course was crawled from, "https://www.college-de-france.fr/site/anne-cheng/symposium-2017-06-23-16h15.htm".
	Source string `json:"source_url"`
	// DurationSec is how long the audio file is.
	DurationSec int `json:"duration"`
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
}

// Hints returns a list of sentences or words to help speech recognition.
func (c *Course) Hints() []string {
	s := []string{c.Title, c.Lecturer, c.Chaire}
	if c.TypeTitle != "" {
		s = append(s, c.TypeTitle)
	}
	return s
}
