// users.go provides Azure AD user enumeration functionality.
package graph

import (
	"encoding/json"

	"github.com/loosehose/azonk/internal/types"
	"github.com/loosehose/azonk/internal/ui"
)

// EnumerateUsers retrieves all users from Azure AD directory.
func (c *Client) EnumerateUsers() ([]types.User, error) {
	ui.Info("Enumerating Azure AD users...")

	endpoint := "/users?$select=id,displayName,userPrincipalName,mail,jobTitle,department,accountEnabled&$top=999"

	results, err := c.GetAllPages(endpoint, 0)
	if err != nil {
		return nil, err
	}

	users := make([]types.User, 0, len(results))
	for _, raw := range results {
		var user types.User
		if err := json.Unmarshal(raw, &user); err != nil {
			continue
		}
		users = append(users, user)
	}

	ui.Success("Found %d users", len(users))
	return users, nil
}

// GetMe retrieves the currently authenticated user.
func (c *Client) GetMe() (*types.User, error) {
	data, err := c.Get("/me")
	if err != nil {
		return nil, err
	}

	var user types.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}

	return &user, nil
}
