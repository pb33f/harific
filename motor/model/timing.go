package model

// Timings contains various timings for network latency.
type Timings struct {
	// Send is the Time required to send this request to the server.
	Send float64 `json:"send"`
	// Wait is the Time spent waiting on a response from the server.
	Wait float64 `json:"wait"`
	// Receive is the Time spent reading the entire response from the server.
	Receive float64 `json:"receive"`

	// Blocked is the Time spent in a queue waiting for a network connection
	Blocked float64 `json:"blocked,omitempty"`
	// DNS is the domain name resolution time - The time required to resolve a host name
	DNS float64 `json:"dns,omitempty"`
	// Connect is the Time required to create TCP connection.
	Connect float64 `json:"connect,omitempty"`

	// SSL is the Time required to negotiate the SSL/TLS connection.
	// Note: if defined this time is included in Connect.
	SSL float64 `json:"ssl,omitempty"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}
