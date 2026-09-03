package context

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	jsTestRegex = regexp.MustCompile(`(?:describe|test|it)\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`)
	goTestRegex = regexp.MustCompile(`func\s+(Test[A-Za-z0-9_]+)\s*\(`)
	pyTestRegex = regexp.MustCompile(`def\s+(test_[A-Za-z0-9_]+)\s*\(`)

	goExportRegex = regexp.MustCompile(`(?:type|func)\s+([A-Z][A-Za-z0-9_]+)`)
	tsExportRegex = regexp.MustCompile(`export\s+(?:class|interface|type|function|const|enum)\s+([A-Za-z0-9_]+)`)
	pyExportRegex = regexp.MustCompile(`(?:class|def)\s+([A-Za-z0-9_]+)`)
)

// CompressFile generates a deterministic structural outline of a file.
// It extracts key declarations, test cases, or headings, returning the summary text
// and an estimated token count for that summary.
func CompressFile(root, path string) (string, int) {
	fullPath := filepath.Join(root, path)
	f, err := os.Open(fullPath)
	if err != nil {
		summary := fmt.Sprintf("[outline: %s | unavailable]", path)
		return summary, max(15, len(summary)/4)
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))
	isTest := strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".spec.js") ||
		strings.HasPrefix(base, "test_")

	scanner := bufio.NewScanner(f)
	var lineCount int
	var items []string
	seen := map[string]bool{}

	addItem := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] || len(items) >= 10 {
			return
		}
		seen[s] = true
		items = append(items, s)
	}

	for scanner.Scan() && lineCount < 1000 {
		lineCount++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if isTest {
			if matches := jsTestRegex.FindStringSubmatch(line); len(matches) > 1 {
				addItem(matches[1])
			} else if matches := goTestRegex.FindStringSubmatch(line); len(matches) > 1 {
				addItem(matches[1])
			} else if matches := pyTestRegex.FindStringSubmatch(line); len(matches) > 1 {
				addItem(matches[1])
			}
			continue
		}

		if ext == ".md" {
			if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
				heading := strings.TrimLeft(line, "# ")
				addItem(heading)
			}
			continue
		}

		if ext == ".go" {
			if matches := goExportRegex.FindStringSubmatch(line); len(matches) > 1 {
				addItem(matches[1])
			}
			continue
		}

		if ext == ".ts" || ext == ".js" {
			if matches := tsExportRegex.FindStringSubmatch(line); len(matches) > 1 {
				addItem(matches[1])
			}
			continue
		}

		if ext == ".py" {
			if matches := pyExportRegex.FindStringSubmatch(line); len(matches) > 1 {
				addItem(matches[1])
			}
			continue
		}
	}

	for scanner.Scan() {
		lineCount++
	}

	category := "structure"
	if isTest {
		category = "tests"
	} else if ext == ".md" {
		category = "headings"
	} else if len(items) > 0 {
		category = "exports"
	}

	var summary string
	if len(items) > 0 {
		summary = fmt.Sprintf("[outline: %s (%d lines) | %s: %s]", path, lineCount, category, strings.Join(items, ", "))
	} else {
		summary = fmt.Sprintf("[outline: %s (%d lines)]", path, lineCount)
	}

	tokens := max(15, len(summary)/4)
	return summary, tokens
}
