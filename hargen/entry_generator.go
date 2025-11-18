package hargen

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/pb33f/harhar"
)

// EntryGenerator creates HAR entries with optional term injection
type EntryGenerator struct {
	dict    *Dictionary
	jsonGen *JSONGenerator
}

// NewEntryGenerator creates a new entry generator
func NewEntryGenerator(dict *Dictionary, jsonGen *JSONGenerator) *EntryGenerator {
	return &EntryGenerator{
		dict:    dict,
		jsonGen: jsonGen,
	}
}

// GenerateEntry creates a single HAR entry with optional term injection
func (eg *EntryGenerator) GenerateEntry(index int, injectionRequests []injectionRequest, allowedLocations []InjectionLocation) (*harhar.Entry, []InjectedTerm) {
	entry := &harhar.Entry{
		Start:      time.Now().Add(-time.Duration(index) * time.Second).Format(time.RFC3339),
		Time:       float64(rand.Intn(1000)) + rand.Float64(),
		Request:    eg.generateRequest(),
		Response:   eg.generateResponse(),
		ServerIP:   eg.generateIP(),
		Connection: fmt.Sprintf("%d", rand.Intn(65535)),
	}

	var injected []InjectedTerm

	// inject terms into specified locations
	for _, req := range injectionRequests {
		result := eg.injectIntoEntry(entry, req.term, req.location, index)
		injected = append(injected, result)
	}

	return entry, injected
}

func (eg *EntryGenerator) generateRequest() harhar.Request {
	return harhar.Request{
		Method:      eg.randomMethod(),
		URL:         eg.generateURL(),
		HTTPVersion: "HTTP/1.1",
		Headers:     eg.generateHeaders(rand.Intn(8) + 3),
		QueryParams: eg.generateQueryParams(rand.Intn(5)),
		Cookies:     eg.generateCookies(rand.Intn(3)),
		Body:        eg.generateRequestBody(),
		HeadersSize: rand.Intn(500) + 200,
		BodySize:    rand.Intn(2000) + 100,
	}
}

func (eg *EntryGenerator) generateResponse() harhar.Response {
	status := eg.randomStatus()
	return harhar.Response{
		StatusCode:  status,
		StatusText:  eg.statusText(status),
		HTTPVersion: "HTTP/1.1",
		Headers:     eg.generateHeaders(rand.Intn(10) + 5),
		Cookies:     eg.generateCookies(rand.Intn(2)),
		Body:        eg.generateResponseBody(),
		HeadersSize: rand.Intn(700) + 300,
		BodySize:    rand.Intn(5000) + 500,
	}
}

func (eg *EntryGenerator) randomMethod() string {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	return methods[rand.Intn(len(methods))]
}

func (eg *EntryGenerator) randomStatus() int {
	statuses := []int{200, 201, 204, 301, 302, 400, 401, 403, 404, 500, 502, 503}
	return statuses[rand.Intn(len(statuses))]
}

func (eg *EntryGenerator) statusText(code int) string {
	texts := map[int]string{
		200: "OK",
		201: "Created",
		204: "No Content",
		301: "Moved Permanently",
		302: "Found",
		400: "Bad Request",
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		500: "Internal Server Error",
		502: "Bad Gateway",
		503: "Service Unavailable",
	}
	if text, ok := texts[code]; ok {
		return text
	}
	return "Unknown"
}

func (eg *EntryGenerator) generateURL() string {
	domains := []string{"api.example.com", "service.test.org", "app.company.io"}
	domain := domains[rand.Intn(len(domains))]

	paths := make([]string, rand.Intn(3)+1)
	for i := range paths {
		paths[i] = eg.dict.RandomWord()
	}

	url := "https://" + domain
	for _, p := range paths {
		url += "/" + p
	}

	return url
}

func (eg *EntryGenerator) generateIP() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))
}

func (eg *EntryGenerator) generateHeaders(count int) []harhar.NameValuePair {
	if count == 0 {
		count = 3
	}

	commonHeaders := []string{
		"Content-Type", "User-Agent", "Accept", "Accept-Encoding",
		"Cache-Control", "Connection", "Host", "Authorization",
		"Content-Length", "Cookie", "Accept-Language",
	}

	headers := make([]harhar.NameValuePair, 0, count)
	usedHeaders := make(map[string]bool)

	for i := 0; i < count && len(commonHeaders) > len(usedHeaders); i++ {
		header := commonHeaders[rand.Intn(len(commonHeaders))]
		if usedHeaders[header] {
			continue
		}
		usedHeaders[header] = true

		headers = append(headers, harhar.NameValuePair{
			Name:  header,
			Value: eg.headerValue(header),
		})
	}

	return headers
}

func (eg *EntryGenerator) headerValue(name string) string {
	switch name {
	case "Content-Type":
		types := []string{"application/json", "text/html", "application/xml", "text/plain"}
		return types[rand.Intn(len(types))]
	case "User-Agent":
		return "Mozilla/5.0 (compatible; TestBot/1.0)"
	case "Accept":
		return "*/*"
	case "Accept-Encoding":
		return "gzip, deflate"
	case "Connection":
		return "keep-alive"
	case "Cache-Control":
		return "no-cache"
	default:
		return eg.dict.RandomWord()
	}
}

func (eg *EntryGenerator) generateQueryParams(count int) []harhar.NameValuePair {
	params := make([]harhar.NameValuePair, count)
	for i := 0; i < count; i++ {
		params[i] = harhar.NameValuePair{
			Name:  eg.dict.RandomWord(),
			Value: eg.dict.RandomWord(),
		}
	}
	return params
}

func (eg *EntryGenerator) generateCookies(count int) []harhar.Cookie {
	cookies := make([]harhar.Cookie, count)
	for i := 0; i < count; i++ {
		cookies[i] = harhar.Cookie{
			Name:  eg.dict.RandomWord(),
			Value: eg.dict.RandomWord(),
		}
	}
	return cookies
}

func (eg *EntryGenerator) generateRequestBody() harhar.BodyType {
	obj := eg.jsonGen.GenerateObject(0)
	content, _ := json.Marshal(obj)
	return harhar.BodyType{
		MIMEType: "application/json",
		Content:  string(content),
	}
}

func (eg *EntryGenerator) generateResponseBody() harhar.BodyResponseType {
	obj := eg.jsonGen.GenerateRealisticObject("api_response")
	content, _ := json.Marshal(obj)
	return harhar.BodyResponseType{
		Size:     len(content),
		MIMEType: "application/json",
		Content:  string(content),
	}
}

func (eg *EntryGenerator) injectIntoEntry(entry *harhar.Entry, term string, location InjectionLocation, entryIndex int) InjectedTerm {
	result := InjectedTerm{
		Term:       term,
		Location:   location,
		EntryIndex: entryIndex,
	}

	switch location {
	case RequestBody:
		obj, path := eg.jsonGen.InjectTermIntoNewObject(term)
		content, _ := json.Marshal(obj)
		entry.Request.Body.Content = string(content)
		result.FieldPath = path

	case ResponseBody:
		obj, path := eg.jsonGen.InjectTermIntoNewObject(term)
		content, _ := json.Marshal(obj)
		entry.Response.Body.Content = string(content)
		entry.Response.Body.Size = len(content)
		result.FieldPath = path

	case RequestHeader:
		headerName := eg.dict.RandomWord()
		entry.Request.Headers = append(entry.Request.Headers, harhar.NameValuePair{
			Name:  headerName,
			Value: term,
		})
		result.FieldPath = headerName

	case ResponseHeader:
		headerName := eg.dict.RandomWord()
		entry.Response.Headers = append(entry.Response.Headers, harhar.NameValuePair{
			Name:  headerName,
			Value: term,
		})
		result.FieldPath = headerName

	case QueryParam:
		paramName := eg.dict.RandomWord()
		entry.Request.QueryParams = append(entry.Request.QueryParams, harhar.NameValuePair{
			Name:  paramName,
			Value: term,
		})
		result.FieldPath = paramName

	case Cookie:
		cookieName := eg.dict.RandomWord()
		entry.Request.Cookies = append(entry.Request.Cookies, harhar.Cookie{
			Name:  cookieName,
			Value: term,
		})
		result.FieldPath = cookieName

	case URL:
		// inject term as a path segment
		entry.Request.URL = "https://api.example.com/" + term + "/" + eg.dict.RandomWord()
		result.FieldPath = "path"
	}

	return result
}
