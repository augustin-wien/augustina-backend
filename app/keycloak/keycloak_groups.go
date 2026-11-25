package keycloak

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Nerzal/gocloak/v13"
)

// GetUserGroups function returns the user groups
func (k *Keycloak) GetUserGroups(userID string) ([]*gocloak.Group, error) {
	k.checkAdminToken()
	return k.Client.GetUserGroups(k.Context, k.adminAccessToken(), k.Realm, userID, gocloak.GetGroupsParams{})
}

// Assign group to user
func (k *Keycloak) AssignGroup(userID string, groupName string) error {
	if k.Client == nil {
		return fmt.Errorf("AssignGroup: keycloak client not initialized")
	}
	if userID == "" {
		return fmt.Errorf("AssignGroup: userID is empty")
	}
	if groupName == "" {
		return fmt.Errorf("AssignGroup: groupName is empty")
	}
	k.checkAdminToken()
	// Groups can only be searched by group paths and not by group names. Group paths have to start with / and if it's not there, we add it.
	if groupName[0] != '/' {
		groupName = "/" + groupName
	}
	group, err := k.GetGroupByPath(groupName)
	if err != nil {
		log.Errorf("AssignGroup: Error getting group by path %s", groupName)
		return err
	}
	log.Infof("AssignGroup: Assigning user to group %s %s %s", userID, *group.ID, groupName)
	return k.Client.AddUserToGroup(k.Context, k.adminAccessToken(), k.Realm, userID, *group.ID)
}

func (k *Keycloak) AssignDigitalLicenseGroup(userID string, licenseGroup string) error {
	if k.Client == nil {
		return fmt.Errorf("AssignDigitalLicenseGroup: keycloak client not initialized")
	}
	if userID == "" {
		return fmt.Errorf("AssignDigitalLicenseGroup: userID is empty")
	}
	k.checkAdminToken()
	licenseGroupPath := "/" + k.CustomerGroup + "/" + k.NewspaperGroup + "/" + licenseGroup
	// Check if group exists
	_, err := k.GetGroupByPath(licenseGroupPath)
	if err != nil {
		// Create group
		parentGroup, err := k.GetGroupByPath("/" + k.CustomerGroup + "/" + k.NewspaperGroup)
		if err != nil {
			log.Errorf("AssignDigitalLicenseGroup: Error getting group by path %s for %s", "/"+k.CustomerGroup+"/"+k.NewspaperGroup, k.NewspaperGroup)
			return err
		}
		_, err = k.CreateSubGroup(licenseGroup, *parentGroup.ID)
		if err != nil {
			log.Errorf("AssignDigitalLicenseGroup: Error creating group %s", licenseGroup)
			return err
		}
		// Check if group exists
		_, err = k.GetGroupByPath(licenseGroupPath)
		if err != nil {
			log.Errorf("AssignDigitalLicenseGroup: Error getting group by path %s", licenseGroupPath)
			return err
		}
	}
	// Assign user to group
	return k.AssignGroup(userID, licenseGroupPath)
}

// SyncLicenseGroupsDiffToKeycloak applies the diff between oldGroups and newGroups to
// Keycloak: groups added by the admin are assigned, groups explicitly removed by the
// admin are unassigned. Groups that were never in our DB (from other applications) are
// never touched.
func (k *Keycloak) SyncLicenseGroupsDiffToKeycloak(userID string, oldGroups, newGroups []string) error {
	oldSet := make(map[string]bool, len(oldGroups))
	for _, g := range oldGroups {
		if g != "" {
			oldSet[g] = true
		}
	}
	newSet := make(map[string]bool, len(newGroups))
	for _, g := range newGroups {
		if g != "" {
			newSet[g] = true
		}
	}

	for g := range newSet {
		if !oldSet[g] {
			if err := k.AssignDigitalLicenseGroup(userID, g); err != nil {
				log.Errorf("SyncLicenseGroupsDiffToKeycloak: add %s to %s: %v", g, userID, err)
			}
		}
	}
	for g := range oldSet {
		if !newSet[g] {
			licenseGroupPath := "/" + k.CustomerGroup + "/" + k.NewspaperGroup + "/" + g
			group, err := k.GetGroupByPath(licenseGroupPath)
			if err != nil {
				continue // group doesn't exist in Keycloak, nothing to remove
			}
			k.checkAdminToken()
			if err := k.Client.DeleteUserFromGroup(k.Context, k.adminAccessToken(), k.Realm, userID, *group.ID); err != nil {
				log.Errorf("SyncLicenseGroupsDiffToKeycloak: remove %s from %s: %v", g, userID, err)
			}
		}
	}
	return nil
}

func (k *Keycloak) CreateGroup(groupName string) (string, error) {
	k.checkAdminToken()
	group := gocloak.Group{
		Name: &groupName,
	}
	id, err := k.Client.CreateGroup(k.Context, k.adminAccessToken(), k.Realm, group)
	return id, err
}

func (k *Keycloak) CreateSubGroup(groupName string, parentGroupID string) (string, error) {
	k.checkAdminToken()
	group := gocloak.Group{
		Name: &groupName,
	}
	id, err := k.Client.CreateChildGroup(k.Context, k.adminAccessToken(), k.Realm, parentGroupID, group)
	return id, err
}

// EnsureGroupPath ensures that a group path (e.g. "/customers/newspapers/edition") exists.
// It will create any missing top-level groups or subgroups along the path.
// Each level is looked up by its path: Keycloak no longer lists subgroups in a
// group's representation, so they can't be found through the parent.
func (k *Keycloak) EnsureGroupPath(path string) error {
	k.checkAdminToken()
	// Normalize path
	p := "/" + strings.Trim(path, "/")
	if p == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")

	var currentParentID string
	for i, part := range parts {
		currentPath := "/" + strings.Join(parts[:i+1], "/")
		id, err := k.groupIDByPath(currentPath)
		if err != nil {
			return fmt.Errorf("EnsureGroupPath: failed to look up %s: %w", currentPath, err)
		}
		if id == "" {
			if i == 0 {
				id, err = k.CreateGroup(part)
			} else {
				id, err = k.CreateSubGroup(part, currentParentID)
			}
			if err != nil {
				// Created concurrently
				if !strings.Contains(err.Error(), "409") && !strings.Contains(err.Error(), "Conflict") {
					return fmt.Errorf("EnsureGroupPath: failed to create %s: %w", currentPath, err)
				}
				id, err = k.groupIDByPath(currentPath)
				if err != nil || id == "" {
					return fmt.Errorf("EnsureGroupPath: failed to find %s after conflict: %v", currentPath, err)
				}
			}
		}
		currentParentID = id
	}
	return nil
}

// groupIDByPath returns the ID of the group at path, or "" when there is none
func (k *Keycloak) groupIDByPath(path string) (string, error) {
	// gocloak puts a "/" before the path; a second one is rejected as not normalized
	group, err := k.Client.GetGroupByPath(k.Context, k.adminAccessToken(), k.Realm, strings.TrimPrefix(path, "/"))
	if err != nil {
		var apiErr *gocloak.APIError
		if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound {
			return "", nil
		}
		return "", err
	}
	if group == nil || group.ID == nil {
		return "", nil
	}
	return *group.ID, nil
}

// GetGroupByPath function returns the group of the given name
func (k *Keycloak) GetGroupByPath(path string) (*gocloak.Group, error) {
	k.checkAdminToken()
	// Re-implement manual traversal to avoid 400 normalization errors
	p := "/" + strings.Trim(path, "/")
	if p == "/" {
		return nil, fmt.Errorf("root path not supported")
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")

	var currentGroup *gocloak.Group

	for i, part := range parts {
		if i == 0 {
			params := gocloak.GetGroupsParams{Search: &part}
			foundGroups, err := k.Client.GetGroups(k.Context, k.adminAccessToken(), k.Realm, params)
			if err != nil {
				return nil, err
			}
			for _, g := range foundGroups {
				if g.Name != nil && *g.Name == part {
					currentGroup = g
					break
				}
			}
			if currentGroup == nil {
				return nil, fmt.Errorf("group not found: %s", part)
			}
		} else {
			// Retry loop for subgroups (consistency)
			var found *gocloak.Group
			for attempt := 0; attempt < 20; attempt++ {
				// Refresh current group to get subgroups
				fullGroup, err := k.Client.GetGroup(k.Context, k.adminAccessToken(), k.Realm, *currentGroup.ID)
				if err != nil {
					return nil, err
				}

				if fullGroup.SubGroups != nil {
					for _, sub := range *fullGroup.SubGroups {
						if sub.Name != nil && *sub.Name == part {
							// copy to avoid iterator reuse issues
							s := sub
							found = &s
							break
						}
					}
				}
				if found != nil {
					break
				}

				// Fallback: Search globally for the group and check path
				searchParams := gocloak.GetGroupsParams{
					Search: &part,
				}
				searchResults, err := k.Client.GetGroups(k.Context, k.adminAccessToken(), k.Realm, searchParams)
				if err == nil {
					expectedPath := "/" + strings.Join(parts[:i+1], "/")
					for _, g := range searchResults {
						if g.Path != nil && *g.Path == expectedPath {
							found = g
							break
						}
						// Search matches might return the hierarchy (parent -> child)
						// Iterate deeply to find matching path
						found = k.findGroupInHierarchy(g, expectedPath)
						if found != nil {
							break
						}
					}
				}
				if found != nil {
					break
				}

				// Wait before retry
				time.Sleep(500 * time.Millisecond)
			}

			if found == nil {
				return nil, fmt.Errorf("subgroup %s not found in %s after retries", part, *currentGroup.Name)
			}
			currentGroup = found
		}
	}
	return currentGroup, nil
}

func (k *Keycloak) findGroupInHierarchy(g *gocloak.Group, expectedPath string) *gocloak.Group {
	if g.Path != nil && *g.Path == expectedPath {
		return g
	}
	if g.SubGroups != nil {
		for i := range *g.SubGroups {
			sub := &(*g.SubGroups)[i]
			found := k.findGroupInHierarchy(sub, expectedPath)
			if found != nil {
				return found
			}
		}
	}
	return nil
}

// GetGroup function returns the group of the given name
func (k *Keycloak) GetGroup(name string) (*gocloak.Group, error) {
	k.checkAdminToken()
	// First try using GetGroups with search which is safer against path normalization issues
	params := gocloak.GetGroupsParams{
		Search: &name,
	}
	groups, err := k.Client.GetGroups(k.Context, k.adminAccessToken(), k.Realm, params)
	if err == nil {
		for _, g := range groups {
			if g.Name != nil && *g.Name == name {
				return g, nil
			}
		}
	}
	// Fallback to path
	return k.Client.GetGroupByPath(k.Context, k.adminAccessToken(), k.Realm, "/"+name)
}

// DeleteGroup function deletes a group given by name
func (k *Keycloak) DeleteGroup(name string) error {
	k.checkAdminToken()
	group, err := k.GetGroup(name)
	if err != nil {
		return err
	}
	return k.Client.DeleteGroup(k.Context, k.adminAccessToken(), k.Realm, *group.ID)
}

func (k *Keycloak) DeleteSubGroupByPath(path string) error {
	k.checkAdminToken()
	id, err := k.groupIDByPath("/" + strings.Trim(path, "/"))
	if err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("group not found: %s", path)
	}
	return k.Client.DeleteGroup(k.Context, k.adminAccessToken(), k.Realm, id)
}
