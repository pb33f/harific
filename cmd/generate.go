package cmd

import (
	"fmt"
	"strings"

	"github.com/pb33f/braid/hargen"
	"github.com/spf13/cobra"
)

var (
	genEntryCount     int
	genOutputFile     string
	genInjectTerms    []string
	genLocations      []string
	genSeed           int64
	genDictPath       string
	genMaxDepth       int
	genMaxNodes       int
	genShowInjections bool
	genFatMode        bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate HAR files with optional search term injection",
	Long: `Generate HAR (HTTP Archive) files of various sizes for testing.
Can inject specific search terms into known locations for testing search functionality.

Examples:
  harific generate -n 100 -o test.har
  harific generate -n 1000 -i apple,banana -l url,request.body
  harific generate --fat-mode -n 50 -o large.har
  harific generate --entries 10 --inject searchterm --show-injections`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().IntVarP(&genEntryCount, "entries", "n", 10, "Number of HAR entries to generate")
	generateCmd.Flags().StringVarP(&genOutputFile, "output", "o", "", "Output file path (default: hargen-{timestamp}.har)")
	generateCmd.Flags().StringSliceVarP(&genInjectTerms, "inject", "i", []string{}, "Terms to inject (comma-separated)")
	generateCmd.Flags().StringSliceVarP(&genLocations, "locations", "l", []string{}, "Injection locations: url,request.body,response.body,request.header,response.header,query.param,cookie (default: all)")
	generateCmd.Flags().Int64VarP(&genSeed, "seed", "s", 0, "Random seed for reproducibility (0 = use current time)")
	generateCmd.Flags().StringVarP(&genDictPath, "dict", "d", "/usr/share/dict/words", "Dictionary file path")
	generateCmd.Flags().IntVar(&genMaxDepth, "max-depth", 3, "Maximum JSON nesting depth")
	generateCmd.Flags().IntVar(&genMaxNodes, "max-nodes", 10, "Maximum JSON nodes per level")
	generateCmd.Flags().BoolVar(&genShowInjections, "show-injections", true, "Show injection details after generation")
	generateCmd.Flags().BoolVar(&genFatMode, "fat-mode", false, "Generate huge entries (~100KB each) with base64 blobs")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Parse injection locations
	var injectionLocs []hargen.InjectionLocation
	if len(genLocations) > 0 {
		for _, loc := range genLocations {
			switch strings.ToLower(loc) {
			case "url":
				injectionLocs = append(injectionLocs, hargen.URL)
			case "request.body", "requestbody":
				injectionLocs = append(injectionLocs, hargen.RequestBody)
			case "response.body", "responsebody":
				injectionLocs = append(injectionLocs, hargen.ResponseBody)
			case "request.header", "requestheader":
				injectionLocs = append(injectionLocs, hargen.RequestHeader)
			case "response.header", "responseheader":
				injectionLocs = append(injectionLocs, hargen.ResponseHeader)
			case "query.param", "queryparam":
				injectionLocs = append(injectionLocs, hargen.QueryParam)
			case "cookie":
				injectionLocs = append(injectionLocs, hargen.Cookie)
			default:
				return fmt.Errorf("unknown injection location: %s", loc)
			}
		}
	}

	// Build options
	opts := hargen.GenerateOptions{
		EntryCount:         genEntryCount,
		InjectTerms:        genInjectTerms,
		InjectionLocations: injectionLocs,
		DictionaryPath:     genDictPath,
		MaxJSONDepth:       genMaxDepth,
		MaxJSONNodes:       genMaxNodes,
		Seed:               genSeed,
		FatMode:            genFatMode,
	}

	fmt.Printf("Generating HAR file with %d entries", genEntryCount)
	if genFatMode {
		fmt.Printf(" (fat mode: ~100KB per entry)")
	}
	fmt.Println("...")
	if len(genInjectTerms) > 0 {
		fmt.Printf("Injecting terms: %v\n", genInjectTerms)
	}

	var result *hargen.GenerateResult
	var injected []hargen.InjectedTerm
	var err error

	if genOutputFile != "" {
		// Generate to specific file
		injected, err = hargen.GenerateToFile(genOutputFile, opts)
		if err != nil {
			return fmt.Errorf("failed to generate HAR: %w", err)
		}
		result = &hargen.GenerateResult{
			HARFilePath:   genOutputFile,
			InjectedTerms: injected,
			TotalEntries:  genEntryCount,
		}
	} else {
		// Generate to temp file
		result, err = hargen.Generate(opts)
		if err != nil {
			return fmt.Errorf("failed to generate HAR: %w", err)
		}
	}

	fmt.Printf("\n✓ Generated HAR file: %s\n", result.HARFilePath)
	fmt.Printf("  Total entries: %d\n", result.TotalEntries)

	if genShowInjections && len(result.InjectedTerms) > 0 {
		fmt.Printf("\nInjected terms:\n")
		for _, inj := range result.InjectedTerms {
			fmt.Printf("  • '%s' at entry %d in %s", inj.Term, inj.EntryIndex, inj.Location)
			if inj.FieldPath != "" {
				fmt.Printf(" (%s)", inj.FieldPath)
			}
			fmt.Println()
		}
	}

	return nil
}