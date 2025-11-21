package model

// Request contains the request description and content.
type Request struct {
	// Method of the HTTP request, in caps, GET/POST/etc
	Method string `json:"method"`

	// URL of the request (absolute), with fragments removed.
	URL string `json:"url"`

	// HTTPVersion of the request
	HTTPVersion string `json:"httpVersion"` // ex "HTTP/1.1"

	// Cookies sent with the request
	Cookies []Cookie `json:"cookies"`

	// Headers sent with the request
	Headers []NameValuePair `json:"headers"`

	// QueryParams parsed from the URL
	QueryParams []NameValuePair `json:"queryString"`

	// Body of the request (e.g. from a POST)
	Body BodyType `json:"postData,omitempty"`

	// HeadersSize of the request header in bytes.
	// NB counted from start of request to end of double CRLF before body.
	HeadersSize int `json:"headersSize"`

	// BodySize of the request body in bytes.
	BodySize int `json:"bodySize"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}

// BodyType contains information about the Body of a request
type BodyType struct {
	// MIMEType of the body content
	MIMEType string `json:"mimeType"`
	// List of (parsed URL-encoded) parameters, exclusive with Content
	Params []PostNameValuePair `json:"params,omitempty"`
	// Content of the post as plain text (exclusive with Params)
	Content string `json:"text,omitempty"`
}

// PostNameValuePair contains the description and content of a POSTed name and value pair.
// In particular this can include files.
type PostNameValuePair struct {
	// Name of the posted parameter
	Name string `json:"name"`
	// Value of the parameter or file contents
	Value string `json:"value,omitempty"`
	// Name of an uploaded file
	FileName string `json:"fileName,omitempty"`
	// ContentType of an uploaded file
	ContentType string `json:"contentType,omitempty"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}
