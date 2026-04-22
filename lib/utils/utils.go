package utils

import (
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	buildlib "minibp/lib/build"
	"minibp/lib/parser"
	"minibp/lib/version"
)

type RunConfig struct {
	OutFile     string
	All         bool
	CC          string
	CXX         string
	AR          string
	Arch        string
	Multilib    []string
	Host        bool
	TargetOS    string
	Variant     string
	Product     string
	LTO         string
	Sysroot     string
	Ccache      string
	ShowVersion bool
	Inputs      []string
	SrcDir      string
}

func ParseRunConfig(args []string, stderr io.Writer) (RunConfig, error) {
	cfg := RunConfig{}
	fs := flag.NewFlagSet("minibp", flag.ContinueOnError)

	outFile := fs.String("o", "build.ninja", "output ninja file")
	all := fs.Bool("a", false, "parse all .bp files in directory")
	ccFlag := fs.String("cc", "", "C compiler (default: gcc)")
	cxxFlag := fs.String("cxx", "", "C++ compiler (default: g++)")
	arFlag := fs.String("ar", "", "archiver (default: ar)")
	archFlag := fs.String("arch", "", "target architecture (arm, arm64, x86, x86_64)")
	multilibFlag := fs.String("multilib", "", "comma-separated target architectures for multi-arch build (e.g. arm64,x86_64)")
	hostFlag := fs.Bool("host", false, "build for host (overrides arch)")
	osFlag := fs.String("os", "", "target OS (linux, darwin, windows)")
	variantFlag := fs.String("variant", "", "comma-separated variant selectors (e.g. image=recovery,link=shared)")
	productFlag := fs.String("product", "", "comma-separated product variables (e.g. debuggable=true,board=soc_a)")
	ltoFlag := fs.String("lto", "", "default LTO mode: full, thin, or none")
	sysrootFlag := fs.String("sysroot", "", "sysroot path for cross-compilation")
	ccacheFlag := fs.String("ccache", "", "ccache path (empty: auto-detect, 'no': disable)")
	versionFlag := fs.Bool("v", false, "show version information")

	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}

	cfg = RunConfig{
		OutFile:     *outFile,
		All:         *all,
		CC:          *ccFlag,
		CXX:         *cxxFlag,
		AR:          *arFlag,
		Arch:        *archFlag,
		Multilib:    splitCSV(*multilibFlag),
		Host:        *hostFlag,
		TargetOS:    *osFlag,
		Variant:     *variantFlag,
		Product:     *productFlag,
		LTO:         *ltoFlag,
		Sysroot:     *sysrootFlag,
		Ccache:      *ccacheFlag,
		ShowVersion: *versionFlag,
		Inputs:      fs.Args(),
	}

	if cfg.ShowVersion {
		return cfg, nil
	}
	if len(cfg.Inputs) < 1 && !cfg.All {
		fmt.Fprintln(stderr, "Usage: minibp [-o output] [-a] [-cc CC] [-cxx CXX] [-ar AR] [-arch ARCH] [-host] [-os OS] <file.bp | directory>")
		return cfg, fmt.Errorf("missing input path")
	}

	cfg.SrcDir = determineSourceDir(cfg.All, cfg.Inputs)
	files, err := collectBlueprintFiles(cfg.All, cfg.SrcDir, cfg.Inputs)
	if err != nil {
		return cfg, err
	}
	cfg.Inputs = files
	return cfg, nil
}

func NewEvaluatorFromConfig(cfg RunConfig) *parser.Evaluator {
	eval := parser.NewEvaluator()
	eval.SetConfig("arch", cfg.Arch)
	eval.SetConfig("host", fmt.Sprintf("%v", cfg.Host))
	if cfg.TargetOS != "" {
		eval.SetConfig("os", cfg.TargetOS)
	} else {
		eval.SetConfig("os", "linux")
	}
	eval.SetConfig("target", cfg.Arch)
	setKeyValueConfigs(eval, "variant.", cfg.Variant)
	setKeyValueConfigs(eval, "product.", cfg.Product)
	return eval
}

func (cfg RunConfig) BuildOptions() buildlib.Options {
	return buildlib.Options{
		Arch:     cfg.Arch,
		Host:     cfg.Host,
		SrcDir:   cfg.SrcDir,
		OutFile:  cfg.OutFile,
		Inputs:   append([]string(nil), cfg.Inputs...),
		Multilib: append([]string(nil), cfg.Multilib...),
		CC:       cfg.CC,
		CXX:      cfg.CXX,
		AR:       cfg.AR,
		LTO:      cfg.LTO,
		Sysroot:  cfg.Sysroot,
		Ccache:   cfg.Ccache,
	}
}

func GetVersion() string {
	v := version.Get()
	gitCommit := v.GitCommit
	if gitCommit == "unknown" {
		if commit, err := getGitCommit(); err == nil {
			gitCommit = commit
		}
	}
	buildDate := v.BuildDate
	if buildDate == "unknown" {
		buildDate = "2026-04-21"
	}
	return fmt.Sprintf("%s (git: %s, built: %s, go: %s)", v.MinibpVer, gitCommit, buildDate, v.GoVersion)
}

func getGitCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func determineSourceDir(all bool, inputs []string) string {
	if all && len(inputs) > 0 {
		return inputs[0]
	}
	if len(inputs) > 0 {
		return filepath.Dir(inputs[0])
	}
	return "."
}

func collectBlueprintFiles(all bool, srcDir string, inputs []string) ([]string, error) {
	if !all {
		return inputs, nil
	}
	bpFiles, err := filepath.Glob(filepath.Join(srcDir, "*.bp"))
	if err != nil {
		return nil, fmt.Errorf("error globbing bp files: %w", err)
	}
	return bpFiles, nil
}

func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func setKeyValueConfigs(eval *parser.Evaluator, prefix, raw string) {
	if raw == "" {
		return
	}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		eval.SetConfig(prefix+key, val)
	}
}
