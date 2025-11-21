package model

import "time"

// HAR represents the root of an HTTP Archive document.
//
// W3C Spec: https://w3c.github.io/web-performance/specs/HAR/Overview.html
type HAR struct {
	Log Log `json:"log"`
}

// NewHAR creates a new HTTP Archive document with the provided Creator Name.
// The recommended invocation is NewHAR(os.Args[0]).
func NewHAR(creatorName string) *HAR {
	v := time.Now().Format("20060102150405")

	return &HAR{
		Log: Log{
			Version: v,
			Creator: Creator{
				Name:    creatorName,
				Version: v,
			},
		},
	}
}

// Log represents a set of HTTP Request/Response Entries.
type Log struct {
	// Version of the log, defaults to the current time (formatted as "20060102150405")
	Version string `json:"version"`

	// Creator of this set of Log entries.
	Creator Creator `json:"creator"`

	// Browser information that produced this set of Log entries.
	Browser *Creator `json:"browser,omitempty"`

	// Pages contain information about request groupings, such as a page loaded by a web browser.
	Pages []Page `json:"pages,omitempty"`

	// Entries contains all of the Request and Response details that passed
	// through this Client.
	Entries []Entry `json:"entries"`

	// Comment can be added to the log to describe the particulars of this data.
	Comment string `json:"comment,omitempty"`
}
