package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

type contextKey string

const IdentityKey contextKey = "identity"

// Identity represents the authenticated user extracted from the JWT.
type Identity struct {
	Subject     string `json:"sub"`
	Email       string `json:"email"`
	OrgID       string `json:"org_id"`
	TeamID      string `json:"team_id"`
	IsOrg       bool   `json:"-"`
	IsTeamAdmin bool   `json:"-"`
	UserID      string `json:"-"` // same as Subject, explicit for RBAC clarity
}

// AuthMiddleware extracts and validates the JWT from the request header.
// The JWT is passed by ext-auth in the "jwt" header after OIDC authentication.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwtHeader := r.Header.Get("jwt")
		if jwtHeader == "" {
			log.Debug().Str("path", r.URL.Path).Msg("no jwt header found")
			http.Error(w, `{"error":{"message":"Unauthorized"}}`, http.StatusUnauthorized)
			return
		}

		identity, err := ParseJWT(jwtHeader)
		if err != nil {
			log.Warn().Err(err).Msg("failed to parse jwt")
			http.Error(w, `{"error":{"message":"Invalid token"}}`, http.StatusUnauthorized)
			return
		}

		log.Debug().
			Str("sub", identity.Subject).
			Str("email", identity.Email).
			Str("org_id", identity.OrgID).
			Str("team_id", identity.TeamID).
			Bool("is_org", identity.IsOrg).
			Msg("authenticated request")

		ctx := context.WithValue(r.Context(), IdentityKey, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthMiddleware extracts the JWT if present but does not require it.
// Useful for endpoints that can work with or without authentication.
func OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwtHeader := r.Header.Get("jwt")
		if jwtHeader == "" {
			log.Debug().
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Msg("no jwt header found in request")
			next.ServeHTTP(w, r)
			return
		}

		identity, err := ParseJWT(jwtHeader)
		if err != nil {
			log.Debug().Err(err).Msg("failed to parse optional jwt, continuing without auth")
			next.ServeHTTP(w, r)
			return
		}

		log.Debug().
			Str("path", r.URL.Path).
			Str("sub", identity.Subject).
			Str("org_id", identity.OrgID).
			Str("team_id", identity.TeamID).
			Bool("is_org", identity.IsOrg).
			Msg("identity extracted from jwt")

		ctx := context.WithValue(r.Context(), IdentityKey, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetIdentity retrieves the Identity from the request context.
func GetIdentity(ctx context.Context) *Identity {
	if id, ok := ctx.Value(IdentityKey).(*Identity); ok {
		return id
	}
	return nil
}

// ParseJWT parses the JWT and extracts the identity claims.
// Note: Signature verification is handled by ext-auth/Keycloak before the request
// reaches this service. We only need to decode the payload.
func ParseJWT(tokenString string) (*Identity, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	identity := &Identity{}

	if sub, ok := claims["sub"].(string); ok {
		identity.Subject = sub
	}
	if email, ok := claims["email"].(string); ok {
		identity.Email = email
	}
	if orgID, ok := claims["org_id"].(string); ok {
		identity.OrgID = orgID
	}
	if teamID, ok := claims["team_id"].(string); ok {
		identity.TeamID = teamID
	}

	// Parse is_org as a separate claim (can be bool or string "true"/"false")
	if isOrg, ok := claims["is_org"].(bool); ok {
		identity.IsOrg = isOrg
	} else if isOrgStr, ok := claims["is_org"].(string); ok {
		identity.IsOrg = isOrgStr == "true"
	}

	// Parse is_team_admin claim
	if isTeamAdmin, ok := claims["is_team_admin"].(bool); ok {
		identity.IsTeamAdmin = isTeamAdmin
	} else if isTeamAdminStr, ok := claims["is_team_admin"].(string); ok {
		identity.IsTeamAdmin = isTeamAdminStr == "true"
	}

	// UserID mirrors Subject for explicit RBAC use
	identity.UserID = identity.Subject

	return identity, nil
}
