package hargen

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/pb33f/harific/motor/model"
)

// InjectionLocation defines where to inject a search term
type InjectionLocation int

const (
	RequestBody InjectionLocation = iota
	ResponseBody
	RequestHeader
	ResponseHeader
	QueryParam
	Cookie
	URL
)

// String returns the string representation of the injection location
func (il InjectionLocation) String() string {
	switch il {
	case RequestBody:
		return "request.body"
	case ResponseBody:
		return "response.body"
	case RequestHeader:
		return "request.header"
	case ResponseHeader:
		return "response.header"
	case QueryParam:
		return "query.param"
	case Cookie:
		return "cookie"
	case URL:
		return "url"
	default:
		return "unknown"
	}
}

// InjectedTerm represents a term that was injected and where
type InjectedTerm struct {
	Term       string            // the injected word/phrase
	Location   InjectionLocation // where it was injected
	EntryIndex int               // which har entry contains it
	FieldPath  string            // for bodies: json path like "user.name"
}

// GenerateOptions configures har generation
type GenerateOptions struct {
	EntryCount         int                   // number of entries to generate
	InjectTerms        []string              // terms to inject for testing
	InjectionLocations []InjectionLocation   // where to inject (if empty, use all)
	DictionaryPath     string                // path to word dictionary (default: /usr/share/dict/words)
	MaxJSONDepth       int                   // max nesting level (default: 3)
	MaxJSONNodes       int                   // max nodes per level (default: 10)
	Seed               int64                 // random seed for reproducibility (0 = use time)
	FatMode            bool                  // generate ~100KB per entry (huge JSON + base64 blobs)
}

// DefaultGenerateOptions provides sensible defaults
var DefaultGenerateOptions = GenerateOptions{
	EntryCount:     10,
	DictionaryPath: "/usr/share/dict/words",
	MaxJSONDepth:   3,
	MaxJSONNodes:   10,
	Seed:           0,
}

// GenerateResult contains the generated har and injection metadata
type GenerateResult struct {
	HARFilePath   string         // path to generated har file
	InjectedTerms []InjectedTerm // where each term was injected
	TotalEntries  int            // number of entries generated
}

// Generate creates a har file with injected search terms
func Generate(opts GenerateOptions) (*GenerateResult, error) {
	// apply defaults
	if opts.DictionaryPath == "" {
		opts.DictionaryPath = DefaultGenerateOptions.DictionaryPath
	}
	if opts.MaxJSONDepth == 0 {
		opts.MaxJSONDepth = DefaultGenerateOptions.MaxJSONDepth
	}
	if opts.MaxJSONNodes == 0 {
		opts.MaxJSONNodes = DefaultGenerateOptions.MaxJSONNodes
	}

	// generate in memory first
	har, injected, err := GenerateInMemory(opts)
	if err != nil {
		return nil, err
	}

	// create temp file
	tmpFile, err := os.CreateTemp("", "hargen-*.har")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// write har to file
	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(har); err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to write har: %w", err)
	}

	return &GenerateResult{
		HARFilePath:   tmpFile.Name(),
		InjectedTerms: injected,
		TotalEntries:  len(har.Log.Entries),
	}, nil
}

// GenerateInMemory creates a har structure without writing to disk
func GenerateInMemory(opts GenerateOptions) (*model.HAR, []InjectedTerm, error) {
	// apply defaults (honor zero entrycount for empty har testing)
	if opts.DictionaryPath == "" {
		opts.DictionaryPath = DefaultGenerateOptions.DictionaryPath
	}
	if opts.MaxJSONDepth == 0 {
		opts.MaxJSONDepth = DefaultGenerateOptions.MaxJSONDepth
	}
	if opts.MaxJSONNodes == 0 {
		opts.MaxJSONNodes = DefaultGenerateOptions.MaxJSONNodes
	}

	// create local rng (avoid mutating global rand)
	var rng *rand.Rand
	if opts.Seed != 0 {
		rng = rand.New(rand.NewSource(opts.Seed))
	} else {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	// load dictionary
	dict, err := LoadDictionary(opts.DictionaryPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load dictionary: %w", err)
	}

	// create generators with local rng
	jsonGen := NewJSONGenerator(dict, opts.MaxJSONDepth, opts.MaxJSONNodes, rng)
	jsonGen.SetFatMode(opts.FatMode)
	entryGen := NewEntryGenerator(dict, jsonGen, rng)
	entryGen.SetFatMode(opts.FatMode)

	// distribute injection terms across entries
	injectionPlan := createInjectionPlan(opts.InjectTerms, opts.EntryCount, opts.InjectionLocations, rng)

	// generate entries
	var entries []model.Entry
	var allInjected []InjectedTerm

	for i := 0; i < opts.EntryCount; i++ {
		termsForEntry := injectionPlan[i]
		entry, injected := entryGen.GenerateEntry(i, termsForEntry, opts.InjectionLocations)
		entries = append(entries, *entry)
		allInjected = append(allInjected, injected...)
	}

	har := &model.HAR{
		Log: model.Log{
			Version: "1.2",
			Creator: model.Creator{
				Name:    "hargen",
				Version: "1.0.0",
			},
			Entries: entries,
		},
	}

	return har, allInjected, nil
}

// createInjectionPlan distributes terms across entries
func createInjectionPlan(terms []string, entryCount int, locations []InjectionLocation, rng *rand.Rand) map[int][]injectionRequest {
	plan := make(map[int][]injectionRequest)

	if len(terms) == 0 || entryCount == 0 {
		return plan
	}

	// distribute terms randomly across entries
	for _, term := range terms {
		entryIndex := rng.Intn(entryCount)
		location := randomLocation(locations, rng)

		plan[entryIndex] = append(plan[entryIndex], injectionRequest{
			term:     term,
			location: location,
		})
	}

	return plan
}

// injectionRequest represents a single term injection request
type injectionRequest struct {
	term     string
	location InjectionLocation
}

// randomLocation selects a random location from the provided list
// if list is empty, selects from all locations
func randomLocation(locations []InjectionLocation, rng *rand.Rand) InjectionLocation {
	if len(locations) == 0 {
		return InjectionLocation(rng.Intn(7)) // 0-6 covers all locations
	}
	return locations[rng.Intn(len(locations))]
}

// GenerateToFile generates a har and writes it to a specific file path
func GenerateToFile(path string, opts GenerateOptions) ([]InjectedTerm, error) {
	har, injected, err := GenerateInMemory(opts)
	if err != nil {
		return nil, err
	}

	// ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// create file
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// write har
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(har); err != nil {
		return nil, fmt.Errorf("failed to write har: %w", err)
	}

	return injected, nil
}
