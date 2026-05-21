package store

import (
	"context"
	"strings"
)

func (r *Repository) MaterialAlertActions(ctx context.Context) ([]MaterialAlertAction, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT maa.id::text, maa.material_id::text, m.name, maa.alert_type, maa.action, maa.comment, maa.actor, maa.created_at
FROM material_alert_actions maa
JOIN materials m ON m.id = maa.material_id
WHERE ($1::boolean OR maa.tenant_id = $2::uuid)
ORDER BY maa.created_at DESC
LIMIT 100
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialAlertAction, 0)
	for rows.Next() {
		var item MaterialAlertAction
		if err := rows.Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.AlertType, &item.Action, &item.Comment, &item.Actor, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateMaterialAlertAction(ctx context.Context, materialID string, input MaterialAlertActionInput) (MaterialAlertAction, error) {
	tenant := TenantFromContext(ctx)
	materialID = strings.TrimSpace(materialID)
	input.AlertType = strings.TrimSpace(input.AlertType)
	input.Action = strings.TrimSpace(input.Action)
	input.Comment = strings.TrimSpace(input.Comment)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.AlertType == "" || (input.Action != "handled" && input.Action != "ignored") {
		return MaterialAlertAction{}, clientError("invalid material alert action input")
	}
	var item MaterialAlertAction
	err := r.db.QueryRow(ctx, `
INSERT INTO material_alert_actions (tenant_id, material_id, alert_type, action, comment, actor)
SELECT m.tenant_id, m.id, $2, $3, $4, $5
FROM materials m
WHERE m.id = $1 AND ($6::boolean OR m.tenant_id = $7::uuid)
RETURNING id::text, material_id::text, (SELECT name FROM materials WHERE id = material_id),
          alert_type, action, comment, actor, created_at
`, materialID, input.AlertType, input.Action, input.Comment, input.Actor, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.AlertType, &item.Action, &item.Comment, &item.Actor, &item.CreatedAt)
	if err != nil {
		return MaterialAlertAction{}, err
	}
	r.audit(ctx, input.Actor, "material_alert."+input.Action, "material", item.MaterialID, item.AlertType, item.Comment)
	return item, nil
}

func (r *Repository) MaterialAnalytics(ctx context.Context) (MaterialAnalytics, error) {
	materials, err := r.Materials(ctx)
	if err != nil {
		return MaterialAnalytics{}, err
	}
	requests, err := r.MaterialRequests(ctx)
	if err != nil {
		return MaterialAnalytics{}, err
	}
	damages, err := r.MaterialDamages(ctx)
	if err != nil {
		return MaterialAnalytics{}, err
	}
	alertActions, err := r.MaterialAlertActions(ctx)
	if err != nil {
		return MaterialAnalytics{}, err
	}
	now := appNow()
	today := appDateStringAt(now)
	result := MaterialAnalytics{
		ProductTotal:         len(materials),
		MonthlyConsumption:   make([]MaterialConsumptionPoint, 0, 12),
		TopConsumedMaterials: make([]MaterialConsumptionRanking, 0),
		DamageByReason:       make([]MaterialDamageReasonStat, 0),
		ProductTypeBreakdown: make([]MaterialBreakdown, 0),
		CategoryBreakdown:    make([]MaterialBreakdown, 0),
		LatestAlertActions:   alertActions,
	}
	productBreakdown := make(map[string]MaterialBreakdown)
	categoryBreakdown := make(map[string]MaterialBreakdown)
	for _, item := range materials {
		result.StockTotal += item.Stock
		if item.ProductType == "standard" {
			result.StandardTotal++
		}
		switch item.Status {
		case "near_expiry":
			result.NearExpiryTotal++
		case "expired":
			result.ExpiredTotal++
		case "low":
			result.LowStockTotal++
		case "damaged":
			result.DamagedTotal += item.DamagedQuantity
		}
		product := productBreakdown[item.ProductType]
		product.Label = item.ProductType
		product.Count++
		product.Stock += item.Stock
		productBreakdown[item.ProductType] = product
		category := categoryBreakdown[item.Category]
		category.Label = item.Category
		category.Count++
		category.Stock += item.Stock
		categoryBreakdown[item.Category] = category
	}
	monthly := make(map[string]int)
	consumption := make(map[string]MaterialConsumptionRanking)
	for i := 11; i >= 0; i-- {
		month := now.AddDate(0, -i, 0).Format("2006-01")
		monthly[month] = 0
	}
	for _, item := range requests {
		if item.Status != "outbound" {
			continue
		}
		createdDate := appDateStringAt(item.CreatedAt)
		if createdDate == today {
			result.TodayUsageTotal += item.Quantity
		}
		month := item.CreatedAt.In(appLocation).Format("2006-01")
		if _, ok := monthly[month]; ok {
			monthly[month] += item.Quantity
		}
		ranking := consumption[item.MaterialID]
		ranking.MaterialID = item.MaterialID
		ranking.MaterialName = item.MaterialName
		ranking.Quantity += item.Quantity
		consumption[item.MaterialID] = ranking
	}
	for i := 11; i >= 0; i-- {
		month := now.AddDate(0, -i, 0).Format("2006-01")
		result.MonthlyConsumption = append(result.MonthlyConsumption, MaterialConsumptionPoint{Month: month, Quantity: monthly[month]})
	}
	for _, item := range topMaterialConsumption(consumption, 8) {
		result.TopConsumedMaterials = append(result.TopConsumedMaterials, item)
	}
	damageReasons := make(map[string]int)
	for _, item := range damages {
		if item.Status != "processed" {
			continue
		}
		result.DamagedTotal += item.Quantity
		reason := item.Reason
		if reason == "" {
			reason = "未填写原因"
		}
		damageReasons[reason] += item.Quantity
	}
	for reason, quantity := range damageReasons {
		result.DamageByReason = append(result.DamageByReason, MaterialDamageReasonStat{Reason: reason, Quantity: quantity})
	}
	for _, item := range productBreakdown {
		result.ProductTypeBreakdown = append(result.ProductTypeBreakdown, item)
	}
	for _, item := range categoryBreakdown {
		result.CategoryBreakdown = append(result.CategoryBreakdown, item)
	}
	return result, nil
}
