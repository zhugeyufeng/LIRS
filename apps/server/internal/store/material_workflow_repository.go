package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) MaterialRequests(ctx context.Context) ([]MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT mr.id::text, mr.material_id::text, m.name, COALESCE(mr.requester_id::text, ''),
       mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       COALESCE(mr.unit_id::text, ''), COALESCE(mu.unit_code, ''), COALESCE(mu.location, mb.location, ''),
       mr.quantity, mr.purpose, mr.status, mr.created_at
FROM material_requests mr
JOIN materials m ON m.id = mr.material_id
LEFT JOIN material_batches mb ON mb.id = mr.batch_id
LEFT JOIN material_units mu ON mu.id = mr.unit_id
WHERE ($1::boolean OR mr.tenant_id = $2::uuid)
ORDER BY mr.created_at DESC
LIMIT 100
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialRequest, 0)
	for rows.Next() {
		item, err := scanMaterialRequest(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) MaterialRequestsForMonth(ctx context.Context, month string) ([]MaterialRequestExportRow, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT mr.id::text, mr.material_id::text, m.name, COALESCE(mr.requester_id::text, ''),
       mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       COALESCE(mr.unit_id::text, ''), COALESCE(mu.unit_code, ''),
       COALESCE(NULLIF(mu.location, ''), NULLIF(mb.location, ''), NULLIF(concat_ws(' / ', NULLIF(m.storage_room, ''), NULLIF(m.storage_cabinet, ''), NULLIF(m.storage_layer, ''), NULLIF(m.storage_slot, '')), ''), ''),
       mr.quantity, mr.purpose, mr.status, mr.created_at,
       COALESCE(NULLIF(m.catalog_no, ''), NULLIF(m.cas_no, ''), NULLIF(m.grade, ''), ''),
       COALESCE(NULLIF(m.manufacturer, ''), NULLIF(m.supplier, ''), ''),
       m.spec,
       m.unit,
       COALESCE(mu.expires_at::text, mb.expires_at::text, m.expires_at::text, ''),
       COALESCE((
           SELECT string_agg(maa.actor, '，' ORDER BY maa.created_at)
           FROM material_approval_actions maa
           WHERE maa.material_request_id = mr.id
             AND maa.action IN ('approve', 'outbound')
       ), '')
FROM material_requests mr
JOIN materials m ON m.id = mr.material_id
LEFT JOIN material_batches mb ON mb.id = mr.batch_id
LEFT JOIN material_units mu ON mu.id = mr.unit_id
WHERE ($1::boolean OR mr.tenant_id = $2::uuid)
  AND to_char(mr.created_at, 'YYYY-MM') = $3
ORDER BY mr.created_at, mr.id
`, tenant.AllTenants, tenant.TenantID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialRequestExportRow, 0)
	for rows.Next() {
		var item MaterialRequestExportRow
		err := rows.Scan(
			&item.ID,
			&item.MaterialID,
			&item.MaterialName,
			&item.RequesterID,
			&item.Requester,
			&item.GroupName,
			&item.BatchID,
			&item.BatchNo,
			&item.UnitID,
			&item.UnitCode,
			&item.Location,
			&item.Quantity,
			&item.Purpose,
			&item.Status,
			&item.CreatedAt,
			&item.StandardNo,
			&item.Brand,
			&item.Spec,
			&item.Unit,
			&item.ExpiresAt,
			&item.ApprovalInfo,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) MaterialRequest(ctx context.Context, id string) (MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	return scanMaterialRequest(r.db.QueryRow(ctx, `
SELECT mr.id::text, mr.material_id::text, m.name, COALESCE(mr.requester_id::text, ''),
       mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       COALESCE(mr.unit_id::text, ''), COALESCE(mu.unit_code, ''), COALESCE(mu.location, mb.location, ''),
       mr.quantity, mr.purpose, mr.status, mr.created_at
FROM material_requests mr
JOIN materials m ON m.id = mr.material_id
LEFT JOIN material_batches mb ON mb.id = mr.batch_id
LEFT JOIN material_units mu ON mu.id = mr.unit_id
WHERE mr.id = $1
  AND ($2::boolean OR mr.tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID))
}

func scanMaterialRequest(row scanner) (MaterialRequest, error) {
	var item MaterialRequest
	err := row.Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt)
	return item, err
}

func (r *Repository) CreateMaterialRequest(ctx context.Context, input MaterialRequestInput) (MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	input.RequesterID = strings.TrimSpace(input.RequesterID)
	input.Requester = strings.TrimSpace(input.Requester)
	input.BatchID = strings.TrimSpace(input.BatchID)
	input.UnitID = strings.TrimSpace(input.UnitID)
	input.Purpose = strings.TrimSpace(input.Purpose)
	if input.RequesterID == "" {
		input.RequesterID = strings.TrimSpace(tenant.Actor.UserID)
	}
	if input.Requester == "" {
		input.Requester = strings.TrimSpace(tenant.Actor.Name)
	}
	if input.MaterialID == "" || input.Quantity <= 0 || input.Purpose == "" || (input.RequesterID == "" && input.Requester == "") {
		return MaterialRequest{}, clientError("invalid material request input")
	}

	var requesterID, requesterTenantID, requesterName, requesterStatus, groupName string
	var emailVerified bool
	var err error
	if input.RequesterID != "" {
		err = r.db.QueryRow(ctx, `
SELECT id::text, tenant_id::text, name, status, group_name, email_verified
FROM users
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.RequesterID, tenant.AllTenants, tenant.TenantID).Scan(&requesterID, &requesterTenantID, &requesterName, &requesterStatus, &groupName, &emailVerified)
	} else {
		err = r.db.QueryRow(ctx, `
SELECT id::text, tenant_id::text, name, status, group_name, email_verified
FROM users
WHERE name = $1 AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY created_at DESC
LIMIT 1
`, input.Requester, tenant.AllTenants, tenant.TenantID).Scan(&requesterID, &requesterTenantID, &requesterName, &requesterStatus, &groupName, &emailVerified)
	}
	if err != nil {
		return MaterialRequest{}, err
	}
	if requesterStatus != "active" {
		return MaterialRequest{}, clientError("user is not active")
	}
	if !emailVerified {
		return MaterialRequest{}, clientError("email must be verified before requesting materials")
	}
	var availableStock int
	var materialTenantID, materialStatus string
	var expiresAt string
	if err := r.db.QueryRow(ctx, `
SELECT tenant_id::text, stock, status, COALESCE(expires_at::text, '')
FROM materials
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.MaterialID, tenant.AllTenants, tenant.TenantID).Scan(&materialTenantID, &availableStock, &materialStatus, &expiresAt); err != nil {
		return MaterialRequest{}, err
	}
	if requesterTenantID != materialTenantID {
		return MaterialRequest{}, clientError("requester and material must belong to the same tenant")
	}
	if materialStatus == "disabled" {
		return MaterialRequest{}, clientError("material is disabled")
	}
	if expiresAt != "" {
		expireDate, err := time.Parse("2006-01-02", expiresAt)
		if err != nil {
			return MaterialRequest{}, err
		}
		today := appToday()
		if expireDate.Before(today) {
			return MaterialRequest{}, clientError("material is expired")
		}
	}
	batchID := ""
	var batchNo string
	unitID := ""
	var unitCode string
	var unitLocation string
	if input.UnitID == "" {
		return MaterialRequest{}, clientError("material request requires unit")
	}
	if input.Quantity != 1 {
		return MaterialRequest{}, clientError("material unit request quantity must be 1")
	}
	if err := r.db.QueryRow(ctx, `
SELECT mu.id::text, COALESCE(mu.batch_id::text, ''), COALESCE(mb.batch_no, ''), mu.unit_code, COALESCE(mu.location, mb.location, ''), COALESCE(mu.expires_at::text, '')
FROM material_units mu
LEFT JOIN material_batches mb ON mb.id = mu.batch_id
WHERE mu.id = $1
  AND mu.material_id = $2
  AND mu.tenant_id = $3::uuid
  AND mu.status = 'available'
`, input.UnitID, input.MaterialID, materialTenantID).Scan(&unitID, &batchID, &batchNo, &unitCode, &unitLocation, &expiresAt); err != nil {
		return MaterialRequest{}, err
	}
	availableStock = 1
	if expiresAt != "" {
		expireDate, err := time.Parse("2006-01-02", expiresAt)
		if err != nil {
			return MaterialRequest{}, err
		}
		today := appToday()
		if expireDate.Before(today) {
			return MaterialRequest{}, clientError("material unit is expired")
		}
	}
	if availableStock < input.Quantity {
		return MaterialRequest{}, clientError("insufficient material stock")
	}
	requestStatus := "approved"
	if materialApprovalRequired(ctx, r.db, input.MaterialID, materialTenantID) {
		requestStatus = "pending"
	}

	var item MaterialRequest
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialRequest{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	reserveTag, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'reserved', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'available'
`, unitID, input.MaterialID, materialTenantID)
	if err != nil {
		return MaterialRequest{}, err
	}
	if reserveTag.RowsAffected() != 1 {
		return MaterialRequest{}, clientError("material unit is not available")
	}
	if batchID != "" {
		if err := syncMaterialBatchQuantity(ctx, tx, batchID); err != nil {
			return MaterialRequest{}, err
		}
	}
	if _, err := syncMaterialStock(ctx, tx, input.MaterialID, materialTenantID); err != nil {
		return MaterialRequest{}, err
	}
	err = tx.QueryRow(ctx, `
INSERT INTO material_requests (tenant_id, material_id, requester_id, requester, group_name, quantity, purpose, status, decided_at, batch_id, unit_id)
VALUES ($7, $1, $2, $3, $4, $5, $6, $8, CASE WHEN $8 = 'approved' THEN now() ELSE NULL END, NULLIF($9, '')::uuid, NULLIF($11, '')::uuid)
RETURNING id::text, material_id::text, (SELECT name FROM materials WHERE id = material_id),
          COALESCE(requester_id::text, ''), requester, group_name, COALESCE(batch_id::text, ''), $10,
          COALESCE(unit_id::text, ''), $12, $13,
          quantity, purpose, status, created_at
`, input.MaterialID, requesterID, requesterName, groupName, input.Quantity, input.Purpose, materialTenantID, requestStatus, batchID, batchNo, unitID, unitCode, unitLocation).Scan(
		&item.ID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt,
	)
	if err != nil {
		return MaterialRequest{}, err
	}
	notification, err := r.createNotificationTx(ctx, tx, materialTenantID, item.RequesterID, item.GroupName, "", "group", "耗材申领状态更新", fmt.Sprintf("%s 提交了 %s x%d 的申领，当前状态：%s。", item.Requester, item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), "info")
	if err != nil {
		return MaterialRequest{}, err
	}
	notifications = append(notifications, notification)
	if err := r.auditTx(ctx, tx, materialTenantID, item.Requester, "material.request", "material_request", item.ID, "", item.Status); err != nil {
		return MaterialRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialRequest{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) ApproveMaterialRequest(ctx context.Context, id string, approved bool, actor string, comment string) (MaterialRequest, error) {
	status := "rejected"
	if approved {
		status = "approved"
	}
	return r.updateMaterialRequestStatus(ctx, id, status, actor, comment)
}

func materialApprovalRequired(ctx context.Context, db queryRower, materialID string, tenantID string) bool {
	var approvalRequired bool
	err := db.QueryRow(ctx, `SELECT approval_required FROM materials WHERE id = $1 AND tenant_id = $2::uuid`, materialID, tenantID).Scan(&approvalRequired)
	return err == nil && approvalRequired
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *Repository) OutboundMaterialRequest(ctx context.Context, id string, actor string) (MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialRequest{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 2)

	var item MaterialRequest
	var itemTenantID string
	err = tx.QueryRow(ctx, `
SELECT mr.id::text, mr.tenant_id::text, mr.material_id::text, m.name, COALESCE(mr.requester_id::text, ''),
       mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''),
       COALESCE((SELECT batch_no FROM material_batches WHERE id = mr.batch_id), ''),
       COALESCE(mr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mr.unit_id), ''),
       COALESCE((SELECT location FROM material_units WHERE id = mr.unit_id), (SELECT location FROM material_batches WHERE id = mr.batch_id), ''),
       mr.quantity, mr.purpose, mr.status, mr.created_at
FROM material_requests mr
JOIN materials m ON m.id = mr.material_id
WHERE mr.id = $1 AND mr.status = 'approved'
  AND ($2::boolean OR mr.tenant_id = $3::uuid)
FOR UPDATE OF mr, m
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt)
	if err != nil {
		return MaterialRequest{}, err
	}
	var remainingStock, warningLine int
	var materialUnit string
	if item.UnitID == "" || item.Quantity != 1 {
		return MaterialRequest{}, clientError("material request missing unit")
	}
	if err := tx.QueryRow(ctx, `
SELECT COALESCE(batch_id::text, ''), unit_code, COALESCE(location, '')
FROM material_units
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
FOR UPDATE
`, item.UnitID, item.MaterialID, itemTenantID).Scan(&item.BatchID, &item.UnitCode, &item.Location); err != nil {
		return MaterialRequest{}, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'used', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
		return MaterialRequest{}, err
	}
	if item.BatchID != "" {
		if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
			return MaterialRequest{}, err
		}
		_ = tx.QueryRow(ctx, `SELECT batch_no FROM material_batches WHERE id = $1`, item.BatchID).Scan(&item.BatchNo)
	}
	remainingStock, err = syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID)
	if err != nil {
		return MaterialRequest{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT warning_line, unit FROM materials WHERE id = $1 AND tenant_id = $2::uuid`, item.MaterialID, itemTenantID).Scan(&warningLine, &materialUnit); err != nil {
		return MaterialRequest{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO inventory_ledger (tenant_id, material_id, request_id, change_qty, reason)
VALUES ($1, $2, $3, $4, $5)
`, itemTenantID, item.MaterialID, item.ID, -item.Quantity, materialUnitReason("申领出库", item.BatchNo, item.UnitCode)); err != nil {
		return MaterialRequest{}, err
	}
	if remainingStock <= warningLine {
		created, err := r.createMaterialEventNotificationsTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "耗材库存预警", fmt.Sprintf("%s 出库后库存 %d%s，低于预警线 %d%s。", item.MaterialName, remainingStock, materialUnit, warningLine, materialUnit), "warning")
		if err != nil {
			return MaterialRequest{}, err
		}
		notifications = append(notifications, created...)
	}
	err = tx.QueryRow(ctx, `
UPDATE material_requests mr
SET status = 'outbound'
WHERE mr.id = $1 AND mr.tenant_id = $2::uuid
RETURNING mr.id::text, mr.material_id::text, (SELECT name FROM materials WHERE id = mr.material_id),
          COALESCE(mr.requester_id::text, ''), mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''), $3,
          COALESCE(mr.unit_id::text, ''), $4, $5,
          mr.quantity, mr.purpose, mr.status, mr.created_at
`, id, itemTenantID, item.BatchNo, item.UnitCode, item.Location).Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt)
	if err != nil {
		return MaterialRequest{}, err
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申领状态更新", fmt.Sprintf("%s x%d 的申领状态已更新为%s，储存位置：%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status), firstNonEmpty(item.Location, "未登记")), "success")
		if err != nil {
			return MaterialRequest{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material.outbound", "material_request", item.ID, "approved", item.Status); err != nil {
		return MaterialRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialRequest{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) CancelMaterialRequest(ctx context.Context, id string, actor string) (MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var item MaterialRequest
	var itemTenantID string
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialRequest{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	err = tx.QueryRow(ctx, `
UPDATE material_requests mr
SET status = 'cancelled'
WHERE mr.id = $1 AND mr.status IN ('pending', 'approved')
  AND ($2::boolean OR mr.tenant_id = $3::uuid)
RETURNING mr.id::text, mr.tenant_id::text, mr.material_id::text, (SELECT name FROM materials WHERE id = mr.material_id),
          COALESCE(mr.requester_id::text, ''), mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''),
          COALESCE((SELECT batch_no FROM material_batches WHERE id = mr.batch_id), ''),
          COALESCE(mr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mr.unit_id), ''),
          COALESCE((SELECT location FROM material_units WHERE id = mr.unit_id), (SELECT location FROM material_batches WHERE id = mr.batch_id), ''),
          mr.quantity, mr.purpose, mr.status, mr.created_at
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt)
	if err != nil {
		return MaterialRequest{}, err
	}
	if item.UnitID != "" {
		if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'available', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
			return MaterialRequest{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
			return MaterialRequest{}, err
		}
		if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
			return MaterialRequest{}, err
		}
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申领状态更新", fmt.Sprintf("%s x%d 的申领状态已更新为%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialRequest{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material.cancel", "material_request", item.ID, "", item.Status); err != nil {
		return MaterialRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialRequest{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) MaterialDamages(ctx context.Context) ([]MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT mdr.id::text, mdr.material_id::text, m.name, COALESCE(mdr.reporter_id::text, ''),
       mdr.reporter, mdr.group_name, COALESCE(mdr.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       COALESCE(mdr.unit_id::text, ''), COALESCE(mu.unit_code, ''),
       mdr.quantity, mdr.reason, mdr.photo_url, mdr.attachment_url,
       mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
       COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
       COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
FROM material_damage_reports mdr
JOIN materials m ON m.id = mdr.material_id
LEFT JOIN material_batches mb ON mb.id = mdr.batch_id
LEFT JOIN material_units mu ON mu.id = mdr.unit_id
WHERE ($1::boolean OR mdr.tenant_id = $2::uuid)
ORDER BY mdr.created_at DESC
LIMIT 200
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialDamage, 0)
	for rows.Next() {
		item, err := scanMaterialDamage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) MaterialDamage(ctx context.Context, id string) (MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	return scanMaterialDamage(r.db.QueryRow(ctx, `
SELECT mdr.id::text, mdr.material_id::text, m.name, COALESCE(mdr.reporter_id::text, ''),
       mdr.reporter, mdr.group_name, COALESCE(mdr.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       COALESCE(mdr.unit_id::text, ''), COALESCE(mu.unit_code, ''),
       mdr.quantity, mdr.reason, mdr.photo_url, mdr.attachment_url,
       mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
       COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
       COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
FROM material_damage_reports mdr
JOIN materials m ON m.id = mdr.material_id
LEFT JOIN material_batches mb ON mb.id = mdr.batch_id
LEFT JOIN material_units mu ON mu.id = mdr.unit_id
WHERE mdr.id = $1
  AND ($2::boolean OR mdr.tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID))
}

func scanMaterialDamage(row scanner) (MaterialDamage, error) {
	var item MaterialDamage
	err := row.Scan(
		&item.ID,
		&item.MaterialID,
		&item.MaterialName,
		&item.ReporterID,
		&item.Reporter,
		&item.GroupName,
		&item.BatchID,
		&item.BatchNo,
		&item.UnitID,
		&item.UnitCode,
		&item.Quantity,
		&item.Reason,
		&item.PhotoURL,
		&item.AttachmentURL,
		&item.Status,
		&item.Reviewer,
		&item.ReviewComment,
		&item.CreatedAt,
		&item.ReviewedAt,
		&item.ProcessedAt,
	)
	return item, err
}

func (r *Repository) CreateMaterialDamage(ctx context.Context, input MaterialDamageInput) (MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	input.MaterialID = strings.TrimSpace(input.MaterialID)
	input.ReporterID = strings.TrimSpace(input.ReporterID)
	input.Reporter = strings.TrimSpace(input.Reporter)
	input.UnitID = strings.TrimSpace(input.UnitID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.PhotoURL = strings.TrimSpace(input.PhotoURL)
	input.AttachmentURL = strings.TrimSpace(input.AttachmentURL)
	if input.ReporterID == "" {
		input.ReporterID = strings.TrimSpace(tenant.Actor.UserID)
	}
	if input.Reporter == "" {
		input.Reporter = strings.TrimSpace(tenant.Actor.Name)
	}
	if input.MaterialID == "" || input.UnitID == "" || input.Quantity != 1 || input.Reason == "" || (input.ReporterID == "" && input.Reporter == "") {
		return MaterialDamage{}, clientError("invalid material damage input")
	}

	reporterID, reporterTenantID, reporterName, reporterStatus, groupName, _, err := r.resolveMaterialActor(ctx, input.ReporterID, input.Reporter, tenant)
	if err != nil {
		return MaterialDamage{}, err
	}
	if reporterStatus != "active" {
		return MaterialDamage{}, clientError("user is not active")
	}

	var materialTenantID, materialStatus string
	if err := r.db.QueryRow(ctx, `
SELECT tenant_id::text, status
FROM materials
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.MaterialID, tenant.AllTenants, tenant.TenantID).Scan(&materialTenantID, &materialStatus); err != nil {
		return MaterialDamage{}, err
	}
	if reporterTenantID != materialTenantID {
		return MaterialDamage{}, clientError("reporter and material must belong to the same tenant")
	}
	if materialStatus == "disabled" {
		return MaterialDamage{}, clientError("material is disabled")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialDamage{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	var batchID, batchNo, unitID, unitCode string
	if err := tx.QueryRow(ctx, `
SELECT mu.id::text, COALESCE(mu.batch_id::text, ''), COALESCE(mb.batch_no, ''), mu.unit_code
FROM material_units mu
LEFT JOIN material_batches mb ON mb.id = mu.batch_id
WHERE mu.id = $1
  AND mu.material_id = $2
  AND mu.tenant_id = $3::uuid
  AND mu.status = 'available'
FOR UPDATE
`, input.UnitID, input.MaterialID, materialTenantID).Scan(&unitID, &batchID, &batchNo, &unitCode); err != nil {
		return MaterialDamage{}, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'reserved', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'available'
`, unitID, input.MaterialID, materialTenantID); err != nil {
		return MaterialDamage{}, err
	}
	if batchID != "" {
		if err := syncMaterialBatchQuantity(ctx, tx, batchID); err != nil {
			return MaterialDamage{}, err
		}
	}
	if _, err := syncMaterialStock(ctx, tx, input.MaterialID, materialTenantID); err != nil {
		return MaterialDamage{}, err
	}
	item, err := scanMaterialDamage(tx.QueryRow(ctx, `
INSERT INTO material_damage_reports (tenant_id, material_id, reporter_id, reporter, group_name, batch_id, unit_id, quantity, reason, photo_url, attachment_url)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::uuid, $7, $8, $9, $10, $11)
RETURNING id::text, material_id::text, (SELECT name FROM materials WHERE id = material_id),
          COALESCE(reporter_id::text, ''), reporter, group_name, COALESCE(batch_id::text, ''), $12,
          COALESCE(unit_id::text, ''), $13,
          quantity, reason, photo_url, attachment_url,
          status, reviewer, review_comment, created_at,
          COALESCE(reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
          COALESCE(processed_at, '0001-01-01 00:00:00+00'::timestamptz)
`, materialTenantID, input.MaterialID, optionalID(reporterID), reporterName, groupName, batchID, unitID, input.Quantity, input.Reason, input.PhotoURL, input.AttachmentURL, batchNo, unitCode))
	if err != nil {
		return MaterialDamage{}, err
	}
	notification, err := r.createNotificationTx(ctx, tx, materialTenantID, item.ReporterID, item.GroupName, "", "group", "损毁登记状态更新", fmt.Sprintf("%s 登记了 %s 编号 %s 的损毁，当前状态：%s。", item.Reporter, item.MaterialName, item.UnitCode, materialWorkflowStatusLabel(item.Status)), "warning")
	if err != nil {
		return MaterialDamage{}, err
	}
	notifications = append(notifications, notification)
	if err := r.auditTx(ctx, tx, materialTenantID, item.Reporter, "material_damage.create", "material_damage", item.ID, "", item.Status); err != nil {
		return MaterialDamage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialDamage{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) ApproveMaterialDamage(ctx context.Context, id string, approved bool, actor string, comment string) (MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	comment = strings.TrimSpace(comment)
	if actor == "" {
		actor = "system"
	}
	status := "rejected"
	if approved {
		status = "approved"
	}
	if comment == "" {
		comment = status
	}
	var itemTenantID string
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialDamage{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	item, err := scanMaterialDamage(tx.QueryRow(ctx, `
UPDATE material_damage_reports mdr
SET status = $2, reviewer = $3, review_comment = $4, reviewed_at = now()
WHERE mdr.id = $1 AND mdr.status = 'pending'
  AND ($5::boolean OR mdr.tenant_id = $6::uuid)
RETURNING mdr.id::text, mdr.material_id::text, (SELECT name FROM materials WHERE id = mdr.material_id),
          COALESCE(mdr.reporter_id::text, ''), mdr.reporter, mdr.group_name,
          COALESCE(mdr.batch_id::text, ''), COALESCE((SELECT batch_no FROM material_batches WHERE id = mdr.batch_id), ''),
          COALESCE(mdr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mdr.unit_id), ''),
          mdr.quantity, mdr.reason,
          mdr.photo_url, mdr.attachment_url, mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
          COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
          COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
`, id, status, actor, comment, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return MaterialDamage{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT tenant_id::text FROM material_damage_reports WHERE id = $1`, item.ID).Scan(&itemTenantID); err != nil {
		return MaterialDamage{}, err
	}
	if status == "rejected" && item.UnitID != "" {
		if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'available', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
			return MaterialDamage{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
			return MaterialDamage{}, err
		}
		if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
			return MaterialDamage{}, err
		}
	}
	if item.ReporterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.ReporterID, item.GroupName, "", "personal", "损毁登记状态更新", fmt.Sprintf("%s 编号 %s 的损毁登记状态已更新为%s。", item.MaterialName, item.UnitCode, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialDamage{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_damage."+status, "material_damage", item.ID, "pending", status); err != nil {
		return MaterialDamage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialDamage{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) ProcessMaterialDamage(ctx context.Context, id string, actor string) (MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialDamage{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)

	var item MaterialDamage
	var itemTenantID string
	err = tx.QueryRow(ctx, `
SELECT mdr.id::text, mdr.tenant_id::text, mdr.material_id::text, m.name, COALESCE(mdr.reporter_id::text, ''),
       mdr.reporter, mdr.group_name,
       COALESCE(mdr.batch_id::text, ''), COALESCE((SELECT batch_no FROM material_batches WHERE id = mdr.batch_id), ''),
       COALESCE(mdr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mdr.unit_id), ''),
       mdr.quantity, mdr.reason, mdr.photo_url, mdr.attachment_url,
       mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
       COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
       COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
FROM material_damage_reports mdr
JOIN materials m ON m.id = mdr.material_id
WHERE mdr.id = $1 AND mdr.status = 'approved'
  AND ($2::boolean OR mdr.tenant_id = $3::uuid)
FOR UPDATE
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.MaterialID, &item.MaterialName, &item.ReporterID, &item.Reporter, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Quantity, &item.Reason, &item.PhotoURL, &item.AttachmentURL, &item.Status, &item.Reviewer, &item.ReviewComment, &item.CreatedAt, &item.ReviewedAt, &item.ProcessedAt)
	if err != nil {
		return MaterialDamage{}, err
	}
	if item.UnitID == "" || item.Quantity != 1 {
		return MaterialDamage{}, clientError("material damage missing unit")
	}
	if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'damaged', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
		return MaterialDamage{}, err
	}
	if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
		return MaterialDamage{}, err
	}
	if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
		return MaterialDamage{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO inventory_ledger (tenant_id, material_id, damage_id, change_qty, reason)
VALUES ($1, $2, $3, $4, $5)
`, itemTenantID, item.MaterialID, item.ID, -item.Quantity, materialUnitReason("损毁处理："+item.Reason, item.BatchNo, item.UnitCode)); err != nil {
		return MaterialDamage{}, err
	}
	if err := tx.QueryRow(ctx, `
UPDATE material_damage_reports mdr
SET status = 'processed', processed_at = now()
WHERE mdr.id = $1 AND mdr.tenant_id = $2::uuid
RETURNING mdr.id::text, mdr.material_id::text, (SELECT name FROM materials WHERE id = mdr.material_id),
          COALESCE(mdr.reporter_id::text, ''), mdr.reporter, mdr.group_name,
          COALESCE(mdr.batch_id::text, ''), $3,
          COALESCE(mdr.unit_id::text, ''), $4,
          mdr.quantity, mdr.reason,
          mdr.photo_url, mdr.attachment_url, mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
          COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
          COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
`, id, itemTenantID, item.BatchNo, item.UnitCode).Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.ReporterID, &item.Reporter, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Quantity, &item.Reason, &item.PhotoURL, &item.AttachmentURL, &item.Status, &item.Reviewer, &item.ReviewComment, &item.CreatedAt, &item.ReviewedAt, &item.ProcessedAt); err != nil {
		return MaterialDamage{}, err
	}
	if item.ReporterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.ReporterID, item.GroupName, "", "personal", "损毁登记状态更新", fmt.Sprintf("%s 编号 %s 的损毁登记状态已更新为%s。", item.MaterialName, item.UnitCode, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialDamage{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_damage.process", "material_damage", item.ID, "approved", item.Status); err != nil {
		return MaterialDamage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialDamage{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) CancelMaterialDamage(ctx context.Context, id string, actor string) (MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var itemTenantID string
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialDamage{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	item, err := scanMaterialDamage(tx.QueryRow(ctx, `
UPDATE material_damage_reports mdr
SET status = 'cancelled'
WHERE mdr.id = $1 AND mdr.status = 'pending'
  AND ($2::boolean OR mdr.tenant_id = $3::uuid)
RETURNING mdr.id::text, mdr.material_id::text, (SELECT name FROM materials WHERE id = mdr.material_id),
          COALESCE(mdr.reporter_id::text, ''), mdr.reporter, mdr.group_name,
          COALESCE(mdr.batch_id::text, ''), COALESCE((SELECT batch_no FROM material_batches WHERE id = mdr.batch_id), ''),
          COALESCE(mdr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mdr.unit_id), ''),
          mdr.quantity, mdr.reason,
          mdr.photo_url, mdr.attachment_url, mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
          COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
          COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
`, id, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return MaterialDamage{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT tenant_id::text FROM material_damage_reports WHERE id = $1`, item.ID).Scan(&itemTenantID); err != nil {
		return MaterialDamage{}, err
	}
	if item.UnitID != "" {
		if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'available', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
			return MaterialDamage{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
			return MaterialDamage{}, err
		}
		if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
			return MaterialDamage{}, err
		}
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_damage.cancel", "material_damage", item.ID, "", item.Status); err != nil {
		return MaterialDamage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialDamage{}, err
	}
	return item, nil
}
