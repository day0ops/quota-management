package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeJWT(claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, _ := json.Marshal(claims)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return strings.Join([]string{header, encoded, "sig"}, ".")
}

func TestParseJWT_OrgAdmin(t *testing.T) {
	token := makeJWT(map[string]interface{}{
		"sub":    "user-123",
		"email":  "admin@example.com",
		"org_id": "org-abc",
		"is_org": true,
	})
	id, err := ParseJWT(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", id.Subject)
	assert.Equal(t, "user-123", id.UserID)
	assert.True(t, id.IsOrg)
	assert.False(t, id.IsTeamAdmin)
}

func TestParseJWT_TeamAdmin(t *testing.T) {
	token := makeJWT(map[string]interface{}{
		"sub":           "user-456",
		"email":         "teamlead@example.com",
		"org_id":        "org-abc",
		"team_id":       "team-xyz",
		"is_team_admin": true,
	})
	id, err := ParseJWT(token)
	require.NoError(t, err)
	assert.Equal(t, "user-456", id.UserID)
	assert.True(t, id.IsTeamAdmin)
	assert.False(t, id.IsOrg)
}

func TestParseJWT_RegularUser(t *testing.T) {
	token := makeJWT(map[string]interface{}{
		"sub":   "user-789",
		"email": "alice@example.com",
	})
	id, err := ParseJWT(token)
	require.NoError(t, err)
	assert.Equal(t, "user-789", id.UserID)
	assert.False(t, id.IsOrg)
	assert.False(t, id.IsTeamAdmin)
}

func TestParseJWT_TeamAdminStringClaim(t *testing.T) {
	token := makeJWT(map[string]interface{}{
		"sub":           "user-101",
		"is_team_admin": "true",
	})
	id, err := ParseJWT(token)
	require.NoError(t, err)
	assert.True(t, id.IsTeamAdmin)
}
