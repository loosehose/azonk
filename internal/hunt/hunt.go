// Package hunt orchestrates the complete credential hunting pipeline.
package hunt

import (
	"fmt"

	"github.com/loosehose/azonk/internal/config"
	"github.com/loosehose/azonk/internal/download"
	"github.com/loosehose/azonk/internal/extract"
	"github.com/loosehose/azonk/internal/graph"
	"github.com/loosehose/azonk/internal/types"
	"github.com/loosehose/azonk/internal/ui"
)

// Hunter coordinates search, download, and extraction operations.
type Hunter struct {
	client     *graph.Client
	downloader *download.Downloader
	extractor  *extract.Extractor
	outputDir  string
}

// NewHunter creates a Hunter with all required dependencies.
func NewHunter(accessToken, outputDir string) *Hunter {
	return &Hunter{
		client:     graph.NewClient(accessToken),
		downloader: download.NewDownloader(accessToken, outputDir),
		extractor:  extract.NewExtractor(),
		outputDir:  outputDir,
	}
}

// Run executes the complete hunt pipeline with the given options.
func (h *Hunter) Run(opts types.SearchOptions) (*types.HuntResult, error) {
	ui.Header("Credential Hunt")

	result := &types.HuntResult{}

	// Phase 1: Search
	ui.Phase(1, "Searching SharePoint/OneDrive")
	searchResults, err := h.client.SearchWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	result.SearchResults = searchResults

	totalHits, uniqueItems := aggregateSearchResults(searchResults)
	ui.Success("Found %d hits (%d unique files)", totalHits, len(uniqueItems))

	// Phase 2: Download (if enabled)
	if opts.AutoDownload && len(uniqueItems) > 0 {
		ui.Phase(2, "Downloading files")

		extensions := opts.FileTypes
		if len(extensions) == 0 {
			extensions = config.HighValueExtensions()
		}

		downloaded := h.downloader.DownloadFromSearchResults(searchResults, extensions)
		result.DownloadedFiles = downloaded

		// Phase 3: Extract secrets (if enabled)
		if opts.ExtractSecret && len(downloaded) > 0 {
			ui.Phase(3, "Extracting secrets")
			secrets := h.extractor.ScanDownloadedFiles(downloaded)
			result.SecretsFound = secrets
			h.extractor.PrintMatches(secrets)
		}
	}

	// Build summary
	result.Summary = types.HuntSummary{
		QueriesRun:      len(opts.Keywords),
		TotalHits:       totalHits,
		UniqueFiles:     len(uniqueItems),
		FilesDownloaded: len(result.DownloadedFiles),
		SecretsFound:    len(result.SecretsFound),
	}

	h.printSummary(result.Summary)
	return result, nil
}

// QuickHunt runs a hunt with sensible defaults for credential hunting.
func (h *Hunter) QuickHunt() (*types.HuntResult, error) {
	opts := types.SearchOptions{
		Keywords:      config.CredentialKeywords(),
		FileTypes:     config.HighValueExtensions(),
		MaxPerQuery:   config.DefaultMaxResultsPerQuery,
		AutoDownload:  true,
		ExtractSecret: true,
	}
	return h.Run(opts)
}

// SearchOnly performs search without downloading or extracting.
func (h *Hunter) SearchOnly(keywords []string, fileTypes []string, useKQL bool) ([]types.SearchResult, error) {
	opts := types.SearchOptions{
		Keywords:    keywords,
		FileTypes:   fileTypes,
		MaxPerQuery: config.DefaultMaxResultsPerQuery,
		IncludeKQL:  useKQL,
	}
	return h.client.SearchWithOptions(opts)
}

// GetDownloadDir returns the path where files are downloaded.
func (h *Hunter) GetDownloadDir() string {
	return h.downloader.GetOutputDir()
}

// =============================================================================
// Helpers
// =============================================================================

func aggregateSearchResults(results []types.SearchResult) (int, map[string]types.DriveItem) {
	totalHits := 0
	uniqueItems := make(map[string]types.DriveItem)

	for _, sr := range results {
		totalHits += sr.TotalHits
		for _, item := range sr.Items {
			uniqueItems[item.ID] = item
		}
	}

	return totalHits, uniqueItems
}

func (h *Hunter) printSummary(s types.HuntSummary) {
	ui.Header("Summary")
	ui.Stat("Queries run", s.QueriesRun)
	ui.Stat("Total hits", s.TotalHits)
	ui.Stat("Unique files", s.UniqueFiles)
	ui.Stat("Files downloaded", s.FilesDownloaded)

	if s.SecretsFound > 0 {
		ui.StatHighlight("Secrets found", s.SecretsFound)
	} else {
		ui.Stat("Secrets found", s.SecretsFound)
	}

	fmt.Println()
	ui.Success("Results saved to %s", h.outputDir)
}
