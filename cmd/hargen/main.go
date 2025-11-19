package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pb33f/braid/hargen"
	"github.com/spf13/cobra"
)

var (
	entryCount     int
	outputFile     string
	injectTerms    []string
	locations      []string
	seed           int64
	dictPath       string
	maxDepth       int
	maxNodes       int
	showInjections bool
	fatMode        bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "hargen",
		Short: "Generate HAR files with optional search term injection",
		Long: `hargen is a tool for generating HAR (HTTP Archive) files of various sizes.
It can inject specific search terms into known locations for testing search functionality.`,
		RunE: runGenerate,
	}

	rootCmd.Flags().IntVarP(&entryCount, "entries", "n", 10, "Number of HAR entries to generate")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: hargen-{timestamp}.har)")
	rootCmd.Flags().StringSliceVarP(&injectTerms, "inject", "i", []string{}, "Terms to inject (comma-separated)")
	rootCmd.Flags().StringSliceVarP(&locations, "locations", "l", []string{}, "Injection locations: url,request.body,response.body,request.header,response.header,query.param,cookie (default: all)")
	rootCmd.Flags().Int64VarP(&seed, "seed", "s", 0, "Random seed for reproducibility (0 = use current time)")
	rootCmd.Flags().StringVarP(&dictPath, "dict", "d", "/usr/share/dict/words", "Dictionary file path")
	rootCmd.Flags().IntVar(&maxDepth, "max-depth", 3, "Maximum JSON nesting depth")
	rootCmd.Flags().IntVar(&maxNodes, "max-nodes", 10, "Maximum JSON nodes per level")
	rootCmd.Flags().BoolVar(&showInjections, "show-injections", true, "Show injection details after generation")
	rootCmd.Flags().BoolVar(&fatMode, "fat-mode", false, "Generate huge entries (~100KB each) with base64 blobs")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Parse injection locations
	var injectionLocs []hargen.InjectionLocation
	if len(locations) > 0 {
		for _, loc := range locations {
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
		EntryCount:         entryCount,
		InjectTerms:        injectTerms,
		InjectionLocations: injectionLocs,
		DictionaryPath:     dictPath,
		MaxJSONDepth:       maxDepth,
		MaxJSONNodes:       maxNodes,
		Seed:               seed,
		FatMode:            fatMode,
	}

	fmt.Printf("Generating HAR file with %d entries", entryCount)
	if fatMode {
		fmt.Printf(" (fat mode: ~100KB per entry)")
	}
	fmt.Println("...")
	if len(injectTerms) > 0 {
		fmt.Printf("Injecting terms: %v\n", injectTerms)
	}

	var result *hargen.GenerateResult
	var injected []hargen.InjectedTerm
	var err error

	if outputFile != "" {
		// Generate to specific file
		injected, err = hargen.GenerateToFile(outputFile, opts)
		if err != nil {
			return fmt.Errorf("failed to generate HAR: %w", err)
		}
		result = &hargen.GenerateResult{
			HARFilePath:   outputFile,
			InjectedTerms: injected,
			TotalEntries:  entryCount,
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

	if showInjections && len(result.InjectedTerms) > 0 {
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
