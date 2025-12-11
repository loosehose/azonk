# azonk

```
                                 ▀██      ▄█▄
 ▄▄▄▄   ▄▄▄▄▄▄    ▄▄▄   ▄▄ ▄▄▄    ██  ▄▄  ███
▀▀ ▄██  ▀  ▄█▀  ▄█  ▀█▄  ██  ██   ██ ▄▀   ▀█▀
▄█▀ ██   ▄█▀    ██   ██  ██  ██   ██▀█▄    █
▀█▄▄▀█▀ ██▄▄▄▄█  ▀█▄▄█▀ ▄██▄ ██▄ ▄██▄ ██▄  ▄
                                          ▀█▀
```

SharePoint/OneDrive Secrets Finder - Automated credential hunting with Azure AD enumeration.

## Features

- **Native Device Code Authentication** - No dependencies on external tools
- **User Enumeration** - List all Azure AD users
- **Admin Discovery** - Find Global Administrators and privileged roles
- **Credential Hunting** - Search SharePoint/OneDrive for secrets
- **File Download** - Download discovered files with extension filtering
- **Secret Extraction** - Regex-based secret scanning (20+ patterns)
- **Full Assessment** - One command to run the complete pipeline

## Installation

### Prerequisites
- Go 1.21+ (https://go.dev/dl/)

### Build

```bash
cd azonk

# Download dependencies
go mod tidy

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -o azonk.exe ./cmd/azonk

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o azonk ./cmd/azonk
```

## Usage

### Full Assessment (Recommended)
```bash
# Complete assessment: enum + search + download + extract
./azonk assess

# Assessment without file downloads
./azonk assess --no-download
```

### Credential Hunting
```bash
# Full hunt: search + download + extract
./azonk hunt

# Hunt for specific term
./azonk hunt --term "client_secret"

# Filter by file type
./azonk hunt --filetype xlsx,csv,json

# Search only (no downloads)
./azonk hunt --download=false
```

### Search Only
```bash
# Search with default keywords
./azonk search

# Search for specific term
./azonk search --term "password"

# Use KQL syntax for advanced queries
./azonk search --term "api key" --kql --filetype xlsx
```

### Enumeration
```bash
# Enumerate all users
./azonk enum users

# Find Global Administrators
./azonk enum admins

# List all directory roles with members
./azonk enum roles
```

### Download & Extract
```bash
# Download a specific file
./azonk download --drive-id <DRIVE_ID> --item-id <ITEM_ID>

# Extract secrets from downloaded files
./azonk extract --path ./azonk_output/downloads
```

### Authentication Options
```bash
# Interactive device code auth (default)
./azonk assess

# Use existing access token
./azonk assess -t "eyJ0eXAiOiJKV1Qi..."

# Authenticate only
./azonk auth
```

## Global Options

```
-o, --output string   Output directory (default "./azonk_output")
-t, --token string    Access token (skip device code auth)
-v, --verbose         Verbose output
```

## Output Files

```
azonk_output/
├── tokens.json           # Cached auth tokens
├── users.json            # Enumerated users
├── admins.json           # Global administrators
├── roles.json            # Directory roles with members
├── search_results.json   # Credential search hits
├── hunt_results.json     # Hunt pipeline results
├── secrets_found.json    # Extracted secrets
├── assessment.json       # Full assessment results
└── downloads/            # Downloaded files
```

## Default Search Keywords

```
password, credential, secret, api key, apikey,
client_secret, client secret, connection string,
private key, bearer token, access token,
service principal, certificate, .pfx, .pem, .key
```

## Secret Patterns Detected

| Category | Patterns |
|----------|----------|
| Azure | Client Secrets, Tenant IDs, Storage Keys, SAS Tokens |
| Generic | Passwords, API Keys, Bearer Tokens, Connection Strings |
| AWS | Access Keys, Secret Keys |
| GCP | API Keys, Service Accounts |
| GitHub/GitLab | Personal Access Tokens |
| Other | Slack Tokens, Stripe Keys, Private Keys |

## High-Value File Extensions

Automatically downloaded when hunting:
```
xlsx, xls, csv, txt, log, json, xml, yaml, yml,
config, conf, ini, env, ps1, sh, bat, cmd, sql, tf, bak
```

## Architecture

```
azonk/
├── cmd/azonk/main.go           # CLI entry point
└── internal/
    ├── config/config.go        # Centralized configuration
    ├── types/types.go          # Shared data structures
    ├── auth/auth.go            # Device code authentication
    ├── graph/
    │   ├── client.go           # Graph API HTTP client
    │   ├── users.go            # User enumeration
    │   ├── roles.go            # Role/admin discovery
    │   └── search.go           # SharePoint/OneDrive search
    ├── download/download.go    # File download
    ├── extract/extract.go      # Secret extraction
    ├── hunt/hunt.go            # Pipeline orchestration
    └── output/output.go        # JSON file output
```

## Graph API Permissions Used

| Permission | Usage |
|------------|-------|
| User.Read.All | User enumeration |
| Directory.Read.All | Role and admin enumeration |
| Files.Read.All | SharePoint/OneDrive search and download |

## Legal

This tool is intended for **authorized security assessments only**.
Always obtain proper written authorization before testing.
