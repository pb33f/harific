package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/pb33f/harhar"
)

// pre-computed styles to avoid allocation in hot path
var (
	keyStyleBase = lipgloss.NewStyle().
		Foreground(RGBGrey).
		Align(lipgloss.Right)

	sectionHeaderStyleBase = lipgloss.NewStyle().
		Bold(true).
		Foreground(RGBPink)

	emptyValueText = lipgloss.NewStyle().Faint(true).Render("(empty)")
)

// KeyValuePair represents a single key-value pair
type KeyValuePair struct {
	Key   string
	Value string
}

// Section represents a grouped section of key-value pairs
type Section struct {
	Title string
	Pairs []KeyValuePair
}

// RenderOptions configures key-value rendering
type RenderOptions struct {
	Width    int  // total available width
	Truncate bool // whether to truncate long values
	KeyWidth int  // key column width (0 = auto-calculate)
}

// renderSections renders multiple sections as formatted key-value output
func renderSections(sections []Section, opts RenderOptions) string {
	if len(sections) == 0 {
		return ""
	}

	// calculate column widths
	keyWidth := opts.KeyWidth
	if keyWidth == 0 {
		keyWidth = opts.Width * 3 / 10 // 30% for keys
		if keyWidth > 25 {
			keyWidth = 25 // cap at 25 chars
		}
		if keyWidth < 15 {
			keyWidth = 15 // minimum 15 chars
		}
	}
	valueWidth := opts.Width - keyWidth - 3 // -3 for spacing

	var output strings.Builder

	for i, section := range sections {
		// section header
		if section.Title != "" {
			output.WriteString(renderSectionHeader(section.Title, opts.Width))
			output.WriteString("\n")
		}

		// render pairs
		for _, pair := range section.Pairs {
			row := renderKeyValueRow(pair, keyWidth, valueWidth, opts.Truncate)
			output.WriteString(row)
			output.WriteString("\n")
		}

		// add spacing between sections (except after last)
		if i < len(sections)-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}

// renderSectionHeader renders a section title
func renderSectionHeader(title string, width int) string {
	return sectionHeaderStyleBase.Width(width).Render(title)
}

// renderKeyValueRow renders a single key-value pair
func renderKeyValueRow(pair KeyValuePair, keyWidth, valueWidth int, truncate bool) string {
	keyStyle := keyStyleBase.Width(keyWidth)

	value := pair.Value
	if value == "" {
		value = emptyValueText
	} else if truncate && len(value) > valueWidth {
		value = value[:valueWidth-3] + "..."
	}

	return keyStyle.Render(pair.Key) + "  " + value
}

// buildRequestSections converts a HAR request to sections
func buildRequestSections(req *harhar.Request) []Section {
	sections := make([]Section, 1, 5) // pre-allocate for typical case
	sections[0] = Section{
		Title: "Request",
		Pairs: []KeyValuePair{
			{"Method", req.Method},
			{"URL", req.URL},
			{"HTTP Version", req.HTTPVersion},
		},
	}

	if len(req.Headers) > 0 {
		sections = append(sections, Section{
			Title: "Headers",
			Pairs: nameValuePairsToPairs(req.Headers),
		})
	}

	if len(req.QueryParams) > 0 {
		sections = append(sections, Section{
			Title: "Query Parameters",
			Pairs: nameValuePairsToPairs(req.QueryParams),
		})
	}

	if len(req.Cookies) > 0 {
		sections = append(sections, Section{
			Title: "Cookies",
			Pairs: cookiesToPairs(req.Cookies),
		})
	}

	if req.Body.Content != "" {
		sections = append(sections, Section{
			Title: "Body",
			Pairs: []KeyValuePair{
				{"Content-Type", req.Body.MIMEType},
				{"Size", fmt.Sprintf("%d bytes", req.BodySize)},
				{"Content", req.Body.Content},
			},
		})
	}

	return sections
}

// buildResponseSections converts a HAR response to sections
func buildResponseSections(resp *harhar.Response, timings *harhar.Timings) []Section {
	sections := make([]Section, 1, 4) // pre-allocate for typical case
	sections[0] = Section{
		Title: "Response",
		Pairs: []KeyValuePair{
			{"Status", fmt.Sprintf("%d %s", resp.StatusCode, resp.StatusText)},
			{"HTTP Version", resp.HTTPVersion},
		},
	}

	if len(resp.Headers) > 0 {
		sections = append(sections, Section{
			Title: "Headers",
			Pairs: nameValuePairsToPairs(resp.Headers),
		})
	}

	if resp.Body.Content != "" {
		sections = append(sections, Section{
			Title: "Body",
			Pairs: []KeyValuePair{
				{"Content-Type", resp.Body.MIMEType},
				{"Size", fmt.Sprintf("%d bytes", resp.Body.Size)},
				{"Content", resp.Body.Content},
			},
		})
	}

	// add timings if available
	if timings != nil && (timings.DNS >= 0 || timings.Connect >= 0) {
		pairs := make([]KeyValuePair, 0, 6) // max 6 timing fields

		if timings.DNS >= 0 {
			pairs = append(pairs, KeyValuePair{"DNS", fmt.Sprintf("%.2fms", timings.DNS)})
		}
		if timings.Connect >= 0 {
			pairs = append(pairs, KeyValuePair{"Connect", fmt.Sprintf("%.2fms", timings.Connect)})
		}
		if timings.Send >= 0 {
			pairs = append(pairs, KeyValuePair{"Send", fmt.Sprintf("%.2fms", timings.Send)})
		}
		if timings.Wait >= 0 {
			pairs = append(pairs, KeyValuePair{"Wait", fmt.Sprintf("%.2fms", timings.Wait)})
		}
		if timings.Receive >= 0 {
			pairs = append(pairs, KeyValuePair{"Receive", fmt.Sprintf("%.2fms", timings.Receive)})
		}
		if timings.SSL >= 0 {
			pairs = append(pairs, KeyValuePair{"SSL", fmt.Sprintf("%.2fms", timings.SSL)})
		}

		if len(pairs) > 0 {
			sections = append(sections, Section{
				Title: "Timings",
				Pairs: pairs,
			})
		}
	}

	return sections
}

// nameValuePairsToPairs converts HAR name-value pairs to KeyValuePairs
func nameValuePairsToPairs(nvps []harhar.NameValuePair) []KeyValuePair {
	pairs := make([]KeyValuePair, len(nvps))
	for i, nvp := range nvps {
		pairs[i] = KeyValuePair{nvp.Name, nvp.Value}
	}
	return pairs
}

func cookiesToPairs(cookies []harhar.Cookie) []KeyValuePair {
	pairs := make([]KeyValuePair, len(cookies))
	for i, c := range cookies {
		pairs[i] = KeyValuePair{c.Name, c.Value}
	}
	return pairs
}

// renderSectionsWithSearch renders sections with JSON search support
func renderSectionsWithSearch(sections []Section, opts RenderOptions, searchState *ViewportSearchState) string {
	if len(sections) == 0 {
		return ""
	}

	// Only process if search state is provided and active
	if searchState == nil || !searchState.active {
		return renderSections(sections, opts)
	}

	// Check if we have a body section with JSON content
	for i, section := range sections {
		if section.Title == "Body" {
			for j, pair := range section.Pairs {
				if pair.Key == "Content" {
					// Check if content is JSON
					if !isValidJSON(pair.Value) {
						continue
					}

					// Initialize the search renderer if not already done
					if !searchState.HasJSONContent() {
						searchState.SetContent(pair.Value, opts.Width)
					}

					// If we have a renderer, use it to render the content
					if searchState.HasJSONContent() {
						sections[i].Pairs[j].Value = searchState.GetRenderedContent()
					}
				}
			}
		}
	}

	// Render normally
	return renderSections(sections, opts)
}
