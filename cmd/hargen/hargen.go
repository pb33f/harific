package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/pb33f/harhar"
)

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
)

var (
	methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	statuses = []int{200, 201, 204, 301, 302, 400, 401, 403, 404, 500, 502, 503}
	mimeTypes = []string{
		"application/json",
		"text/html",
		"text/plain",
		"application/xml",
		"image/png",
		"image/jpeg",
		"application/javascript",
		"text/css",
	}
	domains = []string{
		"api.example.com",
		"cdn.example.com",
		"auth.example.com",
		"data.example.com",
		"static.example.com",
	}
)

func main() {
	var (
		size     = flag.String("size", "5MB", "target file size (e.g., 700MB, 1GB, 2GB)")
		output   = flag.String("output", "", "output file path (default: test-{size}.har)")
		seed     = flag.Int64("seed", time.Now().UnixNano(), "random seed for reproducibility")
	)
	flag.Parse()

	// parse size
	targetSize, err := parseSize(*size)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid size: %v\n", err)
		os.Exit(1)
	}

	// determine output path
	outputPath := *output
	if outputPath == "" {
		outputPath = fmt.Sprintf("test-%s.har", *size)
	}

	fmt.Printf("generating har file: %s (target size: %d bytes)\n", outputPath, targetSize)

	// initialize random
	rand.Seed(*seed)

	// generate har file
	if err := generateHAR(outputPath, targetSize); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate har: %v\n", err)
		os.Exit(1)
	}

	// check actual size
	stat, err := os.Stat(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to stat file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("generated %s (actual size: %d bytes, %.2f MB)\n",
		outputPath, stat.Size(), float64(stat.Size())/float64(MB))
}

func parseSize(s string) (int64, error) {
	// Parse size with units (e.g., "700MB", "1GB", "2GB")
	var value int64
	var unit string

	// Try to parse number and unit
	n, err := fmt.Sscanf(s, "%d%s", &value, &unit)
	if err != nil || n != 2 {
		return 0, fmt.Errorf("invalid size format: %s (use format like 700MB, 1GB, 2GB)", s)
	}

	// Convert to bytes based on unit
	switch unit {
	case "KB":
		return value * KB, nil
	case "MB":
		return value * MB, nil
	case "GB":
		return value * GB, nil
	default:
		return 0, fmt.Errorf("unsupported unit: %s (use KB, MB, or GB)", unit)
	}
}

func generateHAR(outputPath string, targetSize int64) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// estimate entry size (average ~100KB per entry with body)
	estimatedEntrySize := int64(100 * KB)
	estimatedEntries := int(targetSize / estimatedEntrySize)
	if estimatedEntries < 1 {
		estimatedEntries = 1
	}

	fmt.Printf("generating approximately %d entries...\n", estimatedEntries)

	// create encoder
	encoder := json.NewEncoder(file)

	// write har structure manually for streaming
	if _, err := file.WriteString(`{"log":{"version":"1.2",`); err != nil {
		return err
	}

	// write creator
	creator := harhar.Creator{
		Name:    "hargen",
		Version: "1.0.0",
	}
	if _, err := file.WriteString(`"creator":`); err != nil {
		return err
	}
	if err := encoder.Encode(creator); err != nil {
		return err
	}

	// write entries array start
	if _, err := file.WriteString(`,"entries":[`); err != nil {
		return err
	}

	// generate entries until we reach target size
	startTime := time.Now()
	currentSize := int64(0)
	entryCount := 0

	for currentSize < targetSize {
		if entryCount > 0 {
			if _, err := file.WriteString(","); err != nil {
				return err
			}
		}

		entry := generateEntry(startTime.Add(time.Duration(entryCount) * time.Second))

		// encode entry
		if err := encoder.Encode(entry); err != nil {
			return err
		}

		entryCount++

		// update size estimate
		stat, err := file.Stat()
		if err != nil {
			return err
		}
		currentSize = stat.Size()

		if entryCount%100 == 0 {
			fmt.Printf("generated %d entries (%.2f MB / %.2f MB)\n",
				entryCount,
				float64(currentSize)/float64(MB),
				float64(targetSize)/float64(MB))
		}
	}

	// close entries array and log object
	if _, err := file.WriteString(`]}}`); err != nil {
		return err
	}

	fmt.Printf("generated %d entries total\n", entryCount)

	return nil
}

func generateEntry(timestamp time.Time) harhar.Entry {
	method := methods[rand.Intn(len(methods))]
	statusCode := statuses[rand.Intn(len(statuses))]
	mimeType := mimeTypes[rand.Intn(len(mimeTypes))]
	domain := domains[rand.Intn(len(domains))]

	// generate random url path
	pathSegments := rand.Intn(4) + 1
	path := "/api"
	for i := 0; i < pathSegments; i++ {
		path += fmt.Sprintf("/%s", randomString(8))
	}
	url := fmt.Sprintf("https://%s%s", domain, path)

	// generate random body (10-100KB)
	bodySize := rand.Intn(90*KB) + 10*KB
	body := generateRandomBody(bodySize)

	entry := harhar.Entry{
		Start: timestamp.Format(time.RFC3339),
		Time:  float64(rand.Intn(5000) + 100), // 100-5100ms
		Request: harhar.Request{
			Method:      method,
			URL:         url,
			HTTPVersion: "HTTP/1.1",
			HeadersSize: rand.Intn(500) + 200,
			BodySize:    len(body),
			Headers: []harhar.NameValuePair{
				{Name: "Host", Value: domain},
				{Name: "User-Agent", Value: "hargen/1.0"},
				{Name: "Content-Type", Value: "application/json"},
			},
		},
		Response: harhar.Response{
			StatusCode:  statusCode,
			StatusText:  getStatusText(statusCode),
			HTTPVersion: "HTTP/1.1",
			HeadersSize: rand.Intn(500) + 200,
			BodySize:    len(body),
			Headers: []harhar.NameValuePair{
				{Name: "Content-Type", Value: mimeType},
				{Name: "Content-Length", Value: fmt.Sprintf("%d", len(body))},
			},
			Body: harhar.BodyResponseType{
				Size:     len(body),
				MIMEType: mimeType,
				Content:  body,
			},
		},
		Cache: harhar.CacheState{},
		Timings: harhar.Timings{
			Send:    float64(rand.Intn(50)),
			Wait:    float64(rand.Intn(1000) + 100),
			Receive: float64(rand.Intn(500) + 50),
		},
		ServerIP:   fmt.Sprintf("192.168.%d.%d", rand.Intn(255), rand.Intn(255)),
		Connection: fmt.Sprintf("conn-%d", rand.Intn(10000)),
	}

	return entry
}

func generateRandomBody(size int) string {
	// generate random bytes
	randomBytes := make([]byte, size)
	rand.Read(randomBytes)

	// encode as base64 to make it valid string content
	return base64.StdEncoding.EncodeToString(randomBytes)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func getStatusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	default:
		return "Unknown"
	}
}
