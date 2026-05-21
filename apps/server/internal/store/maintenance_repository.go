package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) MaintenanceOrders(ctx context.Context) ([]MaintenanceOrder, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT mo.id::text, COALESCE(mo.instrument_id::text, ''), COALESCE(i.name, '已删除仪器'), mo.type, mo.priority, mo.status, mo.handler,
       mo.description, mo.result, lower(mo.period), upper(mo.period), mo.created_at
FROM maintenance_orders mo
LEFT JOIN instruments i ON i.id = mo.instrument_id
WHERE ($1::boolean OR mo.tenant_id = $2::uuid)
ORDER BY mo.created_at DESC
LIMIT 100
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaintenanceOrder, 0)
	for rows.Next() {
		var item MaintenanceOrder
		if err := rows.Scan(&item.ID, &item.InstrumentID, &item.InstrumentName, &item.Type, &item.Priority, &item.Status, &item.Handler, &item.Description, &item.Result, &item.StartTime, &item.EndTime, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateMaintenanceOrder(ctx context.Context, input MaintenanceInput) (MaintenanceOrder, error) {
	tenant := TenantFromContext(ctx)
	input.Type = strings.TrimSpace(input.Type)
	input.Priority = strings.TrimSpace(input.Priority)
	input.Handler = strings.TrimSpace(input.Handler)
	input.Description = strings.TrimSpace(input.Description)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Type == "" {
		input.Type = "routine"
	}
	if input.Priority == "" {
		input.Priority = "normal"
	}
	if input.InstrumentID == "" || input.Description == "" || !input.EndTime.After(input.StartTime) {
		return MaintenanceOrder{}, clientError("invalid maintenance input")
	}
	status := "assigned"
	if input.Handler == "" {
		status = "reported"
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaintenanceOrder{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var targetTenantID string
	var instrumentName string
	if err := tx.QueryRow(ctx, `
SELECT tenant_id::text, name
FROM instruments
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.InstrumentID, tenant.AllTenants, tenant.TenantID).Scan(&targetTenantID, &instrumentName); err != nil {
		return MaintenanceOrder{}, err
	}

	var inUseCount int
	if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM reservations
WHERE instrument_id = $1 AND status = 'in_use' AND period && tstzrange($2, $3, '[)')
  AND tenant_id = $4::uuid
`, input.InstrumentID, input.StartTime, input.EndTime, targetTenantID).Scan(&inUseCount); err != nil {
		return MaintenanceOrder{}, err
	}
	if inUseCount > 0 && input.Type != "emergency" {
		return MaintenanceOrder{}, clientError("maintenance conflicts with an in-use reservation")
	}

	var item MaintenanceOrder
	err = tx.QueryRow(ctx, `
INSERT INTO maintenance_orders (tenant_id, instrument_id, type, priority, status, handler, description, period)
VALUES ($9, $1, $2, $3, $4, $5, $6, tstzrange($7, $8, '[)'))
RETURNING id::text, instrument_id::text, $10::text, type, priority, status, handler, description, result, lower(period), upper(period), created_at
`, input.InstrumentID, input.Type, input.Priority, status, input.Handler, input.Description, input.StartTime, input.EndTime, targetTenantID, instrumentName).Scan(
		&item.ID, &item.InstrumentID, &item.InstrumentName, &item.Type, &item.Priority, &item.Status, &item.Handler, &item.Description, &item.Result, &item.StartTime, &item.EndTime, &item.CreatedAt,
	)
	if err != nil {
		return MaintenanceOrder{}, err
	}
	notifications := make([]Notification, 0)

	rows, err := tx.Query(ctx, `
UPDATE reservations
SET status = 'cancelled', cancel_reason = '设备维护窗口冲突', cancelled_at = now()
WHERE instrument_id = $1
  AND status IN ('pending', 'approved')
  AND period && tstzrange($2, $3, '[)')
  AND tenant_id = $4::uuid
RETURNING id::text, COALESCE(user_id::text, ''), user_name, group_name
`, input.InstrumentID, input.StartTime, input.EndTime, targetTenantID)
	if err != nil {
		return MaintenanceOrder{}, err
	}
	cancelled := make([]string, 0)
	for rows.Next() {
		var reservationID string
		var userID string
		var userName string
		var groupName string
		if err := rows.Scan(&reservationID, &userID, &userName, &groupName); err != nil {
			rows.Close()
			return MaintenanceOrder{}, err
		}
		cancelled = append(cancelled, fmt.Sprintf("%s/%s", reservationID, userName))
		if userID != "" {
			notification, err := r.createNotificationTx(ctx, tx, targetTenantID, userID, groupName, "", "personal", "预约受维护影响", fmt.Sprintf("%s 的预约因 %s 维护窗口被取消，请重新安排。", userName, item.InstrumentName), "warning")
			if err != nil {
				rows.Close()
				return MaintenanceOrder{}, err
			}
			notifications = append(notifications, notification)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return MaintenanceOrder{}, err
	}

	if _, err := tx.Exec(ctx, `UPDATE instruments SET status = 'maintenance', maintenance_summary = $2 WHERE id = $1 AND tenant_id = $3::uuid`, input.InstrumentID, input.Description, targetTenantID); err != nil {
		return MaintenanceOrder{}, err
	}
	notification, err := r.createNotificationTx(ctx, tx, targetTenantID, "", "", "", "global", "设备维护安排", fmt.Sprintf("%s 已进入维护，影响预约 %d 条。", item.InstrumentName, len(cancelled)), "warning")
	if err != nil {
		return MaintenanceOrder{}, err
	}
	notifications = append(notifications, notification)
	if err := r.auditTx(ctx, tx, targetTenantID, input.Actor, "maintenance.create", "maintenance_order", item.ID, "", strings.Join(cancelled, ",")); err != nil {
		return MaintenanceOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaintenanceOrder{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	r.invalidateDashboard(ctx)
	return item, nil
}

func (r *Repository) StartMaintenanceOrder(ctx context.Context, id string, actor string) (MaintenanceOrder, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaintenanceOrder{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var item MaintenanceOrder
	var itemTenantID string
	err = tx.QueryRow(ctx, `
UPDATE maintenance_orders mo
SET status = 'in_progress'
WHERE mo.id = $1 AND mo.status IN ('reported', 'assigned')
  AND ($2::boolean OR mo.tenant_id = $3::uuid)
RETURNING mo.id::text, mo.tenant_id::text, COALESCE(mo.instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = mo.instrument_id), '已删除仪器'), mo.type, mo.priority, mo.status, mo.handler, mo.description, mo.result, lower(mo.period), upper(mo.period), mo.created_at
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.InstrumentID, &item.InstrumentName, &item.Type, &item.Priority, &item.Status, &item.Handler, &item.Description, &item.Result, &item.StartTime, &item.EndTime, &item.CreatedAt)
	if err != nil {
		return MaintenanceOrder{}, err
	}
	if item.InstrumentID == "" {
		return MaintenanceOrder{}, clientError("instrument has been deleted")
	}
	if _, err := tx.Exec(ctx, `UPDATE instruments SET status = 'maintenance', maintenance_summary = $2 WHERE id = $1 AND tenant_id = $3::uuid`, item.InstrumentID, item.Description, itemTenantID); err != nil {
		return MaintenanceOrder{}, err
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "maintenance.start", "maintenance_order", item.ID, "", item.Status); err != nil {
		return MaintenanceOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaintenanceOrder{}, err
	}
	r.invalidateDashboard(ctx)
	return item, nil
}

func (r *Repository) CancelMaintenanceOrder(ctx context.Context, id string, reason string, actor string) (MaintenanceOrder, error) {
	tenant := TenantFromContext(ctx)
	reason = strings.TrimSpace(reason)
	actor = strings.TrimSpace(actor)
	if reason == "" {
		reason = "维护取消"
	}
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaintenanceOrder{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var item MaintenanceOrder
	var itemTenantID string
	err = tx.QueryRow(ctx, `
UPDATE maintenance_orders mo
SET status = 'cancelled', result = $2
WHERE mo.id = $1 AND mo.status IN ('reported', 'assigned', 'in_progress')
  AND ($3::boolean OR mo.tenant_id = $4::uuid)
RETURNING mo.id::text, mo.tenant_id::text, COALESCE(mo.instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = mo.instrument_id), '已删除仪器'), mo.type, mo.priority, mo.status, mo.handler, mo.description, mo.result, lower(mo.period), upper(mo.period), mo.created_at
`, id, reason, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.InstrumentID, &item.InstrumentName, &item.Type, &item.Priority, &item.Status, &item.Handler, &item.Description, &item.Result, &item.StartTime, &item.EndTime, &item.CreatedAt)
	if err != nil {
		return MaintenanceOrder{}, err
	}
	if err := r.refreshInstrumentAfterMaintenanceTx(ctx, tx, item.InstrumentID, itemTenantID, reason); err != nil {
		return MaintenanceOrder{}, err
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "maintenance.cancel", "maintenance_order", item.ID, "", reason); err != nil {
		return MaintenanceOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaintenanceOrder{}, err
	}
	r.invalidateDashboard(ctx)
	return item, nil
}

func (r *Repository) CompleteMaintenanceOrder(ctx context.Context, id string, result string, actor string) (MaintenanceOrder, error) {
	tenant := TenantFromContext(ctx)
	result = strings.TrimSpace(result)
	actor = strings.TrimSpace(actor)
	if result == "" {
		result = "维护完成，仪器恢复可用。"
	}
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaintenanceOrder{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var item MaintenanceOrder
	var itemTenantID string
	err = tx.QueryRow(ctx, `
UPDATE maintenance_orders mo
SET status = 'completed', result = $2
WHERE mo.id = $1 AND mo.status IN ('reported', 'assigned', 'in_progress')
  AND ($3::boolean OR mo.tenant_id = $4::uuid)
RETURNING mo.id::text, mo.tenant_id::text, COALESCE(mo.instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = mo.instrument_id), '已删除仪器'), mo.type, mo.priority, mo.status, mo.handler, mo.description, mo.result, lower(mo.period), upper(mo.period), mo.created_at
`, id, result, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.InstrumentID, &item.InstrumentName, &item.Type, &item.Priority, &item.Status, &item.Handler, &item.Description, &item.Result, &item.StartTime, &item.EndTime, &item.CreatedAt)
	if err != nil {
		return MaintenanceOrder{}, err
	}
	if err := r.refreshInstrumentAfterMaintenanceTx(ctx, tx, item.InstrumentID, itemTenantID, result); err != nil {
		return MaintenanceOrder{}, err
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "maintenance.complete", "maintenance_order", item.ID, "", result); err != nil {
		return MaintenanceOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaintenanceOrder{}, err
	}
	r.invalidateDashboard(ctx)
	return item, nil
}

func (r *Repository) refreshInstrumentAfterMaintenanceTx(ctx context.Context, tx pgx.Tx, instrumentID string, tenantID string, summary string) error {
	if strings.TrimSpace(instrumentID) == "" {
		return nil
	}
	var activeCount int
	if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM maintenance_orders
WHERE instrument_id = $1 AND tenant_id = $2::uuid AND status IN ('reported', 'assigned', 'in_progress')
`, instrumentID, tenantID).Scan(&activeCount); err != nil {
		return err
	}
	if activeCount > 0 {
		_, err := tx.Exec(ctx, `UPDATE instruments SET status = 'maintenance', maintenance_summary = $2 WHERE id = $1 AND tenant_id = $3::uuid`, instrumentID, summary, tenantID)
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE instruments SET status = 'available', maintenance_summary = $2 WHERE id = $1 AND tenant_id = $3::uuid`, instrumentID, summary, tenantID)
	return err
}
