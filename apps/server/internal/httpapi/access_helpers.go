package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lirs/apps/server/internal/store"
)

func isAdmin(actor store.Actor) bool {
	return actor.Role == "tenant_admin" || actor.Role == "lab_admin" || actor.Role == "super_admin"
}

func canManageMaterials(actor store.Actor) bool {
	return actor.Role == "material_admin" || isAdmin(actor)
}

func canManageFinance(actor store.Actor) bool {
	return actor.Role == "finance_admin" || isAdmin(actor)
}

func canManageTraining(actor store.Actor) bool {
	return actor.Role == "material_admin" || isAdmin(actor)
}

func canAccessReservation(actor store.Actor, item store.Reservation) bool {
	if isAdmin(actor) {
		return true
	}
	if actor.Role == "group_leader" && actor.GroupName != "" && actor.GroupName == item.GroupName {
		return true
	}
	return item.UserID != "" && item.UserID == actor.UserID
}

func canReviewGroup(actor store.Actor, groupName string) bool {
	if isAdmin(actor) {
		return true
	}
	return actor.Role == "group_leader" && actor.GroupName != "" && actor.GroupName == groupName
}

func canReviewMaterialGroup(actor store.Actor, groupName string) bool {
	if canManageMaterials(actor) {
		return true
	}
	return actor.Role == "group_leader" && actor.GroupName != "" && actor.GroupName == groupName
}

func filterReservationsForActor(actor store.Actor, items []store.Reservation) []store.Reservation {
	return filterItems(items, func(item store.Reservation) bool {
		return canAccessReservation(actor, item)
	})
}

func filterLedgerForActor(actor store.Actor, items []store.LedgerEntry) []store.LedgerEntry {
	if canManageFinance(actor) {
		return items
	}
	return filterItems(items, func(item store.LedgerEntry) bool {
		return item.UserID == actor.UserID
	})
}

func filterFinancialAccountsForActor(actor store.Actor, items []store.FinancialAccount) []store.FinancialAccount {
	if canManageFinance(actor) {
		return items
	}
	return filterItems(items, func(item store.FinancialAccount) bool {
		return item.UserID == actor.UserID
	})
}

func filterMaterialRequestsForActor(actor store.Actor, items []store.MaterialRequest) []store.MaterialRequest {
	if canManageMaterials(actor) {
		return items
	}
	return filterItems(items, func(item store.MaterialRequest) bool {
		return canAccessMaterialActorRow(actor, item.GroupName, item.RequesterID)
	})
}

func filterMaterialRequestExportRowsForActor(actor store.Actor, items []store.MaterialRequestExportRow) []store.MaterialRequestExportRow {
	if canManageMaterials(actor) {
		return items
	}
	return filterItems(items, func(item store.MaterialRequestExportRow) bool {
		return canAccessMaterialActorRow(actor, item.GroupName, item.RequesterID)
	})
}

func filterMaterialPurchasesForActor(actor store.Actor, items []store.MaterialPurchase) []store.MaterialPurchase {
	if canManageMaterials(actor) {
		return items
	}
	return filterItems(items, func(item store.MaterialPurchase) bool {
		return canAccessMaterialActorRow(actor, item.GroupName, item.RequesterID)
	})
}

func filterMaterialDamagesForActor(actor store.Actor, items []store.MaterialDamage) []store.MaterialDamage {
	if canManageMaterials(actor) {
		return items
	}
	return filterItems(items, func(item store.MaterialDamage) bool {
		return canAccessMaterialActorRow(actor, item.GroupName, item.ReporterID)
	})
}

func canAccessNotification(actor store.Actor, item store.Notification) bool {
	switch item.TargetScope {
	case "", "global":
		return true
	case "personal":
		return item.UserID != "" && item.UserID == actor.UserID
	case "group":
		return item.GroupName != "" && item.GroupName == actor.GroupName
	case "department":
		return item.Department != "" && item.Department == actor.Department
	default:
		return isAdmin(actor)
	}
}

func filterNotificationsForActor(actor store.Actor, items []store.Notification) []store.Notification {
	return filterItems(items, func(item store.Notification) bool {
		return canAccessNotification(actor, item)
	})
}

func filterItems[T any](items []T, keep func(T) bool) []T {
	filtered := make([]T, 0, len(items))
	for _, item := range items {
		if keep(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func canAccessMaterialActorRow(actor store.Actor, groupName string, ownerID string) bool {
	if actor.Role == "group_leader" && actor.GroupName != "" && groupName == actor.GroupName {
		return true
	}
	return ownerID != "" && ownerID == actor.UserID
}

func authorizeReservationReview(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findReservation(c, repo, id)
	if !ok {
		return false
	}
	if canReviewGroup(actor, item.GroupName) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeReservationOwnerOrAdmin(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findReservation(c, repo, id)
	if !ok {
		return false
	}
	if isAdmin(actor) || item.UserID == actor.UserID {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeReservationCancel(c *gin.Context, repo repository, actor store.Actor, id string) (store.Reservation, bool) {
	item, ok := findReservation(c, repo, id)
	if !ok {
		return store.Reservation{}, false
	}
	if isAdmin(actor) || item.UserID == actor.UserID || canReviewGroup(actor, item.GroupName) {
		return item, true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return store.Reservation{}, false
}

func authorizeMaterialRequestReview(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialRequest(c, repo, id)
	if !ok {
		return false
	}
	if canReviewMaterialGroup(actor, item.GroupName) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeMaterialRequestOwnerOrAdmin(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialRequest(c, repo, id)
	if !ok {
		return false
	}
	if canManageMaterials(actor) || item.RequesterID == actor.UserID {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeMaterialPurchaseReview(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialPurchase(c, repo, id)
	if !ok {
		return false
	}
	if canReviewMaterialGroup(actor, item.GroupName) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeMaterialPurchaseOwnerOrAdmin(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialPurchase(c, repo, id)
	if !ok {
		return false
	}
	if canManageMaterials(actor) || item.RequesterID == actor.UserID {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeMaterialDamageReview(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialDamage(c, repo, id)
	if !ok {
		return false
	}
	if canReviewMaterialGroup(actor, item.GroupName) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeMaterialDamageOwnerOrAdmin(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialDamage(c, repo, id)
	if !ok {
		return false
	}
	if canManageMaterials(actor) || item.ReporterID == actor.UserID {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeNotificationAccess(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	items, err := repo.Notifications(c.Request.Context(), actor)
	if err != nil {
		respond(c, nil, err)
		return false
	}
	for _, item := range items {
		if item.ID == id {
			if canAccessNotification(actor, item) {
				return true
			}
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return false
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
	return false
}

func findReservation(c *gin.Context, repo repository, id string) (store.Reservation, bool) {
	item, err := repo.Reservation(c.Request.Context(), id)
	if err != nil {
		respond(c, nil, err)
		return store.Reservation{}, false
	}
	return item, true
}

func findMaterialRequest(c *gin.Context, repo repository, id string) (store.MaterialRequest, bool) {
	item, err := repo.MaterialRequest(c.Request.Context(), id)
	if err != nil {
		respond(c, nil, err)
		return store.MaterialRequest{}, false
	}
	return item, true
}

func findMaterialPurchase(c *gin.Context, repo repository, id string) (store.MaterialPurchase, bool) {
	item, err := repo.MaterialPurchase(c.Request.Context(), id)
	if err != nil {
		respond(c, nil, err)
		return store.MaterialPurchase{}, false
	}
	return item, true
}

func findMaterialDamage(c *gin.Context, repo repository, id string) (store.MaterialDamage, bool) {
	item, err := repo.MaterialDamage(c.Request.Context(), id)
	if err != nil {
		respond(c, nil, err)
		return store.MaterialDamage{}, false
	}
	return item, true
}
