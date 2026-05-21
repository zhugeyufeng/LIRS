package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"lirs/apps/server/internal/store"
)

func anyAdminRoles() []string {
	return []string{"material_admin", "finance_admin", "tenant_admin", "lab_admin", "super_admin"}
}

func tenantAdminRoles() []string {
	return []string{"tenant_admin", "lab_admin", "super_admin"}
}

func userReaderRoles() []string {
	return []string{"finance_admin", "tenant_admin", "lab_admin", "super_admin"}
}

func materialAdminRoles() []string {
	return []string{"material_admin", "tenant_admin", "lab_admin", "super_admin"}
}

func financeAdminRoles() []string {
	return []string{"finance_admin", "tenant_admin", "lab_admin", "super_admin"}
}

const (
	anonymousTenantID      = "00000000-0000-0000-0000-000000000000"
	currentUserContextKey  = "lirs.current_user"
	currentActorContextKey = "lirs.current_actor"
)

func bearerToken(c *gin.Context) (string, bool) {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return "", false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return "", false
	}
	return token, true
}

func tenantContextMiddleware(repo authRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(store.WithTenantContext(c.Request.Context(), store.TenantContext{TenantID: anonymousTenantID}))
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(header, "Bearer ") {
			token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			if token != "" {
				if user, err := repo.CurrentUser(c.Request.Context(), token); err == nil {
					rememberCurrentUser(c, user)
				}
			}
		}
		c.Next()
	}
}

func optionalCurrentUser(c *gin.Context, repo authRepository) (store.User, bool) {
	if user, ok := cachedCurrentUser(c); ok {
		return user, true
	}
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return store.User{}, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return store.User{}, false
	}
	user, err := repo.CurrentUser(c.Request.Context(), token)
	if err != nil {
		return store.User{}, false
	}
	rememberCurrentUser(c, user)
	return user, true
}

func requireAuthenticated(c *gin.Context, repo authRepository) (store.Actor, bool) {
	token, ok := bearerToken(c)
	if !ok {
		return store.Actor{}, false
	}
	if actor, ok := cachedCurrentActor(c); ok {
		return actor, true
	}
	user, err := repo.CurrentUser(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
		return store.Actor{}, false
	}
	return rememberCurrentUser(c, user), true
}

func cachedCurrentUser(c *gin.Context) (store.User, bool) {
	value, ok := c.Get(currentUserContextKey)
	if !ok {
		return store.User{}, false
	}
	user, ok := value.(store.User)
	return user, ok
}

func cachedCurrentActor(c *gin.Context) (store.Actor, bool) {
	value, ok := c.Get(currentActorContextKey)
	if !ok {
		return store.Actor{}, false
	}
	actor, ok := value.(store.Actor)
	return actor, ok
}

func rememberCurrentUser(c *gin.Context, user store.User) store.Actor {
	actor := actorFromUser(user)
	c.Set(currentUserContextKey, user)
	c.Set(currentActorContextKey, actor)
	c.Request = c.Request.WithContext(store.WithTenantContext(c.Request.Context(), store.TenantContext{
		TenantID:       user.TenantID,
		TenantName:     user.TenantName,
		FinanceEnabled: user.FinanceEnabled,
		AllTenants:     user.Role == "super_admin",
		Actor:          actor,
	}))
	return actor
}

func actorFromUser(user store.User) store.Actor {
	return store.Actor{
		UserID:         user.ID,
		TenantID:       user.TenantID,
		TenantName:     user.TenantName,
		Name:           user.Name,
		Email:          user.Email,
		Department:     user.Department,
		Role:           user.Role,
		Status:         user.Status,
		GroupName:      user.GroupName,
		EmailVerified:  user.EmailVerified,
		FinanceEnabled: user.FinanceEnabled,
		AuthEpoch:      user.AuthEpoch,
	}
}

func requireActiveUser(c *gin.Context, repo authRepository) (store.Actor, bool) {
	actor, ok := requireAuthenticated(c, repo)
	if !ok {
		return store.Actor{}, false
	}
	if actor.Status != "active" || actor.Role == "unassigned" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is not approved"})
		return store.Actor{}, false
	}
	if !actor.EmailVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "email is not verified"})
		return store.Actor{}, false
	}
	return actor, true
}

func requireAnyRole(c *gin.Context, repo authRepository, roles ...string) (store.Actor, bool) {
	actor, ok := requireActiveUser(c, repo)
	if !ok {
		return store.Actor{}, false
	}
	for _, allowed := range roles {
		if actor.Role == allowed {
			return actor, true
		}
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return store.Actor{}, false
}

func requireFinanceEnabled(c *gin.Context, actor store.Actor) bool {
	if actor.Role == "super_admin" || actor.FinanceEnabled {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "finance module is disabled for this tenant"})
	return false
}

func financeRequestContext(c *gin.Context, repo repository, actor store.Actor) (context.Context, bool) {
	if actor.Role != "super_admin" {
		if !requireFinanceEnabled(c, actor) {
			return nil, false
		}
		return c.Request.Context(), true
	}

	tenantID := strings.TrimSpace(c.Query("tenantId"))
	if tenantID == "" || tenantID == actor.TenantID {
		if !actor.FinanceEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "finance module is disabled for this tenant"})
			return nil, false
		}
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       actor.TenantID,
			TenantName:     actor.TenantName,
			FinanceEnabled: actor.FinanceEnabled,
		}), true
	}

	tenants, err := repo.Tenants(c.Request.Context())
	if err != nil {
		respond(c, nil, err)
		return nil, false
	}
	for _, tenant := range tenants {
		if tenant.ID != tenantID {
			continue
		}
		if tenant.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant is disabled"})
			return nil, false
		}
		if !tenant.FinanceEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "finance module is disabled for this tenant"})
			return nil, false
		}
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       tenant.ID,
			TenantName:     tenant.Name,
			FinanceEnabled: tenant.FinanceEnabled,
		}), true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "tenant not found"})
	return nil, false
}

func organizationUnitRequestContext(c *gin.Context, repo repository, actor store.Actor) (context.Context, bool) {
	tenantID := strings.TrimSpace(c.Query("tenantId"))
	if actor.Role != "super_admin" || tenantID == "" || tenantID == actor.TenantID {
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       actor.TenantID,
			TenantName:     actor.TenantName,
			FinanceEnabled: actor.FinanceEnabled,
		}), true
	}

	tenants, err := repo.Tenants(c.Request.Context())
	if err != nil {
		respond(c, nil, err)
		return nil, false
	}
	for _, tenant := range tenants {
		if tenant.ID != tenantID {
			continue
		}
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       tenant.ID,
			TenantName:     tenant.Name,
			FinanceEnabled: tenant.FinanceEnabled,
		}), true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "tenant not found"})
	return nil, false
}

func tenantAdminRequestContext(c *gin.Context, repo repository, actor store.Actor) (context.Context, bool) {
	tenantID := strings.TrimSpace(c.Query("tenantId"))
	if actor.Role != "super_admin" || tenantID == "" || tenantID == actor.TenantID {
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       actor.TenantID,
			TenantName:     actor.TenantName,
			FinanceEnabled: actor.FinanceEnabled,
			AllTenants:     false,
			Actor:          actor,
		}), true
	}

	tenants, err := repo.Tenants(c.Request.Context())
	if err != nil {
		respond(c, nil, err)
		return nil, false
	}
	for _, tenant := range tenants {
		if tenant.ID != tenantID {
			continue
		}
		if tenant.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant is disabled"})
			return nil, false
		}
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       tenant.ID,
			TenantName:     tenant.Name,
			FinanceEnabled: tenant.FinanceEnabled,
			AllTenants:     false,
			Actor:          actor,
		}), true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "tenant not found"})
	return nil, false
}

func materialWriteRequestContext(c *gin.Context, repo repository, actor store.Actor) (context.Context, bool) {
	if actor.Role != "super_admin" {
		return c.Request.Context(), true
	}
	ctx, ok := tenantAdminRequestContext(c, repo, actor)
	if ok {
		c.Request = c.Request.WithContext(ctx)
	}
	return ctx, ok
}
