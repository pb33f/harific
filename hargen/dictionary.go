package hargen

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

// fallback word list for when /usr/share/dict/words doesn't exist (windows, containers)
var fallbackWords = []string{
	"test", "search", "api", "user", "data", "request", "response",
	"header", "body", "method", "status", "error", "success", "server",
	"client", "service", "endpoint", "query", "param", "cookie",
	"auth", "token", "key", "value", "name", "type", "content",
	"message", "result", "code", "text", "json", "xml", "html",
	"get", "post", "put", "delete", "patch", "head", "options",
	"accept", "encoding", "language", "cache", "connection", "host",
	"agent", "referer", "origin", "location", "redirect", "proxy",
	"session", "id", "timestamp", "date", "time", "duration",
	"size", "length", "count", "total", "limit", "offset",
	"version", "format", "charset", "boundary", "transfer",
	"encoding", "compression", "gzip", "deflate", "chunked",
	"websocket", "upgrade", "protocol", "scheme", "port",
	"path", "fragment", "hash", "domain", "subdomain", "tld",
	"ipv4", "ipv6", "address", "network", "mask", "gateway",
	"dns", "tcp", "udp", "http", "https", "ftp", "ssh",
	"ssl", "tls", "certificate", "cipher", "algorithm", "hash",
}

// Dictionary holds a list of words for random selection
type Dictionary struct {
	words []string
}

// LoadDictionary loads words from a dictionary file
func LoadDictionary(path string) (*Dictionary, error) {
	file, err := os.Open(path)
	if err != nil {
		// fallback to built-in word list if file doesn't exist
		if os.IsNotExist(err) {
			return &Dictionary{words: fallbackWords}, nil
		}
		return nil, fmt.Errorf("failed to open dictionary: %w", err)
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())

		// filter to reasonable length (3-15 chars) and alpha only
		if len(word) >= 3 && len(word) <= 15 && isAlpha(word) {
			words = append(words, strings.ToLower(word))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read dictionary: %w", err)
	}

	if len(words) == 0 {
		return nil, fmt.Errorf("no valid words found in dictionary")
	}

	return &Dictionary{words: words}, nil
}

// isAlpha checks if a string contains only alphabetic characters
func isAlpha(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

// RandomWord returns a random word from the dictionary
func (d *Dictionary) RandomWord(rng *rand.Rand) string {
	if len(d.words) == 0 {
		return "word"
	}
	return d.words[rng.Intn(len(d.words))]
}

// RandomWords returns n random words from the dictionary
func (d *Dictionary) RandomWords(n int, rng *rand.Rand) []string {
	if n <= 0 {
		return nil
	}

	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = d.RandomWord(rng)
	}
	return result
}

// Size returns the number of words in the dictionary
func (d *Dictionary) Size() int {
	return len(d.words)
}
