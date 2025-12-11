// Package extract provides secret detection via regex pattern matching.
package extract

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loosehose/azonk/internal/config"
	"github.com/loosehose/azonk/internal/types"
	"github.com/loosehose/azonk/internal/ui"
)

// =============================================================================
// Secret Patterns
// =============================================================================

type pattern struct {
	name  string
	regex *regexp.Regexp
}

func secretPatterns() []pattern {
	return []pattern{
		// Azure/Microsoft
		{name: "Azure Client Secret", regex: regexp.MustCompile(`(?i)(client[_-]?secret|clientsecret)["'\s:=]+([A-Za-z0-9~._\-]{30,})`)},
		{name: "Azure Tenant/App ID", regex: regexp.MustCompile(`(?i)(tenant[_-]?id|app[_-]?id|client[_-]?id|application[_-]?id)["'\s:=]+([a-f0-9\-]{36})`)},
		{name: "Azure Storage Key", regex: regexp.MustCompile(`(?i)(account[_-]?key|storage[_-]?key)["'\s:=]+([A-Za-z0-9+/=]{60,})`)},
		{name: "Azure SAS Token", regex: regexp.MustCompile(`(\?sv=.+&sig=[A-Za-z0-9%]+)`)},

		// Generic Credentials
		{name: "Password", regex: regexp.MustCompile(`(?i)(password|passwd|pwd)["'\s:=]+([^\s"',\]\}]{4,50})`)},
		{name: "API Key", regex: regexp.MustCompile(`(?i)(api[_-]?key|apikey)["'\s:=]+([A-Za-z0-9_\-]{16,})`)},
		{name: "Generic Secret", regex: regexp.MustCompile(`(?i)(secret[_-]?id|secret[_-]?key)["'\s:=]+([^\s"',]{8,})`)},
		{name: "Bearer Token", regex: regexp.MustCompile(`(?i)(bearer|authorization)["'\s:=]+(eyJ[A-Za-z0-9_\-]+\.eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+)`)},

		// Connection Strings
		{name: "Connection String", regex: regexp.MustCompile(`(?i)(connection[_-]?string|connstring|connectionstring)["'\s:=]+([^"'\n]{20,})`)},
		{name: "SQL Connection", regex: regexp.MustCompile(`(?i)Server=.+;.*(Password|Pwd)=([^;]+)`)},

		// Private Keys
		{name: "Private Key", regex: regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
		{name: "PFX/PKCS12", regex: regexp.MustCompile(`(?i)(\.pfx|\.p12|pkcs12)["'\s:=]+([^\s"',]+)`)},

		// AWS
		{name: "AWS Access Key", regex: regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
		{name: "AWS Secret Key", regex: regexp.MustCompile(`(?i)(aws[_-]?secret|secret[_-]?access[_-]?key)["'\s:=]+([A-Za-z0-9/+=]{40})`)},

		// GCP
		{name: "GCP API Key", regex: regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`)},
		{name: "GCP Service Account", regex: regexp.MustCompile(`"type"\s*:\s*"service_account"`)},

		// GitHub/GitLab
		{name: "GitHub Token", regex: regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{36,}`)},
		{name: "GitLab Token", regex: regexp.MustCompile(`glpat-[A-Za-z0-9\-]{20,}`)},

		// Slack
		{name: "Slack Token", regex: regexp.MustCompile(`xox[baprs]-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*`)},

		// Stripe
		{name: "Stripe Key", regex: regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24,}`)},
	}
}

// =============================================================================
// Extractor
// =============================================================================

type Extractor struct {
	patterns []pattern
}

func NewExtractor() *Extractor {
	return &Extractor{patterns: secretPatterns()}
}

// =============================================================================
// Scanning Methods
// =============================================================================

func (e *Extractor) ScanFile(filePath string) ([]types.SecretMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []types.SecretMatch
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, p := range e.patterns {
			if p.regex.MatchString(line) {
				matches = append(matches, types.SecretMatch{
					File:        filePath,
					Line:        lineNum,
					PatternName: p.name,
					Match:       p.regex.FindString(line),
					Context:     truncateContext(line, 100),
				})
			}
		}
	}

	return matches, scanner.Err()
}

func (e *Extractor) ScanDirectory(dirPath string) ([]types.SecretMatch, error) {
	ui.Info("Scanning for secrets: %s", dirPath)

	var allMatches []types.SecretMatch
	scannableExts := config.ScannableExtensions()

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if info.Size() > config.MaxFileSizeForScan {
			return nil
		}

		if !scannableExts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}

		matches, err := e.ScanFile(path)
		if err != nil {
			return nil
		}

		if len(matches) > 0 {
			ui.Warning("Found %d secrets in %s", len(matches), filepath.Base(path))
			allMatches = append(allMatches, matches...)
		}

		return nil
	})

	if err != nil {
		return allMatches, err
	}

	ui.Success("Scan complete: %d potential secrets found", len(allMatches))
	return allMatches, nil
}

func (e *Extractor) ScanDownloadedFiles(downloads []types.DownloadedFile) []types.SecretMatch {
	ui.Info("Scanning %d files for secrets...", len(downloads))

	var allMatches []types.SecretMatch

	for _, dl := range downloads {
		matches, err := e.ScanFile(dl.LocalPath)
		if err != nil {
			continue
		}

		for i := range matches {
			matches[i].SourceItem = dl.SourceItem.Name
		}

		if len(matches) > 0 {
			ui.Warning("Found %d secrets in %s", len(matches), filepath.Base(dl.LocalPath))
			allMatches = append(allMatches, matches...)
		}
	}

	ui.Success("Extracted %d potential secrets", len(allMatches))
	return allMatches
}

// =============================================================================
// Output Methods
// =============================================================================

func (e *Extractor) PrintMatches(matches []types.SecretMatch) {
	if len(matches) == 0 {
		return
	}

	// Group by file
	byFile := make(map[string][]types.SecretMatch)
	for _, m := range matches {
		byFile[m.File] = append(byFile[m.File], m)
	}

	fmt.Println()
	for file, fileMatches := range byFile {
		fmt.Printf("  %s\n", filepath.Base(file))
		for _, m := range fileMatches {
			ui.Critical("[%s] Line %d", m.PatternName, m.Line)
			fmt.Printf("      %s\n", ui.Dim(m.Context))
		}
		fmt.Println()
	}
}

// =============================================================================
// Helpers
// =============================================================================

func truncateContext(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
