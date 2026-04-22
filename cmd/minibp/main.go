// Package main implements minibp, a build system that generates Ninja build files from Blueprint definitions.
// It parses .bp files, resolves dependencies, handles architecture variants, and outputs build.ninja.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	applib "minibp/lib/utils"
	buildlib "minibp/lib/build"
	"minibp/lib/namespace"
	"minibp/lib/parser"
	"minibp/lib/props"
)

// openInputFile is a dependency injection for opening input files.
// It defaults to os.Open but can be replaced for testing.
var (
	openInputFile      = func(path string) (io.ReadCloser, error) { return os.Open(path) }
	createOutputFile   = func(path string) (io.WriteCloser, error) { return os.Create(path) }
	parseBlueprintFile = parser.ParseFile
)

// main is the entry point for the minibp command-line tool.
// It parses command-line flags, loads Blueprint definitions, and generates a Ninja build file.
// On success, it exits with code 0; on failure, it exits with code 1 and prints an error to stderr.
func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run is the main logic function that processes command-line arguments and generates the build file.

// It handles flag parsing, Blueprint file loading, dependency resolution, variant merging, and Ninja file generation.

func run(args []string, stdout, stderr io.Writer) error {
	cfg, err := applib.ParseRunConfig(args, stderr)
	if err != nil {
		return err
	}
	if cfg.ShowVersion {
		fmt.Fprintf(stdout, "minibp version %s\n", applib.GetVersion())
		return nil
	}

	eval := applib.NewEvaluatorFromConfig(cfg)

	// Parse all Blueprint files into definitions
	allDefs, err := parseDefinitionsFromFiles(cfg.Inputs)
	if err != nil {
		return err
	}

	// Process variable assignments in definitions
	eval.ProcessAssignmentsFromDefs(allDefs)

	buildOpts := cfg.BuildOptions()
	modules, err := buildlib.CollectModulesWithNames(allDefs, eval, buildOpts, func(m *parser.Module, name string) string {
		return props.GetStringPropEval(m, name, eval)
	})
	if err != nil {
		return err
	}

	// Build namespace map for soong_namespace resolution
	namespaces := namespace.BuildMap(modules, func(m *parser.Module, name string) string {
		return props.GetStringPropEval(m, name, eval)
	})

	graph := buildlib.BuildGraph(modules, namespaces, eval)
	gen := buildlib.NewGenerator(graph, modules, buildOpts)

	// Generate and write the Ninja build file
	if err := generateNinjaFile(cfg.OutFile, gen); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Generated %s with %d modules\n", cfg.OutFile, len(modules))
	return nil
}

func parseDefinitionsFromFiles(files []string) ([]parser.Definition, error) {
	var allDefs []parser.Definition
	var parseErrors []string

	for _, file := range files {
		f, err := openInputFile(file)
		if err != nil {
			return nil, fmt.Errorf("error opening %s: %w", file, err)
		}

		parsedFile, parseErr := parseBlueprintFile(f, file)
		closeErr := f.Close()
		if parseErr != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("parse error in %s: %v", file, parseErr))
			continue
		}
		if closeErr != nil {
			return nil, fmt.Errorf("error closing %s: %w", file, closeErr)
		}
		allDefs = append(allDefs, parsedFile.Defs...)
	}

	if len(parseErrors) > 0 {
		return nil, fmt.Errorf("parsing failed: %s", strings.Join(parseErrors, "; "))
	}
	return allDefs, nil
}

func generateNinjaFile(path string, gen interface{ Generate(io.Writer) error }) error {
	out, err := createOutputFile(path)
	if err != nil {
		return fmt.Errorf("error creating output: %w", err)
	}

	genErr := gen.Generate(out)
	closeErr := out.Close()
	if genErr != nil {
		closeErr = os.Remove(path)
		if closeErr != nil {
			return fmt.Errorf("error generating ninja: %w; error removing incomplete file: %v", genErr, closeErr)
		}
		return fmt.Errorf("error generating ninja: %w", genErr)
	}
	if closeErr != nil {
		return fmt.Errorf("error closing output: %w", closeErr)
	}
	return nil
}
