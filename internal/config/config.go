// Package config provides centralized configuration and constants for azonk.
// All shared values, defaults, and configuration structures are defined here
// to ensure consistency across packages and eliminate magic strings/numbers.
package config

import "time"

// =============================================================================
// OAuth Configuration
// =============================================================================

const (
	// MicrosoftOfficeClientID is the well-known client ID for Microsoft Office.
	// This client ID is used by GraphRunner and provides broad M365 scopes
	// including Files.Read.All, Mail.ReadWrite, and Directory.Read.All when
	// authenticated from trusted networks.
	// Source: https://github.com/dafthack/GraphRunner
	MicrosoftOfficeClientID = "d3590ed6-52b3-4102-aeff-aad2292ab01c"

	// DeviceCodeEndpoint is the OAuth 2.0 device authorization endpoint.
	// Uses v1.0 API which accepts 'resource' parameter instead of 'scope'.
	DeviceCodeEndpoint = "https://login.microsoftonline.com/common/oauth2/devicecode?api-version=1.0"

	// TokenEndpoint is the OAuth 2.0 token endpoint for device code redemption.
	TokenEndpoint = "https://login.microsoftonline.com/Common/oauth2/token?api-version=1.0"

	// GraphResource is the resource identifier for Microsoft Graph API.
	// Used with OAuth v1.0 flow instead of scopes.
	GraphResource = "https://graph.microsoft.com"
)

// =============================================================================
// Microsoft Graph API Configuration
// =============================================================================

const (
	// GraphBaseURL is the base URL for Microsoft Graph API v1.0 endpoints.
	GraphBaseURL = "https://graph.microsoft.com/v1.0"

	// GraphBetaURL is the base URL for Microsoft Graph API beta endpoints.
	// Beta endpoints provide access to preview features but may change.
	GraphBetaURL = "https://graph.microsoft.com/beta"
)

// =============================================================================
// HTTP Client Configuration
// =============================================================================

const (
	// DefaultHTTPTimeout is the default timeout for HTTP requests.
	DefaultHTTPTimeout = 30 * time.Second

	// DownloadTimeout is the timeout for file download operations.
	// Longer timeout to accommodate large files.
	DownloadTimeout = 5 * time.Minute

	// RateLimitDelay is the delay between paginated API requests
	// to avoid throttling by Microsoft Graph.
	RateLimitDelay = 100 * time.Millisecond

	// DownloadRateLimitDelay is the delay between file downloads.
	DownloadRateLimitDelay = 200 * time.Millisecond
)

// =============================================================================
// Search Configuration
// =============================================================================

const (
	// DefaultMaxResultsPerQuery limits results per search query.
	DefaultMaxResultsPerQuery = 25

	// MaxFileSizeForScan is the maximum file size (50MB) for secret scanning.
	// Files larger than this are skipped to avoid memory issues.
	MaxFileSizeForScan = 50 * 1024 * 1024
)

// =============================================================================
// User Agent
// =============================================================================

const (
	// UserAgent mimics a standard browser to avoid detection/blocking.
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// =============================================================================
// Default Keywords and File Types
// =============================================================================

// CredentialKeywords returns search terms commonly associated with credentials.
// These are used for SharePoint/OneDrive content searches.
func CredentialKeywords() []string {
	return []string{
		"password",
		"credential",
		"secret",
		"api key",
		"apikey",
		"client_secret",
		"client secret",
		"connection string",
		"connectionstring",
		"private key",
		"privatekey",
		"bearer token",
		"access token",
		"service principal",
		"certificate",
		".pfx",
		".pem",
		".key",
	}
}

// HighValueExtensions returns file extensions that commonly contain credentials.
// These are prioritized for automatic download during hunt operations.
func HighValueExtensions() []string {
	return []string{
		// Spreadsheets - often contain credential inventories
		"xlsx", "xls", "csv",
		// Plain text and logs
		"txt", "log",
		// Configuration files
		"json", "xml", "yaml", "yml", "config", "conf", "ini", "env",
		// Scripts that may contain hardcoded credentials
		"ps1", "sh", "bat", "cmd",
		// Database and infrastructure
		"sql", "tf",
		// Backup files
		"bak",
	}
}

// ScannableExtensions returns file extensions that can be scanned for secrets.
// Binary files and non-text formats are excluded.
func ScannableExtensions() map[string]bool {
	return map[string]bool{
		// Text and logs
		".txt": true, ".log": true, ".csv": true,
		// Data formats
		".json": true, ".xml": true, ".yaml": true, ".yml": true,
		// Config files
		".config": true, ".ini": true, ".env": true, ".conf": true, ".properties": true,
		// Scripts
		".ps1": true, ".sh": true, ".bat": true, ".cmd": true,
		// Source code
		".py": true, ".js": true, ".ts": true, ".go": true,
		".cs": true, ".java": true, ".php": true, ".rb": true,
		// Documentation and infrastructure
		".md": true, ".rst": true, ".sql": true, ".tf": true,
	}
}
