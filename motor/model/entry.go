package model

// Entry describes one request-response pair and associated information.
type Entry struct {
	// PageRef references the parent page (if supported), Page.ID.
	PageRef string `json:"pageref,omitempty"`

	// Start of the request (ISO 8601)
	Start string `json:"startedDateTime"`

	// Total time in milliseconds, Time=SUM(Timings.*)
	Time float64 `json:"time"`

	// Request details
	Request Request `json:"request"`

	// Response details
	Response Response `json:"response"`

	// Cache contains info about how the request was/is now cached.
	Cache CacheState `json:"cache"`

	// Timings contains detail info about the request/response round trip.
	Timings Timings `json:"timings"`

	// ServerIP contains the connected server address.
	ServerIP string `json:"serverIPAddress,omitempty"`

	// Connection contains the connection info (e.g. a TCP/IP Port/ID)
	Connection string `json:"connection,omitempty"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}

// CacheState represents the cache status before and after a request.
type CacheState struct {
	// Before contains the cache status before the request
	Before *CacheInfo `json:"beforeRequest,omitempty"`

	// After contains the cache status after the request
	After *CacheInfo `json:"afterRequest,omitempty"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}

// CacheInfo describes cache properties of known content
type CacheInfo struct {
	// Expiration time of the cached content (ISO 8601)
	Expires string `json:"expires,omitempty"`

	// LastAccess time of the cached content (ISO 8601)
	LastAccess string `json:"lastAccess"`

	// ETag referencing the cached content
	ETag string `json:"etag"`

	// HitCount is the number of the times the cached content has been opened.
	HitCount int `json:"hitCount"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}
