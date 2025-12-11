// roles.go provides Azure AD directory role enumeration and admin discovery.
package graph

import (
	"encoding/json"
	"fmt"

	"github.com/loosehose/azonk/internal/types"
	"github.com/loosehose/azonk/internal/ui"
)

const GlobalAdminRoleName = "Global Administrator"

// EnumerateDirectoryRoles retrieves all activated directory roles.
func (c *Client) EnumerateDirectoryRoles() ([]types.DirectoryRole, error) {
	ui.Info("Enumerating directory roles...")

	data, err := c.Get("/directoryRoles")
	if err != nil {
		return nil, err
	}

	var result struct {
		Value []types.DirectoryRole `json:"value"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	ui.Success("Found %d activated roles", len(result.Value))
	return result.Value, nil
}

// GetRoleMembers retrieves all members of a specific directory role.
func (c *Client) GetRoleMembers(roleID string) ([]types.RoleMember, error) {
	data, err := c.Get("/directoryRoles/" + roleID + "/members")
	if err != nil {
		return nil, err
	}

	var result struct {
		Value []types.RoleMember `json:"value"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result.Value, nil
}

// GetGlobalAdmins finds all members of the Global Administrator role.
func (c *Client) GetGlobalAdmins() (*types.RoleWithMembers, error) {
	ui.Info("Searching for Global Administrators...")

	roles, err := c.EnumerateDirectoryRoles()
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if role.DisplayName == GlobalAdminRoleName {
			members, err := c.GetRoleMembers(role.ID)
			if err != nil {
				return nil, err
			}

			ui.Success("Found %d Global Administrators", len(members))
			for _, m := range members {
				if m.IsServicePrincipal() {
					ui.Finding("%s [Service Principal]", m.DisplayName)
				} else {
					ui.Finding("%s (%s)", m.DisplayName, m.UserPrincipalName)
				}
			}

			return &types.RoleWithMembers{
				Role:    role,
				Members: members,
			}, nil
		}
	}

	ui.Warning("Global Administrator role not found (may not be activated)")
	return nil, nil
}

// EnumerateAllRolesWithMembers retrieves all directory roles and their members.
func (c *Client) EnumerateAllRolesWithMembers() ([]types.RoleWithMembers, error) {
	ui.Info("Enumerating all roles with members...")

	roles, err := c.EnumerateDirectoryRoles()
	if err != nil {
		return nil, err
	}

	var results []types.RoleWithMembers

	for _, role := range roles {
		members, err := c.GetRoleMembers(role.ID)
		if err != nil {
			ui.Error("Failed to get members for %s: %v", role.DisplayName, err)
			continue
		}

		if len(members) > 0 {
			fmt.Printf("  %-35s %d members\n", role.DisplayName, len(members))
			results = append(results, types.RoleWithMembers{
				Role:    role,
				Members: members,
			})
		}
	}

	ui.Success("Found %d roles with members", len(results))
	return results, nil
}
