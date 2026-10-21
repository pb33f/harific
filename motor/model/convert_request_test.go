package model

import (
	"encoding/json"
	"net/url"
	"reflect"
	"testing"
)

func TestConvertRequestIntoHttpRequestPreservesHARQuery(t *testing.T) {
	tests := []struct {
		name    string
		request string
		query   url.Values
	}{
		{
			name:    "query represented in both HAR fields",
			request: `{"method":"GET","url":"https://example.com/?page=1&limit=10","queryString":[{"name":"page","value":"1"},{"name":"limit","value":"10"}]}`,
			query:   url.Values{"page": {"1"}, "limit": {"10"}},
		},
		{
			name:    "repeated names and identical values",
			request: `{"method":"GET","url":"https://example.com/?tag=a&tag=b&tag=a","queryString":[{"name":"tag","value":"a"},{"name":"tag","value":"b"},{"name":"tag","value":"a"}]}`,
			query:   url.Values{"tag": {"a", "b", "a"}},
		},
		{
			name:    "escaping and empty values",
			request: `{"method":"GET","url":"https://example.com/a%2fb?x%26y=a%2Bb%20c%26%3D%23&space=a+b&literal=%252F&empty=&flag&=value","queryString":[{"name":"x&y","value":"a+b c&=#"},{"name":"space","value":"a b"},{"name":"literal","value":"%2F"},{"name":"empty","value":""},{"name":"flag","value":""},{"name":"","value":"value"}]}`,
			query:   url.Values{"x&y": {"a+b c&=#"}, "space": {"a b"}, "literal": {"%2F"}, "empty": {""}, "flag": {""}, "": {"value"}},
		},
		{
			name:    "URL only query",
			request: `{"method":"GET","url":"https://example.com/?tag=a&tag=b&empty="}`,
			query:   url.Values{"tag": {"a", "b"}, "empty": {""}},
		},
		{
			name:    "conflicting metadata does not rewrite URL",
			request: `{"method":"GET","url":"https://example.com/?page=1","queryString":[{"name":"page","value":"2"},{"name":"extra","value":"3"}]}`,
			query:   url.Values{"page": {"1"}},
		},
		{
			name:    "metadata does not invent a query",
			request: `{"method":"GET","url":"https://example.com/","queryString":[{"name":"page","value":"1"}]}`,
			query:   url.Values{},
		},
		{
			name:    "explicit empty query",
			request: `{"method":"GET","url":"https://example.com/?","queryString":[]}`,
			query:   url.Values{},
		},
		{
			name:    "no query",
			request: `{"method":"GET","url":"https://example.com/","queryString":[]}`,
			query:   url.Values{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req Request
			if err := json.Unmarshal([]byte(tt.request), &req); err != nil {
				t.Fatal(err)
			}
			got, err := ConvertRequestIntoHttpRequest(req)
			if err != nil {
				t.Fatal(err)
			}
			defer got.Body.Close()
			if got.URL.String() != req.URL {
				t.Errorf("URL = %q, want recorded URL %q", got.URL.String(), req.URL)
			}
			if query := got.URL.Query(); !reflect.DeepEqual(query, tt.query) {
				t.Errorf("query = %#v, want %#v", query, tt.query)
			}
		})
	}
}
