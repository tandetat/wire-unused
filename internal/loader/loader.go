// Package loader wraps go/packages to load
// wire-tagged packages with full type information.
package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

const mode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo

func Load(pattern string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode:       mode,
		BuildFlags: []string{"-tags=wireinject"},
	}

	// If pattern is a filesystem path (optionally with
	// /... suffix), resolve to dir + Go pattern.
	if dir, pat, ok := parseFilesystemPattern(
		pattern,
	); ok {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf(
				"abs path: %w", err)
		}
		cfg.Dir = abs
		pattern = pat
	}

	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, fmt.Errorf(
			"packages.Load: %w", err)
	}

	var errs []error
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			errs = append(errs,
				fmt.Errorf("%s: %s", pkg.PkgPath, e))
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf(
			"package errors: %v", errs)
	}

	return pkgs, nil
}

// parseFilesystemPattern checks if pattern is a
// filesystem path (optionally ending in /...) and
// returns (dir, goPattern, true) if so.
func parseFilesystemPattern(
	pattern string,
) (string, string, bool) {
	dir := pattern
	goPat := "."

	if strings.HasSuffix(dir, "/...") {
		dir = strings.TrimSuffix(dir, "/...")
		goPat = "./..."
	} else if strings.HasSuffix(dir, "...") {
		dir = strings.TrimSuffix(dir, "...")
		goPat = "./..."
	}

	if strings.HasPrefix(dir, "/") {
		return dir, goPat, true
	}
	if strings.HasPrefix(dir, "./") ||
		strings.HasPrefix(dir, "../") {
		info, err := os.Stat(dir)
		return dir, goPat, err == nil && info.IsDir()
	}
	info, err := os.Stat(dir)
	return dir, goPat, err == nil && info.IsDir()
}
