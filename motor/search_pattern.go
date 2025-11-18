package motor

import (
	"fmt"
	"regexp"
	"strings"
)

// SearchMode defines the type of search to perform
type SearchMode int

const (
	PlainText SearchMode = iota
	Regex
)

// compiledPattern holds a compiled search pattern
type compiledPattern struct {
	mode      SearchMode
	plainText string
	regex     *regexp.Regexp
}

// compilePattern compiles a search pattern based on search mode
func compilePattern(pattern string, opts SearchOptions) (compiledPattern, error) {
	cp := compiledPattern{
		mode: opts.Mode,
	}

	if opts.Mode == Regex {
		// compile regex pattern
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return cp, fmt.Errorf("invalid regex pattern: %w", err)
		}
		cp.regex = regex
	} else {
		// plain text pattern
		cp.plainText = pattern
	}

	return cp, nil
}

// matches checks if haystack matches the compiled pattern
func matches(haystack string, pattern compiledPattern) bool {
	if pattern.mode == Regex {
		return pattern.regex.MatchString(haystack)
	}

	// plain text: use strings.contains (faster than regex)
	return strings.Contains(haystack, pattern.plainText)
}
