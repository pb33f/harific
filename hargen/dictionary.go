package hargen

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

// Dictionary holds a list of words for random selection
type Dictionary struct {
	words []string
}

// LoadDictionary loads words from a dictionary file
func LoadDictionary(path string) (*Dictionary, error) {
	file, err := os.Open(path)
	if err != nil {
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
func (d *Dictionary) RandomWord() string {
	if len(d.words) == 0 {
		return "word"
	}
	return d.words[rand.Intn(len(d.words))]
}

// RandomWords returns n random words from the dictionary
func (d *Dictionary) RandomWords(n int) []string {
	if n <= 0 {
		return nil
	}

	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = d.RandomWord()
	}
	return result
}

// Size returns the number of words in the dictionary
func (d *Dictionary) Size() int {
	return len(d.words)
}
