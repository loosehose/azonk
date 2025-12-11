// search.go provides SharePoint/OneDrive content search functionality.
package graph

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/loosehose/azonk/internal/config"
	"github.com/loosehose/azonk/internal/types"
	"github.com/loosehose/azonk/internal/ui"
)

// =============================================================================
// Search API Types
// =============================================================================

type searchRequest struct {
	Requests []searchRequestItem `json:"requests"`
}

type searchRequestItem struct {
	EntityTypes []string    `json:"entityTypes"`
	Query       searchQuery `json:"query"`
	From        int         `json:"from"`
	Size        int         `json:"size"`
}

type searchQuery struct {
	QueryString string `json:"queryString"`
}

type searchHit struct {
	Resource struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		WebURL          string `json:"webUrl"`
		Size            int64  `json:"size"`
		CreatedDateTime string `json:"createdDateTime"`
		LastModified    string `json:"lastModifiedDateTime"`
		ParentReference struct {
			DriveID string `json:"driveId"`
			Path    string `json:"path"`
		} `json:"parentReference"`
		CreatedBy struct {
			User struct {
				Email       string `json:"email"`
				DisplayName string `json:"displayName"`
			} `json:"user"`
		} `json:"createdBy"`
	} `json:"resource"`
}

// =============================================================================
// Search Methods
// =============================================================================

// Search performs a single search query against SharePoint/OneDrive.
func (c *Client) Search(query string, maxResults int) (*types.SearchResult, error) {
	if maxResults <= 0 {
		maxResults = config.DefaultMaxResultsPerQuery
	}

	req := searchRequest{
		Requests: []searchRequestItem{{
			EntityTypes: []string{"driveItem"},
			Query:       searchQuery{QueryString: query},
			From:        0,
			Size:        maxResults,
		}},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	data, err := c.Post("/search/query", payload)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}

	return parseSearchResponse(data, query)
}

// SearchWithOptions performs credential hunting with configurable options.
func (c *Client) SearchWithOptions(opts types.SearchOptions) ([]types.SearchResult, error) {
	ui.Info("Searching SharePoint/OneDrive...")

	if opts.MaxPerQuery <= 0 {
		opts.MaxPerQuery = config.DefaultMaxResultsPerQuery
	}

	var results []types.SearchResult
	seenIDs := make(map[string]bool)

	for _, keyword := range opts.Keywords {
		queries := buildQueries(keyword, opts.FileTypes, opts.IncludeKQL)

		for _, query := range queries {
			fmt.Printf("  %s\n", query)

			result, err := c.Search(query, opts.MaxPerQuery)
			if err != nil {
				ui.Error("Query failed: %v", err)
				continue
			}

			if result.TotalHits == 0 {
				continue
			}

			unique := deduplicateItems(result.Items, seenIDs)
			if len(unique) > 0 {
				ui.Success("Found %d results (%d new)", result.TotalHits, len(unique))
				result.Items = unique
				results = append(results, *result)
				printTopHits(unique, 3)
			}
		}
	}

	totalHits := 0
	for _, r := range results {
		totalHits += r.TotalHits
	}

	return results, nil
}

// SearchForCredentials is a convenience method using default credential keywords.
func (c *Client) SearchForCredentials() ([]types.SearchResult, error) {
	opts := types.SearchOptions{
		Keywords:    config.CredentialKeywords(),
		MaxPerQuery: config.DefaultMaxResultsPerQuery,
	}
	return c.SearchWithOptions(opts)
}

// =============================================================================
// Response Parsing
// =============================================================================

func parseSearchResponse(data []byte, query string) (*types.SearchResult, error) {
	var response struct {
		Value []struct {
			HitsContainers []struct {
				Total int         `json:"total"`
				Hits  []searchHit `json:"hits"`
			} `json:"hitsContainers"`
		} `json:"value"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	result := &types.SearchResult{Query: query}

	if len(response.Value) > 0 && len(response.Value[0].HitsContainers) > 0 {
		container := response.Value[0].HitsContainers[0]
		result.TotalHits = container.Total
		result.Items = make([]types.DriveItem, 0, len(container.Hits))

		for _, hit := range container.Hits {
			item := hitToDriveItem(hit, query)
			result.Items = append(result.Items, item)
		}
	}

	return result, nil
}

func hitToDriveItem(hit searchHit, query string) types.DriveItem {
	return types.DriveItem{
		ID:        hit.Resource.ID,
		DriveID:   hit.Resource.ParentReference.DriveID,
		Name:      hit.Resource.Name,
		Path:      hit.Resource.ParentReference.Path,
		WebURL:    hit.Resource.WebURL,
		Size:      hit.Resource.Size,
		Extension: strings.TrimPrefix(filepath.Ext(hit.Resource.Name), "."),
		Owner:     hit.Resource.CreatedBy.User.DisplayName,
		OwnerMail: hit.Resource.CreatedBy.User.Email,
		Created:   hit.Resource.CreatedDateTime,
		Modified:  hit.Resource.LastModified,
		MatchedOn: query,
	}
}

// =============================================================================
// Query Building
// =============================================================================

func buildQueries(keyword string, fileTypes []string, includeKQL bool) []string {
	queries := []string{keyword}

	if includeKQL && len(fileTypes) > 0 {
		for _, ft := range fileTypes {
			ft = strings.TrimPrefix(ft, ".")
			queries = append(queries, fmt.Sprintf("%s filetype:%s", keyword, ft))
		}
	}

	return queries
}

// =============================================================================
// Helpers
// =============================================================================

func deduplicateItems(items []types.DriveItem, seen map[string]bool) []types.DriveItem {
	unique := make([]types.DriveItem, 0)
	for _, item := range items {
		if !seen[item.ID] {
			seen[item.ID] = true
			unique = append(unique, item)
		}
	}
	return unique
}

func printTopHits(items []types.DriveItem, limit int) {
	for i, item := range items {
		if i >= limit {
			if remaining := len(items) - limit; remaining > 0 {
				fmt.Printf("    ... and %d more\n", remaining)
			}
			break
		}
		fmt.Printf("    %s\n", item.Name)
	}
}
