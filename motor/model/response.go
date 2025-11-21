package model

// Response contains the response description and content.
type Response struct {
	// StatusCode indicates the response status
	StatusCode int `json:"status"` // 200

	// StatusText describes the response status
	StatusText string `json:"statusText"` // "OK"

	// HTTPVersion of the HTTP response
	HTTPVersion string `json:"httpVersion"` // ex "HTTP/1.1"

	// RedirectURL from the location header
	RedirectURL string `json:"redirectURL"`

	// Cookies sent with the response
	Cookies []Cookie `json:"cookies"`

	// Headers sent with the response
	// NB Headers may include values added by the browser but not included in server's response.
	Headers []NameValuePair `json:"headers"`

	// Body describes the response body content.
	Body BodyResponseType `json:"content"`

	// HeadersSize of the request header in bytes.
	// NB counted from start of request to end of double CRLF before body.
	// NB only includes the size of headers sent by the server, not those added by a browser.
	HeadersSize int `json:"headersSize"`

	// BodySize of the response body in bytes (as sent)
	BodySize int `json:"bodySize"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}

// BodyResponseType contains various information about the response body.
type BodyResponseType struct {
	// Size of response content in bytes (decompressed).
	Size int `json:"size"`
	// Compression is the number of bytes saved by compression
	Compression int `json:"compression,omitempty"`
	// MIMEType of the body content
	MIMEType string `json:"mimeType"`
	// Content is the text content of the response body.
	Content string `json:"text,omitempty"`
	// Encoding used by the response.
	Encoding string `json:"encoding,omitempty"`
	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}
