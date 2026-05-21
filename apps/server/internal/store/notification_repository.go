package store

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) Notifications(ctx context.Context, actor Actor) ([]Notification, error) {
	tenant := TenantFromContext(ctx)
	source := strings.TrimSpace(notificationSourceFromContext(ctx))
	rows, err := r.db.Query(ctx, `
SELECT n.id::text, n.tenant_id::text, COALESCE(t.name, ''), COALESCE(n.user_id::text, ''), n.group_name, n.department, n.target_scope,
       n.source, n.publisher, n.title, n.body, n.level, COALESCE(nr.read_at IS NOT NULL, false), n.created_at, n.updated_at
FROM notifications n
LEFT JOIN tenants t ON t.id = n.tenant_id
LEFT JOIN notification_reads nr ON nr.notification_id = n.id AND nr.user_id = NULLIF($1, '')::uuid
WHERE ($4 IN ('tenant_admin', 'lab_admin', 'super_admin')
   OR n.target_scope IN ('', 'global')
   OR (n.target_scope = 'personal' AND n.user_id = NULLIF($1, '')::uuid)
   OR (n.target_scope = 'group' AND n.group_name <> '' AND n.group_name = $2)
   OR (n.target_scope = 'department' AND n.department <> '' AND n.department = $3))
  AND ($5::boolean OR n.tenant_id = $6::uuid)
  AND ($7 = '' OR n.source = $7)
ORDER BY n.created_at DESC
LIMIT 50
`, actor.UserID, actor.GroupName, actor.Department, actor.Role, tenant.AllTenants, tenant.TenantID, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Notification, 0)
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanNotification(row scanner) (Notification, error) {
	var item Notification
	err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.TenantName,
		&item.UserID,
		&item.GroupName,
		&item.Department,
		&item.TargetScope,
		&item.Source,
		&item.Publisher,
		&item.Title,
		&item.Body,
		&item.Level,
		&item.Read,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func (r *Repository) MarkNotificationRead(ctx context.Context, id string, actor Actor) (Notification, error) {
	tenant := TenantFromContext(ctx)
	item, err := scanNotification(r.db.QueryRow(ctx, `
WITH accessible AS (
    SELECT n.*
    FROM notifications n
    WHERE n.id = $2
      AND (
        $5 IN ('tenant_admin', 'lab_admin', 'super_admin')
        OR n.target_scope IN ('', 'global')
        OR (n.target_scope = 'personal' AND n.user_id = NULLIF($1, '')::uuid)
        OR (n.target_scope = 'group' AND n.group_name <> '' AND n.group_name = $3)
        OR (n.target_scope = 'department' AND n.department <> '' AND n.department = $4)
      )
      AND ($6::boolean OR n.tenant_id = $7::uuid)
),
marked AS (
    INSERT INTO notification_reads (tenant_id, notification_id, user_id)
    SELECT tenant_id, id, NULLIF($1, '')::uuid FROM accessible
    ON CONFLICT (notification_id, user_id) DO UPDATE SET read_at = EXCLUDED.read_at
    RETURNING notification_id
)
SELECT a.id::text, a.tenant_id::text, COALESCE(t.name, ''), COALESCE(a.user_id::text, ''), a.group_name, a.department, a.target_scope,
       a.source, a.publisher, a.title, a.body, a.level, true, a.created_at, a.updated_at
FROM accessible a
LEFT JOIN tenants t ON t.id = a.tenant_id
JOIN marked m ON m.notification_id = a.id
`, actor.UserID, id, actor.GroupName, actor.Department, actor.Role, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return Notification{}, err
	}
	return item, nil
}

func (r *Repository) MarkAllNotificationsRead(ctx context.Context, actor Actor) (int, error) {
	tenant := TenantFromContext(ctx)
	source := strings.TrimSpace(notificationSourceFromContext(ctx))
	actor.UserID = strings.TrimSpace(actor.UserID)
	if actor.UserID == "" {
		return 0, clientError("user must be active")
	}
	var count int
	err := r.db.QueryRow(ctx, `
WITH accessible AS (
    SELECT n.id, n.tenant_id
    FROM notifications n
    LEFT JOIN notification_reads nr ON nr.notification_id = n.id AND nr.user_id = NULLIF($1, '')::uuid
    WHERE nr.notification_id IS NULL
      AND (
        $4 IN ('tenant_admin', 'lab_admin', 'super_admin')
        OR n.target_scope IN ('', 'global')
        OR (n.target_scope = 'personal' AND n.user_id = NULLIF($1, '')::uuid)
        OR (n.target_scope = 'group' AND n.group_name <> '' AND n.group_name = $2)
        OR (n.target_scope = 'department' AND n.department <> '' AND n.department = $3)
      )
      AND ($5::boolean OR n.tenant_id = $6::uuid)
      AND ($7 = '' OR n.source = $7)
),
marked AS (
    INSERT INTO notification_reads (tenant_id, notification_id, user_id)
    SELECT tenant_id, id, NULLIF($1, '')::uuid
    FROM accessible
    ON CONFLICT (notification_id, user_id) DO UPDATE SET read_at = EXCLUDED.read_at
    RETURNING 1
)
SELECT count(*)::int FROM marked
`, actor.UserID, actor.GroupName, actor.Department, actor.Role, tenant.AllTenants, tenant.TenantID, source).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) DeleteNotification(ctx context.Context, id string, actor string) (Notification, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	item, err := scanNotification(r.db.QueryRow(ctx, `
WITH deleted AS (
    DELETE FROM notifications n
    WHERE n.id = $1
      AND ($2::boolean OR n.tenant_id = $3::uuid)
      AND n.source = 'announcement'
    RETURNING n.*
)
SELECT d.id::text, d.tenant_id::text, COALESCE(t.name, ''), COALESCE(d.user_id::text, ''), d.group_name, d.department, d.target_scope,
       d.source, d.publisher, d.title, d.body, d.level, false, d.created_at, d.updated_at
FROM deleted d
LEFT JOIN tenants t ON t.id = d.tenant_id
`, id, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return Notification{}, err
	}
	r.audit(ctx, actor, "notification.delete", "notification", item.ID, item.Title, "deleted")
	return item, nil
}

type notificationWriter interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *Repository) createNotification(ctx context.Context, tenantID string, userID string, groupName string, department string, targetScope string, title string, body string, level string) (Notification, error) {
	item, err := r.insertNotification(ctx, r.db, tenantID, userID, groupName, department, targetScope, title, body, level, "system", "")
	if err != nil {
		return Notification{}, err
	}
	r.enqueueNotificationDelivery(item)
	return item, nil
}

func (r *Repository) createNotificationTx(ctx context.Context, tx pgx.Tx, tenantID string, userID string, groupName string, department string, targetScope string, title string, body string, level string) (Notification, error) {
	return r.insertNotification(ctx, tx, tenantID, userID, groupName, department, targetScope, title, body, level, "system", "")
}

func (r *Repository) createMaterialEventNotificationsTx(ctx context.Context, tx pgx.Tx, tenantID string, requesterID string, requesterGroup string, title string, body string, level string) ([]Notification, error) {
	seen := make(map[string]struct{})
	items := make([]Notification, 0, 4)
	requesterID = strings.TrimSpace(requesterID)
	if requesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, tenantID, requesterID, requesterGroup, "", "personal", title, body, level)
		if err != nil {
			return nil, err
		}
		items = append(items, notification)
		seen[requesterID] = struct{}{}
	}
	admins, err := r.materialAdminRecipientsTx(ctx, tx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, admin := range admins {
		if _, ok := seen[admin.ID]; ok {
			continue
		}
		notification, err := r.createNotificationTx(ctx, tx, tenantID, admin.ID, admin.GroupName, admin.Department, "personal", title, body, level)
		if err != nil {
			return nil, err
		}
		items = append(items, notification)
		seen[admin.ID] = struct{}{}
	}
	return items, nil
}

func (r *Repository) materialAdminRecipientsTx(ctx context.Context, tx pgx.Tx, tenantID string) ([]User, error) {
	rows, err := tx.Query(ctx, `
SELECT u.id::text, u.tenant_id::text, COALESCE(t.name, ''), COALESCE(t.code, ''),
       u.name, u.email, u.phone, u.department, u.group_name, u.role, u.status, u.email_verified,
       u.dingtalk_user_id, u.dingtalk_union_id, u.dingtalk_name, u.dingtalk_user_id <> '',
       COALESCE(t.finance_enabled, false), u.auth_epoch
FROM users u
LEFT JOIN tenants t ON t.id = u.tenant_id
WHERE u.tenant_id = $1::uuid
  AND u.status = 'active'
  AND u.role IN ('material_admin', 'tenant_admin', 'lab_admin', 'super_admin')
ORDER BY CASE u.role
    WHEN 'material_admin' THEN 1
    WHEN 'tenant_admin' THEN 2
    WHEN 'lab_admin' THEN 3
    ELSE 4
END, u.created_at DESC
`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]User, 0)
	for rows.Next() {
		var item User
		if err := rows.Scan(&item.ID, &item.TenantID, &item.TenantName, &item.TenantCode, &item.Name, &item.Email, &item.Phone, &item.Department, &item.GroupName, &item.Role, &item.Status, &item.EmailVerified, &item.DingTalkUserID, &item.DingTalkUnionID, &item.DingTalkName, &item.DingTalkBound, &item.FinanceEnabled, &item.AuthEpoch); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) materialRequesterForMaterialTx(ctx context.Context, tx pgx.Tx, tenantID string, materialID string) (string, string, error) {
	var requesterID, groupName string
	err := tx.QueryRow(ctx, `
SELECT COALESCE(requester_id::text, ''), group_name
FROM material_purchases
WHERE tenant_id = $1::uuid
  AND material_id = $2
  AND requester_id IS NOT NULL
ORDER BY received_at DESC NULLS LAST, created_at DESC
LIMIT 1
`, tenantID, materialID).Scan(&requesterID, &groupName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	return requesterID, groupName, err
}

func (r *Repository) insertNotification(ctx context.Context, writer notificationWriter, tenantID string, userID string, groupName string, department string, targetScope string, title string, body string, level string, source string, publisher string) (Notification, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = TenantFromContext(ctx).TenantID
	}
	userID = strings.TrimSpace(userID)
	groupName = strings.TrimSpace(groupName)
	department = strings.TrimSpace(department)
	targetScope = strings.TrimSpace(targetScope)
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	level = strings.TrimSpace(level)
	source = strings.TrimSpace(source)
	publisher = strings.TrimSpace(publisher)
	if targetScope == "" {
		targetScope = "global"
	}
	if level == "" {
		level = "info"
	}
	if source == "" {
		source = "system"
	}
	if source != "system" && source != "announcement" {
		return Notification{}, clientError("invalid notification source")
	}
	item, err := scanNotification(writer.QueryRow(ctx, `
INSERT INTO notifications (tenant_id, user_id, group_name, department, title, body, level, target_scope, source, publisher)
VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id::text, tenant_id::text, COALESCE((SELECT name FROM tenants WHERE id = notifications.tenant_id), ''),
          COALESCE(user_id::text, ''), group_name, department, target_scope, source, publisher, title, body, level, is_read, created_at, updated_at
`, tenantID, userID, groupName, department, title, body, level, targetScope, source, publisher))
	if err != nil {
		return Notification{}, err
	}
	return item, nil
}

func (r *Repository) Announce(ctx context.Context, input AnnouncementInput) (Notification, error) {
	input, err := r.resolveAnnouncementInput(ctx, input)
	if err != nil {
		return Notification{}, err
	}
	tenant := TenantFromContext(ctx)
	item, err := r.insertNotification(ctx, r.db, tenant.TenantID, input.UserID, input.GroupName, input.Department, input.TargetScope, input.Title, input.Body, input.Level, "announcement", input.Actor)
	if err != nil {
		return Notification{}, err
	}
	r.enqueueNotificationDelivery(item)
	r.audit(ctx, input.Actor, "notification.announce", "notification", item.ID, "", input.TargetScope)
	return item, nil
}

func (r *Repository) UpdateNotification(ctx context.Context, id string, input AnnouncementInput) (Notification, error) {
	tenant := TenantFromContext(ctx)
	input, err := r.resolveAnnouncementInput(ctx, input)
	if err != nil {
		return Notification{}, err
	}
	item, err := scanNotification(r.db.QueryRow(ctx, `
UPDATE notifications
SET user_id = NULLIF($2, '')::uuid,
    group_name = $3,
    department = $4,
    target_scope = $5,
    title = $6,
    body = $7,
    level = $8,
    publisher = $11,
    updated_at = now()
WHERE id = $1
  AND ($9::boolean OR tenant_id = $10::uuid)
  AND source = 'announcement'
RETURNING id::text, tenant_id::text, COALESCE((SELECT name FROM tenants WHERE id = notifications.tenant_id), ''),
          COALESCE(user_id::text, ''), group_name, department, target_scope, source, publisher, title, body, level, is_read, created_at, updated_at
`, id, input.UserID, input.GroupName, input.Department, input.TargetScope, input.Title, input.Body, input.Level, tenant.AllTenants, tenant.TenantID, input.Actor))
	if err != nil {
		return Notification{}, err
	}
	r.enqueueNotificationDelivery(item)
	r.audit(ctx, input.Actor, "notification.update", "notification", item.ID, "", input.TargetScope)
	return item, nil
}

func (r *Repository) resolveAnnouncementInput(ctx context.Context, input AnnouncementInput) (AnnouncementInput, error) {
	tenant := TenantFromContext(ctx)
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	input.Level = strings.TrimSpace(input.Level)
	input.TargetScope = strings.TrimSpace(input.TargetScope)
	input.Target = strings.TrimSpace(input.Target)
	input.UserID = strings.TrimSpace(input.UserID)
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.Department = strings.TrimSpace(input.Department)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Level == "" {
		input.Level = "info"
	}
	if input.TargetScope == "" {
		input.TargetScope = "global"
	}
	if input.Title == "" || input.Body == "" {
		return AnnouncementInput{}, clientError("invalid announcement input")
	}
	userID := input.UserID
	groupName := input.GroupName
	department := input.Department
	switch input.TargetScope {
	case "global":
		userID, groupName, department = "", "", ""
	case "personal":
		target := firstNonEmpty(input.UserID, input.Target)
		if target == "" {
			return AnnouncementInput{}, clientError("personal announcement requires a target user")
		}
		if err := r.db.QueryRow(ctx, `
SELECT id::text, group_name, department
FROM users
WHERE (id::text = $1 OR lower(email) = lower($1))
  AND ($2::boolean OR tenant_id = $3::uuid)
LIMIT 1
`, target, tenant.AllTenants, tenant.TenantID).Scan(&userID, &groupName, &department); err != nil {
			return AnnouncementInput{}, err
		}
	case "group":
		groupName = firstNonEmpty(input.GroupName, input.Target)
		if groupName == "" {
			return AnnouncementInput{}, clientError("group announcement requires a group name")
		}
		userID, department = "", ""
	case "department":
		department = firstNonEmpty(input.Department, input.Target)
		if department == "" {
			return AnnouncementInput{}, clientError("department announcement requires a department")
		}
		userID, groupName = "", ""
	default:
		return AnnouncementInput{}, clientError("invalid announcement scope")
	}
	input.UserID = userID
	input.GroupName = groupName
	input.Department = department
	return input, nil
}

func (r *Repository) enqueueNotificationDelivery(item Notification) {
	notificationDeliveryWG.Add(1)
	select {
	case notificationDeliveryQueue <- notificationDeliveryJob{repo: r, item: item}:
	default:
		notificationDeliveryWG.Done()
		slog.Warn("notification delivery queue full", "notificationId", item.ID)
	}
}

func (r *Repository) enqueueDingTalkNotifications(items ...Notification) {
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		r.enqueueNotificationDelivery(item)
	}
}
