package model

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ConvertRequestIntoHttpRequest converts a HAR Request into a standard Go http.Request.
func ConvertRequestIntoHttpRequest(req Request) (*http.Request, error) {
	r := bytes.NewReader([]byte(req.Body.Content))
	// Prefer the recorded query to preserve repeated parameters and escaping.
	httpRequest, err := http.NewRequest(req.Method, req.URL, nil)
	if err != nil {
		return nil, err
	}
	// Some HAR producers (including hargen) store the query only in QueryParams.
	// Preserve that representation without overwriting an explicit empty query.
	if httpRequest.URL.RawQuery == "" && !httpRequest.URL.ForceQuery {
		params := make([]string, 0, len(req.QueryParams))
		for _, param := range req.QueryParams {
			params = append(params, url.QueryEscape(param.Name)+"="+url.QueryEscape(param.Value))
		}
		httpRequest.URL.RawQuery = strings.Join(params, "&")
	}

	values := make(url.Values)
	for _, param := range req.Body.Params {
		decoded, _ := url.QueryUnescape(param.Value)
		values.Add(param.Name, decoded)
	}
	if req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodPatch {
		httpRequest.PostForm = values
		httpRequest.Form = values
	}

	for _, header := range req.Headers {
		if !strings.HasPrefix(header.Name, ":") {
			httpRequest.Header.Set(header.Name, header.Value)
		}
	}
	for _, cookie := range req.Cookies {
		httpRequest.AddCookie(&http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
		})
	}
	if req.Body.MIMEType != "" {
		httpRequest.Header.Set("Content-Type", req.Body.MIMEType)
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	httpRequest.Body = io.NopCloser(bytes.NewReader(body))
	httpRequest.ContentLength = int64(len(body))
	return httpRequest, nil
}
