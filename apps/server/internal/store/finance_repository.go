package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) Ledger(ctx context.Context, actor Actor) ([]LedgerEntry, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT le.id::text,
       COALESCE(le.user_id::text, r.user_id::text, ''),
       COALESCE(NULLIF(le.user_name, ''), r.user_name, u.name, ''),
       COALESCE(le.reservation_id::text, ''), le.group_name, le.description,
       le.amount::float8, le.entry_type, COALESCE(le.reference_id::text, ''), le.created_at
FROM ledger_entries le
LEFT JOIN reservations r ON r.id = le.reservation_id
LEFT JOIN users u ON u.id = COALESCE(le.user_id, r.user_id)
WHERE ($1 IN ('finance_admin', 'tenant_admin', 'lab_admin', 'super_admin')
   OR COALESCE(le.user_id, r.user_id) = NULLIF($2, '')::uuid)
  AND ($3::boolean OR le.tenant_id = $4::uuid)
  AND COALESCE(le.user_id, r.user_id) IS NOT NULL
ORDER BY le.created_at DESC
LIMIT 100
`, actor.Role, actor.UserID, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]LedgerEntry, 0)
	for rows.Next() {
		var item LedgerEntry
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserName, &item.ReservationID, &item.GroupName, &item.Description, &item.Amount, &item.EntryType, &item.ReferenceID, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) AdjustLedger(ctx context.Context, input LedgerAdjustmentInput) (LedgerEntry, error) {
	tenant := TenantFromContext(ctx)
	input.UserID = strings.TrimSpace(input.UserID)
	input.UserName = strings.TrimSpace(input.UserName)
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if (input.UserID == "" && input.UserName == "") || input.Reason == "" || input.Amount == 0 {
		return LedgerEntry{}, clientError("invalid ledger adjustment input")
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return LedgerEntry{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var targetID, targetName, targetDepartment, targetGroupName, targetStatus string
	if input.UserID != "" {
		err = tx.QueryRow(ctx, `
SELECT id::text, name, department, group_name, status
FROM users
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.UserID, tenant.AllTenants, tenant.TenantID).Scan(&targetID, &targetName, &targetDepartment, &targetGroupName, &targetStatus)
	} else {
		err = tx.QueryRow(ctx, `
SELECT id::text, name, department, group_name, status
FROM users
WHERE name = $1 AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY created_at DESC
LIMIT 1
`, input.UserName, tenant.AllTenants, tenant.TenantID).Scan(&targetID, &targetName, &targetDepartment, &targetGroupName, &targetStatus)
	}
	if err != nil {
		return LedgerEntry{}, err
	}
	if targetStatus != "active" {
		return LedgerEntry{}, clientError("financial user is not active")
	}
	targetGroupName = firstNonEmpty(input.GroupName, targetGroupName, targetDepartment, "个人账户")

	var item LedgerEntry
	err = tx.QueryRow(ctx, `
INSERT INTO ledger_entries (tenant_id, user_id, user_name, group_name, description, amount, entry_type, reference_id)
VALUES ($7, $1, $2, $3, $4, $5, 'adjustment', NULLIF($6, '')::uuid)
RETURNING id::text, user_id::text, user_name, COALESCE(reservation_id::text, ''), group_name, description, amount::float8, entry_type, COALESCE(reference_id::text, ''), created_at
`, targetID, targetName, targetGroupName, "个人账务调整: "+input.Reason, input.Amount, input.OriginalEntryID, tenant.TenantID).Scan(
		&item.ID,
		&item.UserID,
		&item.UserName,
		&item.ReservationID,
		&item.GroupName,
		&item.Description,
		&item.Amount,
		&item.EntryType,
		&item.ReferenceID,
		&item.CreatedAt,
	)
	if err != nil {
		return LedgerEntry{}, err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO financial_accounts (tenant_id, user_id, user_name, group_name, balance)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, user_id) WHERE user_id IS NOT NULL
DO UPDATE SET user_name = EXCLUDED.user_name,
              group_name = EXCLUDED.group_name,
              balance = financial_accounts.balance + EXCLUDED.balance,
              updated_at = now()
`, tenant.TenantID, targetID, targetName, targetGroupName, input.Amount)
	if err != nil {
		return LedgerEntry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LedgerEntry{}, err
	}
	r.audit(ctx, input.Actor, "ledger.adjust", "ledger_entry", item.ID, input.OriginalEntryID, fmt.Sprintf("%.2f", input.Amount))
	return item, nil
}

func (r *Repository) FinancialAccounts(ctx context.Context, actor Actor) ([]FinancialAccount, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT COALESCE(fa.id::text, ''), u.id::text, u.name, u.department, u.group_name,
       COALESCE(fa.balance, 0)::float8,
       COALESCE(fa.credit_limit, 0)::float8,
       COALESCE(fa.updated_at, u.updated_at, u.created_at)
FROM users u
LEFT JOIN financial_accounts fa ON fa.tenant_id = u.tenant_id AND fa.user_id = u.id
WHERE u.status <> 'disabled'
  AND ($1::boolean OR u.tenant_id = $2::uuid)
  AND ($3 IN ('finance_admin', 'tenant_admin', 'lab_admin', 'super_admin') OR u.id = NULLIF($4, '')::uuid)
ORDER BY u.name
`, tenant.AllTenants, tenant.TenantID, actor.Role, actor.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]FinancialAccount, 0)
	for rows.Next() {
		var item FinancialAccount
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserName, &item.Department, &item.GroupName, &item.Balance, &item.CreditLimit, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveFinancialAccount(ctx context.Context, id string, input FinancialAccountInput) (FinancialAccount, error) {
	tenant := TenantFromContext(ctx)
	input.UserID = strings.TrimSpace(input.UserID)
	input.UserName = strings.TrimSpace(input.UserName)
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if (input.UserID == "" && input.UserName == "" && id == "") || input.CreditLimit < 0 {
		return FinancialAccount{}, clientError("invalid financial account input")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return FinancialAccount{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var targetID, targetName, targetDepartment, targetGroupName, targetStatus string
	var oldCredit float64
	if id != "" {
		err = tx.QueryRow(ctx, `
SELECT u.id::text, u.name, u.department, u.group_name, u.status, fa.credit_limit::float8
FROM financial_accounts fa
JOIN users u ON u.id = fa.user_id
WHERE fa.id = $1 AND ($2::boolean OR fa.tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID).Scan(&targetID, &targetName, &targetDepartment, &targetGroupName, &targetStatus, &oldCredit)
	} else if input.UserID != "" {
		err = tx.QueryRow(ctx, `
SELECT id::text, name, department, group_name, status, 0::float8
FROM users
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.UserID, tenant.AllTenants, tenant.TenantID).Scan(&targetID, &targetName, &targetDepartment, &targetGroupName, &targetStatus, &oldCredit)
	} else {
		err = tx.QueryRow(ctx, `
SELECT id::text, name, department, group_name, status, 0::float8
FROM users
WHERE name = $1 AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY created_at DESC
LIMIT 1
`, input.UserName, tenant.AllTenants, tenant.TenantID).Scan(&targetID, &targetName, &targetDepartment, &targetGroupName, &targetStatus, &oldCredit)
	}
	if err != nil {
		return FinancialAccount{}, err
	}
	if targetStatus != "active" {
		return FinancialAccount{}, clientError("financial user is not active")
	}
	targetGroupName = firstNonEmpty(input.GroupName, targetGroupName, targetDepartment, "个人账户")

	var item FinancialAccount
	if id != "" {
		err = tx.QueryRow(ctx, `
UPDATE financial_accounts
SET user_name = $2,
    group_name = $3,
    credit_limit = $4,
    updated_at = now()
WHERE id = $1 AND ($5::boolean OR tenant_id = $6::uuid)
RETURNING id::text, user_id::text, user_name, $7::text, group_name, balance::float8, credit_limit::float8, updated_at
`, id, targetName, targetGroupName, input.CreditLimit, tenant.AllTenants, tenant.TenantID, targetDepartment).Scan(&item.ID, &item.UserID, &item.UserName, &item.Department, &item.GroupName, &item.Balance, &item.CreditLimit, &item.UpdatedAt)
	} else {
		var existingID string
		existingErr := tx.QueryRow(ctx, `
SELECT id::text
FROM financial_accounts
WHERE user_id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, targetID, tenant.AllTenants, tenant.TenantID).Scan(&existingID)
		if existingErr != nil && !errors.Is(existingErr, pgx.ErrNoRows) {
			return FinancialAccount{}, existingErr
		}
		isNewAccount := errors.Is(existingErr, pgx.ErrNoRows)
		err = tx.QueryRow(ctx, `
INSERT INTO financial_accounts (tenant_id, user_id, user_name, group_name, balance, credit_limit)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, user_id) WHERE user_id IS NOT NULL
DO UPDATE SET user_name = EXCLUDED.user_name,
              group_name = EXCLUDED.group_name,
              credit_limit = EXCLUDED.credit_limit,
              updated_at = now()
RETURNING id::text, user_id::text, user_name, $7::text, group_name, balance::float8, credit_limit::float8, updated_at
`, tenant.TenantID, targetID, targetName, targetGroupName, input.InitialBalance, input.CreditLimit, targetDepartment).Scan(&item.ID, &item.UserID, &item.UserName, &item.Department, &item.GroupName, &item.Balance, &item.CreditLimit, &item.UpdatedAt)
		if err == nil && isNewAccount && input.InitialBalance != 0 {
			if _, err := tx.Exec(ctx, `
INSERT INTO ledger_entries (tenant_id, user_id, user_name, group_name, description, amount, entry_type)
VALUES ($1, $2, $3, $4, '个人账户初始余额', $5, 'account_init')
`, tenant.TenantID, targetID, targetName, targetGroupName, input.InitialBalance); err != nil {
				return FinancialAccount{}, err
			}
		}
	}
	if err != nil {
		return FinancialAccount{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FinancialAccount{}, err
	}
	r.audit(ctx, input.Actor, "financial_account.save", "financial_account", item.ID, fmt.Sprintf("%s/%.2f", targetName, oldCredit), fmt.Sprintf("%s/%.2f", item.UserName, item.CreditLimit))
	r.invalidateDashboard(ctx)
	return item, nil
}
