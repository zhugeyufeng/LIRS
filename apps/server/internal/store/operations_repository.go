package store

import (
	"context"
	"time"
)

func (r *Repository) AuditEvents(ctx context.Context) ([]AuditEvent, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, actor, action, target_type, target_id, old_value, new_value, created_at
FROM audit_events
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY created_at DESC
LIMIT 100
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AuditEvent, 0)
	for rows.Next() {
		var item AuditEvent
		if err := rows.Scan(&item.ID, &item.Actor, &item.Action, &item.TargetType, &item.TargetID, &item.OldValue, &item.NewValue, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Operations(ctx context.Context) (Operations, error) {
	tenant := TenantFromContext(ctx)
	dashboard, err := r.Dashboard(ctx)
	if err != nil {
		return Operations{}, err
	}
	ops := Operations{
		Dashboard: dashboard,
		UpdatedAt: time.Now().UTC(),
		Alerts:    make([]OperationAlert, 0),
	}
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM reservations WHERE status = 'in_use' AND ($1::boolean OR tenant_id = $2::uuid)`, tenant.AllTenants, tenant.TenantID).Scan(&ops.InUseInstruments); err != nil {
		return Operations{}, err
	}

	trendRows, err := r.db.Query(ctx, `
SELECT to_char(hour_bucket, 'HH24:MI'), count(r.id)::int
FROM generate_series(date_trunc('hour', now()) - interval '23 hours', date_trunc('hour', now()), interval '1 hour') AS hour_bucket
LEFT JOIN reservations r ON lower(r.period) >= hour_bucket AND lower(r.period) < hour_bucket + interval '1 hour'
  AND ($1::boolean OR r.tenant_id = $2::uuid)
GROUP BY hour_bucket
ORDER BY hour_bucket
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return Operations{}, err
	}
	for trendRows.Next() {
		var point TrendPoint
		if err := trendRows.Scan(&point.Hour, &point.Count); err != nil {
			trendRows.Close()
			return Operations{}, err
		}
		ops.ReservationTrend = append(ops.ReservationTrend, point)
	}
	trendRows.Close()
	if err := trendRows.Err(); err != nil {
		return Operations{}, err
	}

	loadRows, err := r.db.Query(ctx, `
SELECT i.name, COALESCE(sum(EXTRACT(EPOCH FROM (upper(r.period) - lower(r.period))) / 3600), 0)::float8 AS hours
FROM instruments i
LEFT JOIN reservations r ON r.instrument_id = i.id AND r.status IN ('approved', 'in_use', 'completed')
WHERE ($1::boolean OR i.tenant_id = $2::uuid)
GROUP BY i.name
ORDER BY hours DESC, i.name
LIMIT 8
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return Operations{}, err
	}
	for loadRows.Next() {
		var item InstrumentLoad
		if err := loadRows.Scan(&item.InstrumentName, &item.Hours); err != nil {
			loadRows.Close()
			return Operations{}, err
		}
		ops.InstrumentLoads = append(ops.InstrumentLoads, item)
	}
	loadRows.Close()
	if err := loadRows.Err(); err != nil {
		return Operations{}, err
	}

	var reservationApprovalHours float64
	if err := r.db.QueryRow(ctx, `
SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (decided_at - created_at))) / 3600, 0)::float8
FROM reservations
WHERE decided_at IS NOT NULL
  AND ($1::boolean OR tenant_id = $2::uuid)
`, tenant.AllTenants, tenant.TenantID).Scan(&reservationApprovalHours); err != nil {
		return Operations{}, err
	}
	var materialApprovalHours float64
	if err := r.db.QueryRow(ctx, `
SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (decided_at - created_at))) / 3600, 0)::float8
FROM material_requests
WHERE decided_at IS NOT NULL
  AND ($1::boolean OR tenant_id = $2::uuid)
`, tenant.AllTenants, tenant.TenantID).Scan(&materialApprovalHours); err != nil {
		return Operations{}, err
	}
	var maintenanceResponseHours float64
	if err := r.db.QueryRow(ctx, `
SELECT COALESCE(AVG(GREATEST(EXTRACT(EPOCH FROM (lower(period) - created_at)), 0)) / 3600, 0)::float8
FROM maintenance_orders
WHERE ($1::boolean OR tenant_id = $2::uuid)
`, tenant.AllTenants, tenant.TenantID).Scan(&maintenanceResponseHours); err != nil {
		return Operations{}, err
	}
	ops.ApprovalEfficiency = []ApprovalMetric{
		{Label: "预约审批平均处理", Hours: reservationApprovalHours},
		{Label: "耗材审批平均处理", Hours: materialApprovalHours},
		{Label: "维护响应平均处理", Hours: maintenanceResponseHours},
	}
	alertRows, err := r.db.Query(ctx, `
SELECT '仪器' AS source, 'warning' AS level, name || ' 当前维护中' AS body
FROM instruments
WHERE status = 'maintenance'
  AND ($1::boolean OR tenant_id = $2::uuid)
UNION ALL
SELECT '耗材', 'warning', name || ' 库存低于预警线'
FROM materials
WHERE stock <= warning_line
  AND ($1::boolean OR tenant_id = $2::uuid)
LIMIT 10
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return Operations{}, err
	}
	for alertRows.Next() {
		var alert OperationAlert
		if err := alertRows.Scan(&alert.Source, &alert.Level, &alert.Body); err != nil {
			alertRows.Close()
			return Operations{}, err
		}
		ops.Alerts = append(ops.Alerts, alert)
	}
	alertRows.Close()
	if err := alertRows.Err(); err != nil {
		return Operations{}, err
	}
	ops.AlertCount = len(ops.Alerts)
	return ops, nil
}
