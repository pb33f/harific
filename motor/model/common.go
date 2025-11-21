package model

// NameValuePair is a name and value, paired.
type NameValuePair struct {
	// Name of the parameter
	Name string `json:"name"`
	// Value of the parameter
	Value string `json:"value"`

	// Comment can be added by the user
	Comment string `json:"comment,omitempty"`
}

// Cookie describes the cookie information for requests and responses.
type Cookie struct {
	// Name of the cookie.
	Name string `json:"name"`
	// Value stored in the cookie.
	Value string `json:"value"`
	// Path that this cookie applied to.
	Path string `json:"path,omitempty"`
	// Domain is the hostname the cookie applies to.
	Domain string `json:"domain,omitempty"`
	// Expires describes the cookie expiration time (ISO 8601).
	Expires string `json:"expires,omitempty"`
	// Secure is true if the cookie was transferred over SSL.
	Secure bool `json:"secure,omitempty"`
	// HTTPOnly flag status of the cookie.
	HTTPOnly bool `json:"httpOnly,omitempty"`
}
