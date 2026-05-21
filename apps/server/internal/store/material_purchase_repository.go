package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) MaterialPurchases(ctx context.Context) ([]MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT `+materialPurchaseSelectColumnsSQL+`
FROM material_purchases mp
LEFT JOIN materials m ON m.id = mp.material_id
WHERE ($1::boolean OR mp.tenant_id = $2::uuid)
ORDER BY mp.created_at DESC
LIMIT 100
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialPurchase, 0)
	for rows.Next() {
		item, err := scanMaterialPurchase(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) MaterialPurchaseMonthlyConfirmations(ctx context.Context) ([]MaterialPurchaseMonthlyConfirmation, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, month, confirmed_by, confirmed_at
FROM material_purchase_monthly_confirmations
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY month DESC
LIMIT 120
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]MaterialPurchaseMonthlyConfirmation, 0)
	for rows.Next() {
		var item MaterialPurchaseMonthlyConfirmation
		if err := rows.Scan(&item.ID, &item.Month, &item.ConfirmedBy, &item.ConfirmedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ConfirmMaterialPurchaseMonth(ctx context.Context, month string, actor string) (MaterialPurchaseMonthlyConfirmation, error) {
	tenant := TenantFromContext(ctx)
	month = strings.TrimSpace(month)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	if !validMaterialPurchaseMonth(month) {
		return MaterialPurchaseMonthlyConfirmation{}, clientError("invalid material purchase month")
	}
	var item MaterialPurchaseMonthlyConfirmation
	err := r.db.QueryRow(ctx, `
INSERT INTO material_purchase_monthly_confirmations (tenant_id, month, confirmed_by)
VALUES ($1, $2, $3)
ON CONFLICT (tenant_id, month) DO UPDATE
SET confirmed_by = EXCLUDED.confirmed_by,
    confirmed_at = now()
RETURNING id::text, month, confirmed_by, confirmed_at
`, tenant.TenantID, month, actor).Scan(&item.ID, &item.Month, &item.ConfirmedBy, &item.ConfirmedAt)
	if err != nil {
		return MaterialPurchaseMonthlyConfirmation{}, err
	}
	r.audit(ctx, actor, "material_purchase.month_confirm", "material_purchase_month", item.Month, "", item.ConfirmedBy)
	return item, nil
}

func validMaterialPurchaseMonth(month string) bool {
	if len(month) != len("2006-01") {
		return false
	}
	_, err := time.Parse("2006-01", month)
	return err == nil
}

func (r *Repository) MaterialPurchase(ctx context.Context, id string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	return scanMaterialPurchase(r.db.QueryRow(ctx, `
SELECT `+materialPurchaseSelectColumnsSQL+`
FROM material_purchases mp
LEFT JOIN materials m ON m.id = mp.material_id
WHERE mp.id = $1
  AND ($2::boolean OR mp.tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID))
}

func scanMaterialPurchase(row scanner) (MaterialPurchase, error) {
	var item MaterialPurchase
	err := row.Scan(
		&item.ID,
		&item.PurchaseSerialNo,
		&item.MonthlyConfirmed,
		&item.MaterialID,
		&item.MaterialName,
		&item.PurchasableMaterialID,
		&item.PurchaseIDNo,
		&item.PurchaseSequenceNo,
		&item.PurchaseProjectName,
		&item.PurchaseItemName,
		&item.PurchaseBrand,
		&item.PurchaseSpec,
		&item.PurchaseUnit,
		&item.PurchaseRemark,
		&item.PurchaseTechnicalRequirement,
		&item.PurchaseMinSpec,
		&item.RequesterID,
		&item.Requester,
		&item.RequesterPhone,
		&item.RequesterEmail,
		&item.GroupName,
		&item.Quantity,
		&item.EstimatedUnitPrice,
		&item.Supplier,
		&item.Reason,
		&item.Status,
		&item.CreatedAt,
	)
	return item, err
}

const materialPurchaseSelectColumnsSQL = `mp.id::text,
       COALESCE(mp.purchase_serial_no, ''),
       EXISTS (
           SELECT 1
           FROM material_purchase_monthly_confirmations mpmc
           WHERE mpmc.tenant_id = mp.tenant_id
             AND mpmc.month = to_char(mp.created_at, 'YYYY-MM')
       ) AS monthly_confirmed,
       COALESCE(mp.material_id::text, ''),
       COALESCE(NULLIF(mp.purchase_item_name, ''), NULLIF(mp.purchase_project_name, ''), m.name, ''),
       COALESCE(mp.purchasable_material_id::text, ''),
       mp.purchase_id_no,
       mp.purchase_sequence_no,
       COALESCE(NULLIF(mp.purchase_project_name, ''), m.name, ''),
       COALESCE(NULLIF(mp.purchase_item_name, ''), NULLIF(mp.purchase_project_name, ''), m.name, ''),
       COALESCE(NULLIF(mp.purchase_brand, ''), m.manufacturer, ''),
       COALESCE(NULLIF(mp.purchase_spec, ''), m.spec, ''),
       COALESCE(NULLIF(mp.purchase_unit, ''), m.unit, ''),
       mp.purchase_remark,
       mp.purchase_technical_requirement,
       mp.purchase_min_spec,
       COALESCE(mp.requester_id::text, ''),
       mp.requester,
       COALESCE(mp.requester_phone, ''),
       COALESCE(mp.requester_email, ''),
       mp.group_name,
       mp.quantity,
       mp.estimated_unit_price::float8,
       mp.supplier,
       mp.reason,
       mp.status,
       mp.created_at`

func (r *Repository) materialPurchaseBySerial(ctx context.Context, serial string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return MaterialPurchase{}, clientError("material purchase serial no is required")
	}
	item, err := scanMaterialPurchase(r.db.QueryRow(ctx, `
SELECT `+materialPurchaseSelectColumnsSQL+`
FROM material_purchases mp
LEFT JOIN materials m ON m.id = mp.material_id
WHERE mp.tenant_id = $2::uuid
  AND (
      mp.id::text = $1
      OR lower(mp.purchase_serial_no) = lower($1)
      OR lower(mp.purchase_id_no) = lower($1)
      OR lower(mp.purchase_sequence_no) = lower($1)
)
ORDER BY mp.created_at DESC
LIMIT 1
`, serial, tenant.TenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MaterialPurchase{}, clientError("material purchase serial no not found")
		}
		return MaterialPurchase{}, err
	}
	return item, nil
}

func canManageMaterialsRole(role string) bool {
	switch role {
	case "material_admin", "tenant_admin", "lab_admin", "super_admin":
		return true
	default:
		return false
	}
}

func (r *Repository) applyMaterialPurchaseToMaterialInput(ctx context.Context, input MaterialInput, purchase MaterialPurchase) MaterialInput {
	if purchase.ID == "" {
		return normalizeMaterial(input)
	}
	if input.Name == "" {
		input.Name = firstNonEmpty(purchase.PurchaseItemName, purchase.MaterialName, purchase.PurchaseProjectName)
	}
	input.Spec = firstNonEmpty(input.Spec, purchase.PurchaseSpec)
	input.Unit = firstNonEmpty(input.Unit, purchase.PurchaseUnit)
	if input.UnitPrice <= 0 {
		input.UnitPrice = purchase.EstimatedUnitPrice
	}
	if input.Stock <= 0 && purchase.Quantity > 0 {
		input.Stock = purchase.Quantity
	}
	input.Supplier = firstNonEmpty(input.Supplier, purchase.Supplier, purchase.PurchaseBrand)
	input.Manufacturer = firstNonEmpty(input.Manufacturer, purchase.PurchaseBrand)
	input.CatalogNo = firstNonEmpty(input.CatalogNo, purchase.PurchaseIDNo)
	input.TenderContract = firstNonEmpty(input.TenderContract, purchase.PurchaseProjectName)
	input.ContractNo = firstNonEmpty(input.ContractNo, purchase.PurchaseProjectName)
	input.Remark = firstNonEmpty(input.Remark, purchase.PurchaseRemark)
	if input.BatchNo == "" {
		input.BatchNo = firstNonEmpty(purchase.PurchaseSequenceNo, purchase.PurchaseIDNo)
	}
	return normalizeMaterial(input)
}

func (r *Repository) CreateMaterialPurchase(ctx context.Context, input MaterialPurchaseInput) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	input.MaterialID = strings.TrimSpace(input.MaterialID)
	input.PurchasableMaterialID = strings.TrimSpace(input.PurchasableMaterialID)
	input.PurchaseSerialNo = strings.TrimSpace(input.PurchaseSerialNo)
	input.RequesterID = strings.TrimSpace(input.RequesterID)
	input.Requester = strings.TrimSpace(input.Requester)
	input.Supplier = strings.TrimSpace(input.Supplier)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.PurchaseSerialNo != "" {
		purchase, err := r.materialPurchaseBySerial(ctx, input.PurchaseSerialNo)
		if err != nil {
			return MaterialPurchase{}, err
		}
		if input.PurchasableMaterialID == "" {
			input.PurchasableMaterialID = purchase.PurchasableMaterialID
		}
		if input.MaterialID == "" {
			input.MaterialID = purchase.MaterialID
		}
		if input.RequesterID == "" {
			input.RequesterID = purchase.RequesterID
		}
		if input.Requester == "" {
			input.Requester = purchase.Requester
		}
		if input.EstimatedUnitPrice == 0 {
			input.EstimatedUnitPrice = purchase.EstimatedUnitPrice
		}
		if input.Supplier == "" {
			input.Supplier = purchase.Supplier
		}
	}
	if input.RequesterID == "" {
		input.RequesterID = strings.TrimSpace(tenant.Actor.UserID)
	}
	if input.Requester == "" {
		input.Requester = strings.TrimSpace(tenant.Actor.Name)
	}
	if (input.MaterialID == "" && input.PurchasableMaterialID == "") || input.Quantity <= 0 || input.EstimatedUnitPrice < 0 || input.Reason == "" || (input.RequesterID == "" && input.Requester == "") {
		return MaterialPurchase{}, clientError("invalid material purchase input")
	}

	var requesterID, requesterTenantID, requesterName, requesterStatus, requesterPhone, requesterEmail, groupName string
	var emailVerified bool
	var err error
	if input.RequesterID != "" {
		err = r.db.QueryRow(ctx, `
SELECT id::text, tenant_id::text, name, status, phone, email, group_name, email_verified
FROM users
WHERE id = $1 AND status <> 'deleted' AND ($2::boolean OR tenant_id = $3::uuid)
`, input.RequesterID, tenant.AllTenants, tenant.TenantID).Scan(&requesterID, &requesterTenantID, &requesterName, &requesterStatus, &requesterPhone, &requesterEmail, &groupName, &emailVerified)
	} else {
		err = r.db.QueryRow(ctx, `
SELECT id::text, tenant_id::text, name, status, phone, email, group_name, email_verified
FROM users
WHERE name = $1 AND status <> 'deleted' AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY created_at DESC
LIMIT 1
`, input.Requester, tenant.AllTenants, tenant.TenantID).Scan(&requesterID, &requesterTenantID, &requesterName, &requesterStatus, &requesterPhone, &requesterEmail, &groupName, &emailVerified)
	}
	if err != nil {
		return MaterialPurchase{}, err
	}
	if requesterStatus != "active" {
		return MaterialPurchase{}, clientError("user is not active")
	}
	if !emailVerified {
		return MaterialPurchase{}, clientError("email must be verified before purchasing materials")
	}

	var item MaterialPurchase
	var materialTenantID, materialStatus, defaultSupplier string
	if input.PurchasableMaterialID != "" {
		var purchasable PurchasableMaterial
		err := r.db.QueryRow(ctx, `
SELECT pm.id::text, pm.id_no, pm.sequence_no, COALESCE(pm.procurement_project_id::text, ''),
       COALESCE(pp.name, pm.procurement_project), COALESCE(pp.expires_at::text, ''),
       COALESCE(pp.status, 'active'),
       pm.project_name, pm.brand, pm.spec, pm.unit, pm.purchase_price::float8,
       pm.remark, pm.technical_requirement, pm.min_spec, pm.status, pm.created_at, pm.updated_at
FROM purchasable_materials pm
LEFT JOIN procurement_projects pp ON pp.id = pm.procurement_project_id
WHERE pm.id = $1
  AND pm.status = 'active'
  AND (pp.id IS NULL OR pp.status = 'active')
  AND (pp.expires_at IS NULL OR pp.expires_at >= `+appDateSQL()+`)
  AND ($2::boolean OR pm.tenant_id = $3::uuid)
`, input.PurchasableMaterialID, tenant.AllTenants, tenant.TenantID).Scan(
			&purchasable.ID, &purchasable.IDNo, &purchasable.SequenceNo, &purchasable.ProcurementProjectID, &purchasable.ProcurementProject, &purchasable.ProcurementExpiresAt, &purchasable.ProcurementProjectStatus, &purchasable.ProjectName, &purchasable.Brand, &purchasable.Spec, &purchasable.Unit, &purchasable.PurchasePrice, &purchasable.Remark, &purchasable.TechnicalRequirement, &purchasable.MinSpec, &purchasable.Status, &purchasable.CreatedAt, &purchasable.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return MaterialPurchase{}, clientError("material procurement project expired or unavailable")
			}
			return MaterialPurchase{}, err
		}
		materialTenantID = tenant.TenantID
		if requesterTenantID != materialTenantID {
			return MaterialPurchase{}, clientError("requester and purchasable material must belong to the same tenant")
		}
		item.PurchasableMaterialID = purchasable.ID
		item.PurchaseIDNo = purchasable.IDNo
		item.PurchaseSequenceNo = purchasable.SequenceNo
		item.PurchaseProjectName = firstNonEmpty(purchasable.ProcurementProject, purchasable.ProjectName)
		item.PurchaseItemName = purchasable.ProjectName
		item.PurchaseBrand = purchasable.Brand
		item.PurchaseSpec = purchasable.Spec
		item.PurchaseUnit = purchasable.Unit
		item.PurchaseRemark = purchasable.Remark
		item.PurchaseTechnicalRequirement = purchasable.TechnicalRequirement
		item.PurchaseMinSpec = purchasable.MinSpec
		if input.EstimatedUnitPrice == 0 {
			input.EstimatedUnitPrice = purchasable.PurchasePrice
		}
	} else {
		if err := r.db.QueryRow(ctx, `
SELECT tenant_id::text, status, supplier, name, name, manufacturer, spec, unit
FROM materials
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.MaterialID, tenant.AllTenants, tenant.TenantID).Scan(&materialTenantID, &materialStatus, &defaultSupplier, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit); err != nil {
			return MaterialPurchase{}, err
		}
		if requesterTenantID != materialTenantID {
			return MaterialPurchase{}, clientError("requester and material must belong to the same tenant")
		}
		if materialStatus == "disabled" {
			return MaterialPurchase{}, clientError("material is disabled")
		}
		if input.Supplier == "" {
			input.Supplier = defaultSupplier
		}
	}

	notifications := make([]Notification, 0, 1)
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	serialNo, err := r.nextMaterialPurchaseSerialNo(ctx, tx, materialTenantID)
	if err != nil {
		return MaterialPurchase{}, err
	}
	err = tx.QueryRow(ctx, `
WITH inserted AS (
  INSERT INTO material_purchases (
    tenant_id, purchase_serial_no, material_id, purchasable_material_id,
    purchase_id_no, purchase_sequence_no, purchase_project_name, purchase_item_name, purchase_brand, purchase_spec, purchase_unit,
    purchase_remark, purchase_technical_requirement, purchase_min_spec,
    requester_id, requester, requester_phone, requester_email, group_name, quantity, estimated_unit_price, supplier, reason, status, decided_at
  )
  VALUES ($1, $23, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, 'registered', now())
  RETURNING *
)
SELECT `+materialPurchaseSelectColumnsSQL+`
FROM inserted mp
LEFT JOIN materials m ON m.id = mp.material_id
`, materialTenantID, input.MaterialID, item.PurchasableMaterialID, item.PurchaseIDNo, item.PurchaseSequenceNo, item.PurchaseProjectName, item.PurchaseItemName, item.PurchaseBrand, item.PurchaseSpec, item.PurchaseUnit, item.PurchaseRemark, item.PurchaseTechnicalRequirement, item.PurchaseMinSpec, requesterID, requesterName, requesterPhone, requesterEmail, groupName, input.Quantity, input.EstimatedUnitPrice, input.Supplier, input.Reason, serialNo).Scan(
		&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt,
	)
	if err != nil {
		return MaterialPurchase{}, err
	}
	notification, err := r.createNotificationTx(ctx, tx, materialTenantID, item.RequesterID, item.GroupName, "", "group", "耗材申购登记", fmt.Sprintf("%s 登记了 %s x%d 的申购，申购流水号：%s，当前状态：%s。", item.Requester, item.MaterialName, item.Quantity, item.PurchaseSerialNo, materialWorkflowStatusLabel(item.Status)), "info")
	if err != nil {
		return MaterialPurchase{}, err
	}
	notifications = append(notifications, notification)
	if err := r.auditTx(ctx, tx, materialTenantID, item.Requester, "material_purchase.create", "material_purchase", item.ID, "", item.Status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) nextMaterialPurchaseSerialNo(ctx context.Context, tx pgx.Tx, tenantID string) (string, error) {
	month := appNow().Format("200601")
	prefix := "SG" + month + "-"
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, tenantID+":"+prefix); err != nil {
		return "", err
	}
	var maxIndex int
	if err := tx.QueryRow(ctx, `
SELECT COALESCE(MAX(NULLIF(regexp_replace(purchase_serial_no, '^SG[0-9]{6}-', ''), '')::int), 0)
FROM material_purchases
WHERE tenant_id = $1::uuid
  AND purchase_serial_no LIKE $2
  AND purchase_serial_no ~ '^SG[0-9]{6}-[0-9]{4}$'
`, tenantID, prefix+"%").Scan(&maxIndex); err != nil {
		return "", err
	}
	nextIndex := maxIndex + 1
	if nextIndex > 9999 {
		return "", clientError("material purchase serial no exhausted")
	}
	return fmt.Sprintf("%s%04d", prefix, nextIndex), nil
}

func (r *Repository) ApproveMaterialPurchase(ctx context.Context, id string, approved bool, actor string, comment string) (MaterialPurchase, error) {
	status := "rejected"
	if approved {
		status = "approved"
	}
	return r.updateMaterialPurchaseStatus(ctx, id, status, actor, comment)
}

func (r *Repository) ReturnMaterialPurchase(ctx context.Context, id string, actor string, comment string) (MaterialPurchase, error) {
	return r.updateMaterialPurchaseStatus(ctx, id, "returned", actor, comment)
}

func (r *Repository) UpdateMaterialPurchase(ctx context.Context, id string, input MaterialPurchaseUpdateInput) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	id = strings.TrimSpace(id)
	input.PurchasableMaterialID = strings.TrimSpace(input.PurchasableMaterialID)
	input.Supplier = strings.TrimSpace(input.Supplier)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = firstNonEmpty(tenant.Actor.Name, "system")
	}
	if id == "" || input.PurchasableMaterialID == "" || input.Quantity <= 0 || input.EstimatedUnitPrice < 0 || input.Reason == "" {
		return MaterialPurchase{}, clientError("invalid material purchase update input")
	}
	var purchasable PurchasableMaterial
	if err := r.db.QueryRow(ctx, `
SELECT pm.id::text, pm.id_no, pm.sequence_no, COALESCE(pm.procurement_project_id::text, ''),
       COALESCE(pp.name, pm.procurement_project), COALESCE(pp.expires_at::text, ''),
       COALESCE(pp.status, 'active'),
       pm.project_name, pm.brand, pm.spec, pm.unit, pm.purchase_price::float8,
       pm.remark, pm.technical_requirement, pm.min_spec, pm.status, pm.created_at, pm.updated_at
FROM purchasable_materials pm
LEFT JOIN procurement_projects pp ON pp.id = pm.procurement_project_id
WHERE pm.id = $1
  AND pm.status = 'active'
  AND (pp.id IS NULL OR pp.status = 'active')
  AND (pp.expires_at IS NULL OR pp.expires_at >= `+appDateSQL()+`)
  AND ($2::boolean OR pm.tenant_id = $3::uuid)
`, input.PurchasableMaterialID, tenant.AllTenants, tenant.TenantID).Scan(
		&purchasable.ID, &purchasable.IDNo, &purchasable.SequenceNo, &purchasable.ProcurementProjectID, &purchasable.ProcurementProject, &purchasable.ProcurementExpiresAt, &purchasable.ProcurementProjectStatus, &purchasable.ProjectName, &purchasable.Brand, &purchasable.Spec, &purchasable.Unit, &purchasable.PurchasePrice, &purchasable.Remark, &purchasable.TechnicalRequirement, &purchasable.MinSpec, &purchasable.Status, &purchasable.CreatedAt, &purchasable.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MaterialPurchase{}, clientError("material procurement project expired or unavailable")
		}
		return MaterialPurchase{}, err
	}
	if input.EstimatedUnitPrice == 0 {
		input.EstimatedUnitPrice = purchasable.PurchasePrice
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var item MaterialPurchase
	var itemTenantID string
	err = tx.QueryRow(ctx, `
WITH updated AS (
  UPDATE material_purchases
  SET purchasable_material_id = $2,
      purchase_id_no = $3,
      purchase_sequence_no = $4,
      purchase_project_name = $5,
      purchase_item_name = $6,
      purchase_brand = $7,
      purchase_spec = $8,
      purchase_unit = $9,
      purchase_remark = $10,
      purchase_technical_requirement = $11,
      purchase_min_spec = $12,
      quantity = $13,
      estimated_unit_price = $14,
      supplier = $15,
      reason = $16,
      status = 'registered',
      decided_at = now()
  WHERE id = $1 AND status = 'returned'
    AND ($17::boolean OR tenant_id = $18::uuid)
    AND (requester_id::text = $19 OR $20::boolean)
    AND NOT EXISTS (
        SELECT 1
        FROM material_purchase_monthly_confirmations mpmc
        WHERE mpmc.tenant_id = material_purchases.tenant_id
          AND mpmc.month = to_char(material_purchases.created_at, 'YYYY-MM')
  )
  RETURNING *
)
SELECT `+materialPurchaseSelectColumnsSQL+`, mp.tenant_id::text
FROM updated mp
LEFT JOIN materials m ON m.id = mp.material_id
`, id, purchasable.ID, purchasable.IDNo, purchasable.SequenceNo, firstNonEmpty(purchasable.ProcurementProject, purchasable.ProjectName), purchasable.ProjectName, purchasable.Brand, purchasable.Spec, purchasable.Unit, purchasable.Remark, purchasable.TechnicalRequirement, purchasable.MinSpec, input.Quantity, input.EstimatedUnitPrice, input.Supplier, input.Reason, tenant.AllTenants, tenant.TenantID, tenant.Actor.UserID, canManageMaterialsRole(tenant.Actor.Role)).Scan(
		&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt, &itemTenantID,
	)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, 'resubmit', '申请人修改后重新提交')
`, itemTenantID, item.ID, input.Actor); err != nil {
		return MaterialPurchase{}, err
	}
	if err := r.auditTx(ctx, tx, itemTenantID, input.Actor, "material_purchase.resubmit", "material_purchase", item.ID, "returned", item.Status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	return item, nil
}

func (r *Repository) MarkMaterialPurchaseOrdered(ctx context.Context, id string, actor string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	var item MaterialPurchase
	var itemTenantID string
	err = tx.QueryRow(ctx, `
WITH updated AS (
  UPDATE material_purchases
  SET status = 'ordered', ordered_at = now()
  WHERE id = $1 AND status IN ('registered', 'approved')
    AND ($2::boolean OR tenant_id = $3::uuid)
  RETURNING *
)
SELECT `+materialPurchaseSelectColumnsSQL+`, mp.tenant_id::text
FROM updated mp
LEFT JOIN materials m ON m.id = mp.material_id
	`, id, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt, &itemTenantID,
	)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, 'order', '已下单')
`, itemTenantID, item.ID, actor); err != nil {
		return MaterialPurchase{}, err
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申购状态更新", fmt.Sprintf("%s x%d 的申购状态已更新为%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialPurchase{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_purchase.order", "material_purchase", item.ID, "", item.Status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) ReceiveMaterialPurchase(ctx context.Context, id string, actor string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)

	var item MaterialPurchase
	var itemTenantID string
	var productType, defaultBatchNo, defaultExpiresAt, defaultLocation string
	err = tx.QueryRow(ctx, `
SELECT mp.id::text, mp.tenant_id::text, mp.material_id::text, m.name, COALESCE(mp.requester_id::text, ''),
       mp.requester, COALESCE(mp.requester_phone, ''), COALESCE(mp.requester_email, ''), mp.group_name, mp.quantity, mp.estimated_unit_price::float8,
       mp.supplier, mp.reason, mp.status, mp.created_at, m.product_type, m.batch_no, COALESCE(m.expires_at::text, ''),
       concat_ws(' / ', NULLIF(m.storage_room, ''), NULLIF(m.storage_cabinet, ''), NULLIF(m.storage_layer, ''), NULLIF(m.storage_slot, ''))
FROM material_purchases mp
JOIN materials m ON m.id = mp.material_id
WHERE mp.id = $1 AND mp.status IN ('registered', 'approved', 'ordered')
  AND ($2::boolean OR mp.tenant_id = $3::uuid)
FOR UPDATE
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt, &productType, &defaultBatchNo, &defaultExpiresAt, &defaultLocation)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if item.MaterialID == "" {
		return MaterialPurchase{}, clientError("material purchase has no inventory material to receive")
	}
	oldStatus := item.Status
	if productType == "standard" {
		if defaultBatchNo == "" {
			defaultBatchNo = "默认批次"
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO material_batches (tenant_id, material_id, batch_no, quantity, expires_at, location, status)
VALUES ($1, $2, $3, $4, NULLIF($5, '')::date, $6, 'active')
ON CONFLICT (tenant_id, material_id, batch_no) DO UPDATE
SET quantity = material_batches.quantity + EXCLUDED.quantity,
    expires_at = COALESCE(EXCLUDED.expires_at, material_batches.expires_at),
    location = COALESCE(NULLIF(EXCLUDED.location, ''), material_batches.location),
    status = 'active',
    updated_at = now()
`, itemTenantID, item.MaterialID, defaultBatchNo, item.Quantity, defaultExpiresAt, defaultLocation); err != nil {
			return MaterialPurchase{}, err
		}
		var batchID string
		if err := tx.QueryRow(ctx, `
SELECT id::text
FROM material_batches
WHERE tenant_id = $1::uuid AND material_id = $2 AND batch_no = $3
`, itemTenantID, item.MaterialID, defaultBatchNo).Scan(&batchID); err != nil {
			return MaterialPurchase{}, err
		}
		if err := createMaterialUnits(ctx, tx, MaterialUnitGenerationInput{
			TenantID:     itemTenantID,
			MaterialID:   item.MaterialID,
			MaterialName: item.MaterialName,
			BatchID:      batchID,
			Quantity:     item.Quantity,
			ExpiresAt:    defaultExpiresAt,
			Location:     defaultLocation,
		}); err != nil {
			return MaterialPurchase{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, batchID); err != nil {
			return MaterialPurchase{}, err
		}
		if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
			return MaterialPurchase{}, err
		}
	} else {
		if _, err := tx.Exec(ctx, `UPDATE materials SET stock = stock + $2 WHERE id = $1 AND tenant_id = $3::uuid`, item.MaterialID, item.Quantity, itemTenantID); err != nil {
			return MaterialPurchase{}, err
		}
		if err := createMaterialUnits(ctx, tx, MaterialUnitGenerationInput{
			TenantID:     itemTenantID,
			MaterialID:   item.MaterialID,
			MaterialName: item.MaterialName,
			Quantity:     item.Quantity,
			ExpiresAt:    defaultExpiresAt,
			Location:     defaultLocation,
		}); err != nil {
			return MaterialPurchase{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO inventory_ledger (tenant_id, material_id, purchase_id, change_qty, reason)
VALUES ($1, $2, $3, $4, $5)
`, itemTenantID, item.MaterialID, item.ID, item.Quantity, materialBatchReason("申购到货入库", defaultBatchNo)); err != nil {
		return MaterialPurchase{}, err
	}
	err = tx.QueryRow(ctx, `
WITH updated AS (
  UPDATE material_purchases
  SET status = 'received', received_at = now()
  WHERE id = $1 AND tenant_id = $2::uuid
  RETURNING *
)
SELECT `+materialPurchaseSelectColumnsSQL+`
FROM updated mp
LEFT JOIN materials m ON m.id = mp.material_id
	`, id, itemTenantID).Scan(&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, 'receive', '到货入库')
`, itemTenantID, item.ID, actor); err != nil {
		return MaterialPurchase{}, err
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申购状态更新", fmt.Sprintf("%s x%d 的申购状态已更新为%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialPurchase{}, err
		}
		notifications = append(notifications, notification)
	}
	created, err := r.createMaterialEventNotificationsTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "耗材申购完成入库", fmt.Sprintf("%s x%d 已完成入库，储存位置：%s。", item.MaterialName, item.Quantity, firstNonEmpty(defaultLocation, "未登记")), "success")
	if err != nil {
		return MaterialPurchase{}, err
	}
	notifications = append(notifications, created...)
	if materialNearExpiry(defaultExpiresAt, 30) {
		created, err := r.createMaterialEventNotificationsTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "耗材有效期告警", fmt.Sprintf("%s 有效期为 %s，已进入临期预警范围。", item.MaterialName, defaultExpiresAt), "warning")
		if err != nil {
			return MaterialPurchase{}, err
		}
		notifications = append(notifications, created...)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_purchase.receive", "material_purchase", item.ID, oldStatus, item.Status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) CancelMaterialPurchase(ctx context.Context, id string, actor string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	var item MaterialPurchase
	var itemTenantID string
	err = tx.QueryRow(ctx, `
WITH updated AS (
  UPDATE material_purchases
  SET status = 'cancelled'
  WHERE id = $1 AND status IN ('registered', 'approved', 'returned', 'ordered')
    AND ($2::boolean OR tenant_id = $3::uuid)
    AND NOT EXISTS (
        SELECT 1
        FROM material_purchase_monthly_confirmations mpmc
        WHERE mpmc.tenant_id = material_purchases.tenant_id
          AND mpmc.month = to_char(material_purchases.created_at, 'YYYY-MM')
  )
  RETURNING *
)
SELECT `+materialPurchaseSelectColumnsSQL+`, mp.tenant_id::text
FROM updated mp
LEFT JOIN materials m ON m.id = mp.material_id
	`, id, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt, &itemTenantID,
	)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, 'cancel', '已取消')
`, itemTenantID, item.ID, actor); err != nil {
		return MaterialPurchase{}, err
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申购状态更新", fmt.Sprintf("%s x%d 的申购状态已更新为%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialPurchase{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_purchase.cancel", "material_purchase", item.ID, "", item.Status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) updateMaterialPurchaseStatus(ctx context.Context, id string, status string, actor string, comment string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	action, ok := materialPurchaseStatusAction(status)
	if !ok {
		return MaterialPurchase{}, clientError("invalid material purchase status")
	}
	actor = strings.TrimSpace(actor)
	comment = strings.TrimSpace(comment)
	if actor == "" {
		actor = "system"
	}
	if comment == "" {
		comment = status
	}
	var item MaterialPurchase
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	var itemTenantID string
	err = tx.QueryRow(ctx, `
WITH updated AS (
  UPDATE material_purchases
  SET status = $2, decided_at = now()
  WHERE id = $1 AND status = 'registered'
    AND ($3::boolean OR tenant_id = $4::uuid)
    AND NOT EXISTS (
        SELECT 1
        FROM material_purchase_monthly_confirmations mpmc
        WHERE mpmc.tenant_id = material_purchases.tenant_id
          AND mpmc.month = to_char(material_purchases.created_at, 'YYYY-MM')
  )
  RETURNING *
)
SELECT `+materialPurchaseSelectColumnsSQL+`, mp.tenant_id::text
FROM updated mp
LEFT JOIN materials m ON m.id = mp.material_id
	`, id, status, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt, &itemTenantID)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, $4, $5)
`, itemTenantID, item.ID, actor, action, comment); err != nil {
		return MaterialPurchase{}, err
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申购状态更新", fmt.Sprintf("%s x%d 的申购状态已更新为%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialPurchase{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_purchase."+status, "material_purchase", item.ID, "registered", status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func materialPurchaseStatusAction(status string) (string, bool) {
	switch status {
	case "approved":
		return "approve", true
	case "rejected":
		return "reject", true
	case "returned":
		return "return", true
	default:
		return "", false
	}
}
