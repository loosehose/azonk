// Package types provides shared data structures for the azonk tool.
// Centralizing types here ensures consistency across packages and enables
// seamless data flow between authentication, search, download, and extraction.
package types

import "time"

// =============================================================================
// Authentication Types
// =============================================================================

// TokenResponse holds OAuth tokens returned from Microsoft identity platform.
// Supports both v1.0 (resource-based) and v2.0 (scope-based) responses.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    string    `json:"expires_in"`  // Returned as string in v1.0
	ExpiresOn    string    `json:"expires_on"`  // Unix timestamp as string
	Resource     string    `json:"resource"`    // v1.0 only
	Scope        string    `json:"scope"`       // v2.0 only
	ExpiresAt    time.Time `json:"expires_at"`  // Computed expiration time
}

// DeviceCodeResponse contains the device code flow initiation response.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"` // v1.0 uses underscore
	VerificationURI string `json:"verification_uri"` // v2.0 uses URI
	ExpiresIn       string `json:"expires_in"`
	Interval        string `json:"interval"`
	Message         string `json:"message"`
}

// GetVerificationURL returns the verification URL from either v1.0 or v2.0 response.
func (d *DeviceCodeResponse) GetVerificationURL() string {
	if d.VerificationURL != "" {
		return d.VerificationURL
	}
	return d.VerificationURI
}

// =============================================================================
// Azure AD Types
// =============================================================================

// User represents an Azure AD user account.
type User struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	UserPrincipalName string `json:"userPrincipalName"`
	Mail              string `json:"mail"`
	JobTitle          string `json:"jobTitle"`
	Department        string `json:"department"`
	AccountEnabled    bool   `json:"accountEnabled"`
}

// DirectoryRole represents an Azure AD directory role (e.g., Global Administrator).
type DirectoryRole struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	RoleTemplateID string `json:"roleTemplateId"`
}

// RoleMember represents a member of a directory role.
type RoleMember struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	UserPrincipalName string `json:"userPrincipalName"`
	ODataType         string `json:"@odata.type"`
}

// IsServicePrincipal returns true if the member is a service principal (not a user).
func (r *RoleMember) IsServicePrincipal() bool {
	return r.UserPrincipalName == "" || r.ODataType == "#microsoft.graph.servicePrincipal"
}

// RoleWithMembers combines a directory role with its members.
type RoleWithMembers struct {
	Role    DirectoryRole `json:"role"`
	Members []RoleMember  `json:"members"`
}

// =============================================================================
// SharePoint/OneDrive Types
// =============================================================================

// DriveItem represents a file in SharePoint/OneDrive with metadata
// needed for search results, filtering, and download operations.
type DriveItem struct {
	ID        string `json:"id"`
	DriveID   string `json:"driveId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	WebURL    string `json:"webUrl"`
	Size      int64  `json:"size"`
	Extension string `json:"extension"`
	Owner     string `json:"owner"`
	OwnerMail string `json:"ownerEmail"`
	Created   string `json:"created"`
	Modified  string `json:"modified"`
	MatchedOn string `json:"matchedOn,omitempty"` // Search term that matched
}

// SearchResult aggregates results from a single search query.
type SearchResult struct {
	Query     string      `json:"query"`
	TotalHits int         `json:"totalHits"`
	Items     []DriveItem `json:"items"`
}

// =============================================================================
// Download Types
// =============================================================================

// DownloadedFile tracks a successfully downloaded file with its source metadata.
type DownloadedFile struct {
	SourceItem DriveItem `json:"sourceItem"`
	LocalPath  string    `json:"localPath"`
	BytesSize  int64     `json:"bytesSize"`
}

// =============================================================================
// Secret Extraction Types
// =============================================================================

// SecretMatch represents a potential secret found during extraction.
type SecretMatch struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	PatternName string `json:"patternName"`
	Match       string `json:"match"`
	Context     string `json:"context"`              // Surrounding text for context
	SourceItem  string `json:"sourceItem,omitempty"` // Original SharePoint item name
}

// =============================================================================
// Hunt Pipeline Types
// =============================================================================

// SearchOptions configures search behavior including keywords and file filtering.
type SearchOptions struct {
	Keywords      []string // Search terms to query
	FileTypes     []string // File extensions to filter (e.g., "xlsx", "docx")
	MaxPerQuery   int      // Maximum results per query (default 25)
	IncludeKQL    bool     // Use KQL filetype: syntax in queries
	AutoDownload  bool     // Automatically download matching files
	ExtractSecret bool     // Run secret extraction on downloaded files
}

// HuntResult represents the complete output of a hunt operation,
// combining search results with download and extraction outcomes.
type HuntResult struct {
	SearchResults   []SearchResult   `json:"searchResults"`
	DownloadedFiles []DownloadedFile `json:"downloadedFiles"`
	SecretsFound    []SecretMatch    `json:"secretsFound"`
	Summary         HuntSummary      `json:"summary"`
}

// HuntSummary provides aggregate statistics for a hunt operation.
type HuntSummary struct {
	QueriesRun      int `json:"queriesRun"`
	TotalHits       int `json:"totalHits"`
	UniqueFiles     int `json:"uniqueFiles"`
	FilesDownloaded int `json:"filesDownloaded"`
	SecretsFound    int `json:"secretsFound"`
}

// =============================================================================
// Assessment Types
// =============================================================================

// AssessmentResult contains all findings from a full assessment.
type AssessmentResult struct {
	Users        []User            `json:"users,omitempty"`
	GlobalAdmins *RoleWithMembers  `json:"globalAdmins,omitempty"`
	Roles        []RoleWithMembers `json:"roles,omitempty"`
	HuntResult   *HuntResult       `json:"huntResult,omitempty"`
	Summary      AssessmentSummary `json:"summary"`
}

// AssessmentSummary provides high-level statistics for the assessment.
type AssessmentSummary struct {
	UsersFound      int    `json:"usersFound"`
	GlobalAdmins    int    `json:"globalAdmins"`
	RolesWithMembers int   `json:"rolesWithMembers"`
	SearchHits      int    `json:"searchHits"`
	FilesDownloaded int    `json:"filesDownloaded"`
	SecretsFound    int    `json:"secretsFound"`
	Timestamp       string `json:"timestamp"`
}
