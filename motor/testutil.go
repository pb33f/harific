package motor

import (
	"fmt"
	"os"

	"github.com/pb33f/harific/hargen"
)

// generateTestHAR generates a HAR file for testing and returns the path and cleanup function
func generateTestHAR(entries int, seed int64) (string, func(), error) {
	opts := hargen.GenerateOptions{
		EntryCount:     entries,
		Seed:           seed,
		InjectTerms:    []string{},
		MaxJSONDepth:   3,
		MaxJSONNodes:   10,
		FatMode:        false,
		DictionaryPath: "/usr/share/dict/words",
	}

	result, err := hargen.Generate(opts)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate test HAR: %w", err)
	}

	cleanup := func() {
		os.Remove(result.HARFilePath)
	}

	return result.HARFilePath, cleanup, nil
}

// generateSmallHAR generates a small HAR file (~5MB, 67 entries) for testing
func generateSmallHAR() (string, func(), error) {
	return generateTestHAR(67, 42)
}

// generateMediumHAR generates a medium HAR file (~50MB, 700 entries) for testing
func generateMediumHAR() (string, func(), error) {
	return generateTestHAR(700, 42)
}

// generateTinyHAR generates a tiny HAR file (10 entries) for quick tests
func generateTinyHAR() (string, func(), error) {
	return generateTestHAR(10, 42)
}
