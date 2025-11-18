package motor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompilePattern_PlainText(t *testing.T) {
	opts := SearchOptions{Mode: PlainText}

	pattern, err := compilePattern("testword", opts)
	require.NoError(t, err)
	assert.Equal(t, PlainText, pattern.mode)
	assert.Equal(t, "testword", pattern.plainText)
	assert.Nil(t, pattern.regex)
}

func TestCompilePattern_Regex(t *testing.T) {
	opts := SearchOptions{Mode: Regex}

	pattern, err := compilePattern("test.*word", opts)
	require.NoError(t, err)
	assert.Equal(t, Regex, pattern.mode)
	assert.NotNil(t, pattern.regex)
}

func TestCompilePattern_InvalidRegex(t *testing.T) {
	opts := SearchOptions{Mode: Regex}

	_, err := compilePattern("[invalid(", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern")
}

func TestMatches_PlainText_CaseSensitive(t *testing.T) {
	opts := SearchOptions{Mode: PlainText}
	pattern, _ := compilePattern("test", opts)

	assert.True(t, matches("this is a test string", pattern))
	assert.True(t, matches("test", pattern))
	assert.True(t, matches("testing", pattern))
	assert.False(t, matches("TEST", pattern))
	assert.False(t, matches("no match here", pattern))
}

func TestMatches_PlainText_Partial(t *testing.T) {
	opts := SearchOptions{Mode: PlainText}
	pattern, _ := compilePattern("api", opts)

	assert.True(t, matches("https://api.example.com", pattern))
	assert.True(t, matches("apikey", pattern))
	assert.True(t, matches("my_api_token", pattern))
}

func TestMatches_Regex_Simple(t *testing.T) {
	opts := SearchOptions{Mode: Regex}
	pattern, _ := compilePattern("test", opts)

	assert.True(t, matches("this is a test", pattern))
	assert.True(t, matches("testing", pattern))
}

func TestMatches_Regex_Complex(t *testing.T) {
	opts := SearchOptions{Mode: Regex}

	tests := []struct {
		name     string
		pattern  string
		haystack string
		expected bool
	}{
		{"digits", `\d+`, "user123", true},
		{"no digits", `\d+`, "nodigits", false},
		{"email", `\w+@\w+\.\w+`, "user@example.com", true},
		{"not email", `\w+@\w+\.\w+`, "notanemail", false},
		{"api pattern", `api\d{3}`, "api123", true},
		{"api pattern no match", `api\d{3}`, "api12", false},
		{"start anchor", `^test`, "test string", true},
		{"start anchor no match", `^test`, "the test", false},
		{"end anchor", `test$`, "this is a test", true},
		{"end anchor no match", `test$`, "test string", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, err := compilePattern(tt.pattern, opts)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, matches(tt.haystack, pattern))
		})
	}
}

func TestMatches_EmptyString(t *testing.T) {
	opts := SearchOptions{Mode: PlainText}
	pattern, _ := compilePattern("test", opts)

	assert.False(t, matches("", pattern))
}

func TestMatches_EmptyPattern(t *testing.T) {
	opts := SearchOptions{Mode: PlainText}
	pattern, _ := compilePattern("", opts)

	// empty pattern matches everything
	assert.True(t, matches("anything", pattern))
	assert.True(t, matches("", pattern))
}
