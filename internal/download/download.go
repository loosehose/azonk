// Package download handles file retrieval from SharePoint/OneDrive via Graph API.
package download

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/loosehose/azonk/internal/config"
	"github.com/loosehose/azonk/internal/types"
	"github.com/loosehose/azonk/internal/ui"
)

// Downloader manages file downloads from SharePoint/OneDrive.
type Downloader struct {
	accessToken string
	outputDir   string
	httpClient  *http.Client
}

// NewDownloader creates a Downloader with the specified output directory.
func NewDownloader(accessToken, outputDir string) *Downloader {
	downloadDir := filepath.Join(outputDir, "downloads")
	os.MkdirAll(downloadDir, 0755)

	return &Downloader{
		accessToken: accessToken,
		outputDir:   downloadDir,
		httpClient: &http.Client{
			Timeout: config.DownloadTimeout,
		},
	}
}

// DownloadItem downloads a single DriveItem and returns metadata about the download.
func (d *Downloader) DownloadItem(item types.DriveItem) (*types.DownloadedFile, error) {
	downloadURL, err := d.getDownloadURL(item.DriveID, item.ID)
	if err != nil {
		return nil, fmt.Errorf("get download URL: %w", err)
	}

	localPath, bytesWritten, err := d.downloadFile(downloadURL, item.Name)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	return &types.DownloadedFile{
		SourceItem: item,
		LocalPath:  localPath,
		BytesSize:  bytesWritten,
	}, nil
}

// DownloadBatch downloads multiple items with optional extension filtering.
func (d *Downloader) DownloadBatch(items []types.DriveItem, extensions []string) []types.DownloadedFile {
	ui.Info("Downloading %d files...", len(items))

	extFilter := buildExtensionFilter(extensions)

	var downloaded []types.DownloadedFile
	for i, item := range items {
		if len(extFilter) > 0 && !matchesExtension(item.Name, extFilter) {
			continue
		}

		fmt.Printf("  [%d/%d] %s\n", i+1, len(items), item.Name)

		result, err := d.DownloadItem(item)
		if err != nil {
			ui.Error("Failed: %v", err)
			continue
		}

		fmt.Printf("         %s\n", ui.Dim(formatBytes(result.BytesSize)))
		downloaded = append(downloaded, *result)

		time.Sleep(config.DownloadRateLimitDelay)
	}

	ui.Success("Downloaded %d/%d files", len(downloaded), len(items))
	return downloaded
}

// DownloadFromSearchResults extracts items from search results and downloads them.
func (d *Downloader) DownloadFromSearchResults(results []types.SearchResult, extensions []string) []types.DownloadedFile {
	items := collectUniqueItems(results)

	if len(items) == 0 {
		ui.Warning("No files to download")
		return nil
	}

	return d.DownloadBatch(items, extensions)
}

// DownloadByID downloads a file using drive ID and item ID directly.
func (d *Downloader) DownloadByID(driveID, itemID, filename string) (string, error) {
	downloadURL, err := d.getDownloadURL(driveID, itemID)
	if err != nil {
		return "", err
	}

	if filename == "" {
		filename = "downloaded_file"
	}

	path, _, err := d.downloadFile(downloadURL, filename)
	return path, err
}

// GetOutputDir returns the download directory path.
func (d *Downloader) GetOutputDir() string {
	return d.outputDir
}

// =============================================================================
// Internal Methods
// =============================================================================

func (d *Downloader) getDownloadURL(driveID, itemID string) (string, error) {
	url := fmt.Sprintf("%s/drives/%s/items/%s?select=@microsoft.graph.downloadUrl,name",
		config.GraphBaseURL, driveID, itemID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("User-Agent", config.UserAgent)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API error %d", resp.StatusCode)
	}

	var result struct {
		DownloadURL string `json:"@microsoft.graph.downloadUrl"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.DownloadURL == "" {
		return "", fmt.Errorf("no download URL in response")
	}

	return result.DownloadURL, nil
}

func (d *Downloader) downloadFile(url, filename string) (string, int64, error) {
	filename = sanitizeFilename(filename)
	outputPath := filepath.Join(d.outputDir, filename)
	outputPath = d.resolveCollision(outputPath)

	resp, err := http.Get(url)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", 0, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return "", 0, err
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return "", 0, err
	}

	return outputPath, written, nil
}

// =============================================================================
// Helpers
// =============================================================================

func buildExtensionFilter(extensions []string) map[string]bool {
	filter := make(map[string]bool)
	for _, ext := range extensions {
		ext = strings.ToLower(strings.TrimPrefix(ext, "."))
		filter[ext] = true
	}
	return filter
}

func matchesExtension(filename string, filter map[string]bool) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	return filter[ext]
}

func collectUniqueItems(results []types.SearchResult) []types.DriveItem {
	seen := make(map[string]bool)
	var items []types.DriveItem

	for _, result := range results {
		for _, item := range result.Items {
			if !seen[item.ID] {
				seen[item.ID] = true
				items = append(items, item)
			}
		}
	}

	return items
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	replacer := strings.NewReplacer(
		"<", "_", ">", "_", ":", "_", "\"", "_",
		"|", "_", "?", "_", "*", "_",
	)
	return replacer.Replace(name)
}

func (d *Downloader) resolveCollision(path string) string {
	if _, err := os.Stat(path); err == nil {
		ext := filepath.Ext(path)
		base := strings.TrimSuffix(path, ext)
		return fmt.Sprintf("%s_%d%s", base, time.Now().Unix(), ext)
	}
	return path
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
