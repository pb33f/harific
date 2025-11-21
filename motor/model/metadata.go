package model

// Creator describes the source of the logged requests/responses.
type Creator struct {
	// Name of the HTTP creator source.
	Name string `json:"name"`

	// Version of the HTTP creator source.
	Version string `json:"version"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}

// Page represents a group of requests (e.g. an HTML document with multiple resources)
type Page struct {
	// Start of the page load (ISO 8601)
	Start string `json:"startedDateTime"`

	// ID used to reference this page grouping (Entry.PageRef)
	ID string `json:"id"`

	// Title of the page
	Title string `json:"title"`

	// PageTimings contains detailing timing info about the page load
	PageTimings PageTiming `json:"pageTimings"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}

// PageTiming contains DOM-related page timing information.
type PageTiming struct {
	// OnContentLoad is milliseconds since Start for page content to be loaded.
	OnContentLoad float64 `json:"onContentLoad,omitempty"`

	// OnLoad is milliseconds since Start for OnLoad event to be fired.
	OnLoad float64 `json:"onLoad,omitempty"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}
