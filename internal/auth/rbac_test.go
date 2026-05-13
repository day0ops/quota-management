package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func orgAdmin(orgID string) *Identity {
	return &Identity{Subject: "admin-1", OrgID: orgID, IsOrg: true}
}

func teamAdmin(orgID, teamID string) *Identity {
	return &Identity{Subject: "ta-1", OrgID: orgID, TeamID: teamID, IsTeamAdmin: true}
}

func regularUser(userID string) *Identity {
	return &Identity{Subject: userID, UserID: userID}
}

func TestCanViewUserBudget_OrgAdminCanViewAll(t *testing.T) {
	id := orgAdmin("org-1")
	assert.True(t, CanViewUserBudget(id, "alice", "team-1"))
}

func TestCanViewUserBudget_TeamAdminCanViewOwnTeam(t *testing.T) {
	id := teamAdmin("org-1", "team-1")
	assert.True(t, CanViewUserBudget(id, "alice", "team-1"))
	assert.False(t, CanViewUserBudget(id, "alice", "team-2"))
}

func TestCanViewUserBudget_UserCanViewOwnOnly(t *testing.T) {
	id := regularUser("alice")
	assert.True(t, CanViewUserBudget(id, "alice", ""))
	assert.False(t, CanViewUserBudget(id, "bob", ""))
}

func TestCanApproveOrReject_OrgAdmin(t *testing.T) {
	id := orgAdmin("org-1")
	assert.True(t, CanApproveOrReject(id, "org-1", "team-1"))
	assert.False(t, CanApproveOrReject(id, "org-2", "team-1"))
}

func TestCanApproveOrReject_TeamAdmin(t *testing.T) {
	id := teamAdmin("org-1", "team-1")
	assert.True(t, CanApproveOrReject(id, "org-1", "team-1"))
	assert.False(t, CanApproveOrReject(id, "org-1", "team-2"))
}

func TestCanApproveOrReject_RegularUser(t *testing.T) {
	id := regularUser("alice")
	assert.False(t, CanApproveOrReject(id, "org-1", "team-1"))
}
