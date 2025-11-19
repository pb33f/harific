// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

// TODO: Extract to shared pb33f/syntax package for reuse across vacuum and braid

package tui

import (
	"strings"
)

// HighlightYAMLKeyValue handles key-value pair highlighting for YAML
func HighlightYAMLKeyValue(line string) (string, bool) {

	// SUPER SIMPLE: Find colon, color everything before it (key) + colon blue
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return "", false
	}

	// Find leading whitespace
	leadingWhitespace := ""
	contentStart := 0
	for i, r := range line {
		if r != ' ' && r != '\t' {
			leadingWhitespace = line[:i]
			contentStart = i
			break
		}
	}

	// Find trailing whitespace
	trimmedLine := strings.TrimRight(line, " \t\r\n")
	trailingWhitespace := line[len(trimmedLine):]

	// Extract just the content (no leading or trailing whitespace)
	content := trimmedLine[contentStart:]

	// Style only the key portion (before colon)
	var styledContent string
	contentColonIdx := strings.Index(content, ":")
	if contentColonIdx != -1 {
		// Split into key part and value part
		keyPart := content[:contentColonIdx+1]   // include the colon
		valuePart := content[contentColonIdx+1:] // everything after colon

		// Style only the key part, leave value unstyled
		styledContent = SyntaxKeyStyle.Render(keyPart) + valuePart
	} else {
		// Fallback: no colon found in content, style entire content
		styledContent = SyntaxKeyStyle.Render(content)
	}

	return leadingWhitespace + styledContent + trailingWhitespace, true
}

// HighlightJSONLine handles JSON syntax highlighting
func HighlightJSONLine(line string) string {
	// Find leading whitespace
	leadingWhitespace := ""
	contentStart := 0
	for i, r := range line {
		if r != ' ' && r != '\t' {
			leadingWhitespace = line[:i]
			contentStart = i
			break
		}
	}

	// Find trailing whitespace
	trimmedLine := strings.TrimRight(line, " \t\r\n")
	trailingWhitespace := line[len(trimmedLine):]

	// Extract just the content (no leading or trailing whitespace)
	content := trimmedLine[contentStart:]

	// Look for JSON key-value pattern: "key":
	if idx := strings.Index(content, "\":"); idx > 0 {
		// Find where the key starts (opening quote)
		keyStart := strings.LastIndex(content[:idx], "\"")
		if keyStart >= 0 {
			// Split into: before key + key part + value part
			beforeKey := content[:keyStart]
			keyPart := content[keyStart : idx+2] // includes quotes and colon
			valuePart := content[idx+2:]         // everything after colon

			// Style key blue, leave value unstyled, handle brackets in beforeKey and valuePart
			beforeKeyStyled := styleBrackets(beforeKey)
			keyStyled := SyntaxKeyStyle.Render(keyPart)
			valuePartStyled := styleBrackets(valuePart)

			styledContent := beforeKeyStyled + keyStyled + valuePartStyled
			return leadingWhitespace + styledContent + trailingWhitespace
		}
	}

	// No key-value pattern, just handle brackets
	styledContent := styleBrackets(content)
	return leadingWhitespace + styledContent + trailingWhitespace
}

// styleBrackets styles { } characters in pink and [ ] characters in yellow while leaving everything else unstyled
func styleBrackets(text string) string {
	if text == "" {
		return text
	}

	var result strings.Builder
	for _, r := range text {
		if r == '{' || r == '}' {
			// Style curly brackets pink (using SyntaxDashStyle which is pink)
			result.WriteString(SyntaxDashStyle.Render(string(r)))
		} else if r == '[' || r == ']' {
			// Style square brackets yellow (using SyntaxNumberStyle which is yellow)
			result.WriteString(SyntaxNumberStyle.Render(string(r)))
		} else {
			// Leave everything else unstyled
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ApplySyntaxHighlightingToLine applies syntax highlighting to a single line
func ApplySyntaxHighlightingToLine(line string, isYAML bool) string {
	if line == "" {
		return line
	}

	if isYAML {
		// Try simple key-value highlighting
		if result, handled := HighlightYAMLKeyValue(line); handled {
			return result
		}
		// No match, return as-is
		return line
	} else {
		return HighlightJSONLine(line)
	}
}
