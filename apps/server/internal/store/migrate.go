package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationSQL = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS tenants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    code text NOT NULL UNIQUE,
    finance_enabled boolean NOT NULL DEFAULT true,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO tenants (id, name, code, finance_enabled, status)
VALUES ('00000000-0000-0000-0000-000000000001', '默认单位', 'default', true, 'active')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS instruments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    category text NOT NULL,
    department text NOT NULL,
    group_name text NOT NULL DEFAULT '默认归属',
    status text NOT NULL CHECK (status IN ('available', 'busy', 'maintenance', 'disabled')),
    location text NOT NULL,
    hourly_rate numeric(10,2) NOT NULL CHECK (hourly_rate >= 0),
    brand text NOT NULL DEFAULT '',
    model text NOT NULL DEFAULT '',
    asset_code text NOT NULL DEFAULT '',
    access_control_enabled boolean NOT NULL DEFAULT false,
    access_control_group text NOT NULL DEFAULT '',
    access_control_point text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    technical_specs text NOT NULL DEFAULT '',
    booking_rule text NOT NULL DEFAULT '最小预约 1 小时；审批中时段会被锁定；使用前 2 小时可取消。',
    maintenance_summary text NOT NULL DEFAULT '',
    max_booking_hours integer NOT NULL DEFAULT 72 CHECK (max_booking_hours > 0),
    min_advance_hours integer NOT NULL DEFAULT 2 CHECK (min_advance_hours >= 0),
    cancel_cutoff_hours integer NOT NULL DEFAULT 2 CHECK (cancel_cutoff_hours >= 0),
    checkin_window_minutes integer NOT NULL DEFAULT 30 CHECK (checkin_window_minutes >= 0),
    booking_window_days integer NOT NULL DEFAULT 30 CHECK (booking_window_days > 0),
    booking_interval_hours integer NOT NULL DEFAULT 1 CHECK (booking_interval_hours > 0),
    service_start_hour integer NOT NULL DEFAULT 0 CHECK (service_start_hour >= 0 AND service_start_hour <= 23),
    service_end_hour integer NOT NULL DEFAULT 24 CHECK (service_end_hour >= 1 AND service_end_hour <= 24),
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE instruments ADD COLUMN IF NOT EXISTS group_name text NOT NULL DEFAULT '默认归属';
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS brand text NOT NULL DEFAULT '';
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS model text NOT NULL DEFAULT '';
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS asset_code text NOT NULL DEFAULT '';
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS access_control_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS access_control_group text NOT NULL DEFAULT '';
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS access_control_point text NOT NULL DEFAULT '';
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS technical_specs text NOT NULL DEFAULT '';
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS booking_rule text NOT NULL DEFAULT '最小预约 1 小时；审批中时段会被锁定；使用前 2 小时可取消。';
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS maintenance_summary text NOT NULL DEFAULT '';
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS max_booking_hours integer NOT NULL DEFAULT 72;
ALTER TABLE instruments ALTER COLUMN max_booking_hours SET DEFAULT 72;
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS min_advance_hours integer NOT NULL DEFAULT 2;
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS cancel_cutoff_hours integer NOT NULL DEFAULT 2;
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS checkin_window_minutes integer NOT NULL DEFAULT 30;
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS booking_window_days integer NOT NULL DEFAULT 30;
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS booking_interval_hours integer NOT NULL DEFAULT 1;
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS service_start_hour integer NOT NULL DEFAULT 0;
ALTER TABLE instruments ADD COLUMN IF NOT EXISTS service_end_hour integer NOT NULL DEFAULT 24;
ALTER TABLE instruments ALTER COLUMN group_name SET DEFAULT '默认归属';
CREATE UNIQUE INDEX IF NOT EXISTS instruments_asset_code_unique ON instruments (asset_code) WHERE asset_code <> '';

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    email text NOT NULL UNIQUE,
    phone text NOT NULL,
    department text NOT NULL,
    group_name text NOT NULL DEFAULT '未分配归属',
    password_hash text NOT NULL,
    role text NOT NULL DEFAULT 'unassigned',
    status text NOT NULL DEFAULT 'pending_approval' CHECK (status IN ('pending_approval', 'active', 'disabled', 'deleted')),
    email_verified boolean NOT NULL DEFAULT false,
    email_verification_token text NOT NULL DEFAULT '',
    dingtalk_user_id text NOT NULL DEFAULT '',
    dingtalk_union_id text NOT NULL DEFAULT '',
    dingtalk_name text NOT NULL DEFAULT '',
    dingtalk_bound_at timestamptz,
    auth_epoch integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS group_name text NOT NULL DEFAULT '未分配归属';
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified boolean NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_token text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS dingtalk_user_id text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS dingtalk_union_id text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS dingtalk_name text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS dingtalk_bound_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_epoch integer NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE users ALTER COLUMN group_name SET DEFAULT '未分配归属';
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;
UPDATE users SET status = 'disabled' WHERE status NOT IN ('pending_approval', 'active', 'disabled', 'deleted');
ALTER TABLE users ADD CONSTRAINT users_status_check CHECK (status IN ('pending_approval', 'active', 'disabled', 'deleted'));

CREATE TABLE IF NOT EXISTS reservations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id),
    instrument_id uuid REFERENCES instruments(id) ON DELETE SET NULL,
    user_name text NOT NULL,
    group_name text NOT NULL DEFAULT '默认归属',
    purpose text NOT NULL,
    period tstzrange NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'in_use', 'completed', 'cancelled')),
    fee numeric(10,2) NOT NULL DEFAULT 0,
    idempotency_key text,
    checked_in_at timestamptz,
    checked_out_at timestamptz,
    cancel_reason text NOT NULL DEFAULT '',
    cancelled_at timestamptz,
    decided_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (lower(period) < upper(period)),
    EXCLUDE USING gist (instrument_id WITH =, period WITH &&)
      WHERE (status IN ('pending', 'approved', 'in_use'))
);

ALTER TABLE reservations ADD COLUMN IF NOT EXISTS user_id uuid REFERENCES users(id);
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS group_name text NOT NULL DEFAULT '默认归属';
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS idempotency_key text;
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS checked_in_at timestamptz;
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS checked_out_at timestamptz;
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS cancel_reason text NOT NULL DEFAULT '';
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS cancelled_at timestamptz;
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS decided_at timestamptz;
ALTER TABLE reservations ALTER COLUMN group_name SET DEFAULT '默认归属';
CREATE UNIQUE INDEX IF NOT EXISTS reservations_idempotency_unique ON reservations (idempotency_key) WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';
CREATE INDEX IF NOT EXISTS reservations_instrument_status_idx ON reservations (instrument_id, status);
CREATE INDEX IF NOT EXISTS reservations_completed_usage_idx ON reservations (instrument_id) WHERE status = 'completed';
CREATE INDEX IF NOT EXISTS reservations_status_created_at_idx ON reservations (status, created_at);

CREATE TABLE IF NOT EXISTS notifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id),
    group_name text NOT NULL DEFAULT '',
    department text NOT NULL DEFAULT '',
    title text NOT NULL,
    body text NOT NULL,
    level text NOT NULL DEFAULT 'info',
    target_scope text NOT NULL DEFAULT 'global',
    is_read boolean NOT NULL DEFAULT false,
    read_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE notifications ADD COLUMN IF NOT EXISTS user_id uuid REFERENCES users(id);
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS group_name text NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS department text NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS target_scope text NOT NULL DEFAULT 'global';
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'system';
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS publisher text NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS read_at timestamptz;
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();
UPDATE notifications SET source = 'system' WHERE source = '';
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_source_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_source_check CHECK (source IN ('system', 'announcement'));

CREATE TABLE IF NOT EXISTS notification_reads (
    notification_id uuid NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (notification_id, user_id)
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id uuid REFERENCES reservations(id),
    user_id uuid REFERENCES users(id),
    user_name text NOT NULL DEFAULT '',
    group_name text NOT NULL,
    description text NOT NULL,
    amount numeric(10,2) NOT NULL,
    entry_type text NOT NULL DEFAULT 'debit',
    reference_id uuid,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE ledger_entries ADD COLUMN IF NOT EXISTS entry_type text NOT NULL DEFAULT 'debit';
ALTER TABLE ledger_entries ADD COLUMN IF NOT EXISTS reference_id uuid;
ALTER TABLE ledger_entries ADD COLUMN IF NOT EXISTS user_id uuid REFERENCES users(id);
ALTER TABLE ledger_entries ADD COLUMN IF NOT EXISTS user_name text NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id),
    token_hash text NOT NULL UNIQUE,
    auth_epoch integer NOT NULL DEFAULT 0,
    device_info text NOT NULL DEFAULT '',
    revoked_at timestamptz,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS auth_epoch integer NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);
CREATE INDEX IF NOT EXISTS sessions_revoked_at_idx ON sessions (revoked_at) WHERE revoked_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS financial_accounts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id),
    user_name text NOT NULL DEFAULT '',
    group_name text NOT NULL UNIQUE,
    balance numeric(12,2) NOT NULL DEFAULT 0,
    credit_limit numeric(12,2) NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS organization_units (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind text NOT NULL CHECK (kind IN ('department', 'group')),
    name text NOT NULL,
    parent_name text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (kind, name)
);

ALTER TABLE organization_units ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE organization_units ADD COLUMN IF NOT EXISTS parent_name text NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS organization_units_kind_name_unique ON organization_units (kind, name);

CREATE TABLE IF NOT EXISTS site_settings (
    setting_key text PRIMARY KEY,
    value jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_by text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE site_settings ADD COLUMN IF NOT EXISTS updated_by text NOT NULL DEFAULT '';
ALTER TABLE site_settings ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();

UPDATE site_settings
SET value = jsonb_strip_nulls(
    jsonb_build_object(
        'schemaVersion', to_jsonb(2),
        'enabled', COALESCE(value->'enabled', 'false'::jsonb),
        'clientId', to_jsonb(COALESCE(value->>'clientId', '')),
        'clientSecret', to_jsonb(COALESCE(value->>'clientSecret', '')),
        'corpId', to_jsonb(COALESCE(value->>'corpId', '')),
        'robotCode', to_jsonb(COALESCE(value->>'robotCode', '')),
        'oauthRedirectUri', to_jsonb(COALESCE(value->>'oauthRedirectUri', '')),
        'eventCallbackUrl', to_jsonb(COALESCE(value->>'eventCallbackUrl', '')),
        'eventAesKey', to_jsonb(COALESCE(value->>'eventAesKey', 'O3qwhUsprT1XONy8p7K1jhuq3O2fg7xP9kRw27b8MKq')),
        'eventToken', to_jsonb(COALESCE(value->>'eventToken', 'Hml3sD9iYksE0CtwtHPMJBPvAF'))
    )
)
WHERE setting_key = 'dingtalk';

INSERT INTO site_settings (setting_key, value, updated_by)
SELECT 'dingtalk:' || id::text, (SELECT value FROM site_settings WHERE setting_key = 'dingtalk'), 'migration'
FROM tenants
WHERE EXISTS (SELECT 1 FROM site_settings WHERE setting_key = 'dingtalk')
ON CONFLICT (setting_key) DO NOTHING;

WITH migration_marker AS (
    INSERT INTO site_settings (setting_key, value, updated_by)
    VALUES ('migration.max_booking_hours_72', '{"target":72}'::jsonb, 'migration')
    ON CONFLICT (setting_key) DO NOTHING
    RETURNING setting_key
)
UPDATE instruments
SET max_booking_hours = 72,
    booking_rule = replace(booking_rule, '最长 8 小时', '最长 72 小时')
WHERE max_booking_hours = 8
  AND EXISTS (SELECT 1 FROM migration_marker);

CREATE TABLE IF NOT EXISTS audit_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor text NOT NULL DEFAULT 'system',
    action text NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    old_value text NOT NULL DEFAULT '',
    new_value text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS approval_actions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id uuid NOT NULL REFERENCES reservations(id),
    actor text NOT NULL DEFAULT 'system',
    action text NOT NULL,
    comment text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS maintenance_orders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    instrument_id uuid REFERENCES instruments(id) ON DELETE SET NULL,
    type text NOT NULL CHECK (type IN ('routine', 'fault', 'emergency')),
    priority text NOT NULL DEFAULT 'normal',
    status text NOT NULL DEFAULT 'reported' CHECK (status IN ('reported', 'assigned', 'in_progress', 'completed', 'cancelled')),
    handler text NOT NULL DEFAULT '',
    description text NOT NULL,
    result text NOT NULL DEFAULT '',
    period tstzrange NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (lower(period) < upper(period))
);

CREATE TABLE IF NOT EXISTS materials (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    product_type text NOT NULL DEFAULT 'consumable',
    category text NOT NULL,
    subcategory text NOT NULL DEFAULT '',
    spec text NOT NULL,
    unit text NOT NULL,
    unit_price numeric(10,2) NOT NULL DEFAULT 0,
    stock integer NOT NULL DEFAULT 0 CHECK (stock >= 0),
    warning_line integer NOT NULL DEFAULT 0 CHECK (warning_line >= 0),
    supplier text NOT NULL DEFAULT '',
    manufacturer text NOT NULL DEFAULT '',
    batch_no text NOT NULL DEFAULT '',
    catalog_no text NOT NULL DEFAULT '',
    cas_no text NOT NULL DEFAULT '',
    grade text NOT NULL DEFAULT '',
    concentration text NOT NULL DEFAULT '',
    parent_material_id uuid REFERENCES materials(id),
    dilution_factor text NOT NULL DEFAULT '',
    preparation_method text NOT NULL DEFAULT '',
    storage_condition text NOT NULL DEFAULT '',
    storage_room text NOT NULL DEFAULT '',
    storage_cabinet text NOT NULL DEFAULT '',
    storage_layer text NOT NULL DEFAULT '',
    storage_slot text NOT NULL DEFAULT '',
    tender_contract text NOT NULL DEFAULT '',
    contract_no text NOT NULL DEFAULT '',
    remark text NOT NULL DEFAULT '',
    certificate_url text NOT NULL DEFAULT '',
    standard_certificate_url text NOT NULL DEFAULT '',
    attachment_url text NOT NULL DEFAULT '',
    qr_code text NOT NULL DEFAULT '',
    expires_at date,
    opened_at date,
    open_expire_days integer NOT NULL DEFAULT 0 CHECK (open_expire_days >= 0),
    freeze_thaw_count integer NOT NULL DEFAULT 0 CHECK (freeze_thaw_count >= 0),
    freeze_thaw_limit integer NOT NULL DEFAULT 0 CHECK (freeze_thaw_limit >= 0),
    approval_required boolean NOT NULL DEFAULT false,
    near_expiry_days integer NOT NULL DEFAULT 30 CHECK (near_expiry_days >= 0),
    status text NOT NULL DEFAULT 'normal' CHECK (status IN ('normal', 'near_expiry', 'low', 'expired', 'open_expired', 'freeze_thaw_exceeded', 'damaged', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE materials ADD COLUMN IF NOT EXISTS product_type text NOT NULL DEFAULT 'consumable';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS subcategory text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS manufacturer text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS cas_no text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS grade text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS concentration text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS parent_material_id uuid REFERENCES materials(id);
ALTER TABLE materials ADD COLUMN IF NOT EXISTS dilution_factor text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS preparation_method text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS storage_condition text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS storage_room text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS storage_cabinet text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS storage_layer text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS storage_slot text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS remark text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS certificate_url text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS standard_certificate_url text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS attachment_url text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS qr_code text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS opened_at date;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS open_expire_days integer NOT NULL DEFAULT 0;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS freeze_thaw_count integer NOT NULL DEFAULT 0;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS freeze_thaw_limit integer NOT NULL DEFAULT 0;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS approval_required boolean NOT NULL DEFAULT false;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS near_expiry_days integer NOT NULL DEFAULT 30;
UPDATE materials SET status = 'normal' WHERE status NOT IN ('normal', 'near_expiry', 'low', 'expired', 'open_expired', 'freeze_thaw_exceeded', 'damaged', 'disabled');
ALTER TABLE materials DROP CONSTRAINT IF EXISTS materials_status_check;
ALTER TABLE materials ADD CONSTRAINT materials_status_check CHECK (status IN ('normal', 'near_expiry', 'low', 'expired', 'open_expired', 'freeze_thaw_exceeded', 'damaged', 'disabled'));
ALTER TABLE materials DROP CONSTRAINT IF EXISTS materials_product_type_check;
UPDATE materials SET product_type = 'standard' WHERE product_type IN ('working_solution', 'mixed_standard') OR category LIKE '%标准%';
ALTER TABLE materials ADD CONSTRAINT materials_product_type_check CHECK (product_type IN ('consumable', 'reagent', 'standard'));
ALTER TABLE materials DROP CONSTRAINT IF EXISTS materials_open_expire_days_check;
ALTER TABLE materials ADD CONSTRAINT materials_open_expire_days_check CHECK (open_expire_days >= 0);
ALTER TABLE materials DROP CONSTRAINT IF EXISTS materials_freeze_thaw_count_check;
ALTER TABLE materials ADD CONSTRAINT materials_freeze_thaw_count_check CHECK (freeze_thaw_count >= 0);
ALTER TABLE materials DROP CONSTRAINT IF EXISTS materials_freeze_thaw_limit_check;
ALTER TABLE materials ADD CONSTRAINT materials_freeze_thaw_limit_check CHECK (freeze_thaw_limit >= 0);
ALTER TABLE materials DROP CONSTRAINT IF EXISTS materials_near_expiry_days_check;
ALTER TABLE materials ADD CONSTRAINT materials_near_expiry_days_check CHECK (near_expiry_days >= 0);
UPDATE materials SET product_type = 'reagent' WHERE product_type = 'consumable' AND category LIKE '%试剂%';

CREATE TABLE IF NOT EXISTS material_categories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    name text NOT NULL,
    parent_name text NOT NULL DEFAULT '',
    display_order integer NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE IF NOT EXISTS material_alert_actions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    material_id uuid NOT NULL REFERENCES materials(id),
    alert_type text NOT NULL,
    action text NOT NULL CHECK (action IN ('handled', 'ignored')),
    comment text NOT NULL DEFAULT '',
    actor text NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS material_alert_actions_material_created_at_idx ON material_alert_actions (material_id, created_at DESC);
CREATE INDEX IF NOT EXISTS material_alert_actions_tenant_created_at_idx ON material_alert_actions (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS procurement_projects (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    name text NOT NULL,
    expires_at date,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS procurement_projects_tenant_status_idx ON procurement_projects (tenant_id, status, expires_at);

CREATE TABLE IF NOT EXISTS purchasable_materials (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    id_no text NOT NULL,
    sequence_no text NOT NULL,
    procurement_project_id uuid REFERENCES procurement_projects(id),
    procurement_project text NOT NULL DEFAULT '',
    project_name text NOT NULL,
    brand text NOT NULL,
    spec text NOT NULL,
    unit text NOT NULL,
    purchase_price numeric(12,2) NOT NULL CHECK (purchase_price >= 0),
    remark text NOT NULL DEFAULT '',
    technical_requirement text NOT NULL DEFAULT '',
    min_spec text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'deleted')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id_no)
);
ALTER TABLE purchasable_materials ADD COLUMN IF NOT EXISTS procurement_project_id uuid REFERENCES procurement_projects(id);
ALTER TABLE purchasable_materials ADD COLUMN IF NOT EXISTS procurement_project text NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS purchasable_materials_tenant_status_idx ON purchasable_materials (tenant_id, status, project_name, sequence_no);
INSERT INTO procurement_projects (tenant_id, name)
SELECT DISTINCT tenant_id, procurement_project
FROM purchasable_materials
WHERE procurement_project <> ''
ON CONFLICT (tenant_id, name) DO NOTHING;
UPDATE purchasable_materials pm
SET procurement_project_id = pp.id
FROM procurement_projects pp
WHERE pm.procurement_project <> ''
  AND pm.procurement_project_id IS NULL
  AND pp.tenant_id = pm.tenant_id
  AND pp.name = pm.procurement_project;

CREATE TABLE IF NOT EXISTS material_batches (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    material_id uuid NOT NULL REFERENCES materials(id),
    batch_no text NOT NULL,
    quantity integer NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    expires_at date,
    location text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'depleted', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, material_id, batch_no)
);
CREATE INDEX IF NOT EXISTS material_batches_material_status_idx ON material_batches (material_id, status, expires_at);

CREATE TABLE IF NOT EXISTS material_units (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    material_id uuid NOT NULL REFERENCES materials(id),
    batch_id uuid REFERENCES material_batches(id),
    unit_code text NOT NULL,
    expires_at date,
    location text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'reserved', 'used', 'damaged', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, unit_code)
);
CREATE INDEX IF NOT EXISTS material_units_material_status_idx ON material_units (material_id, status, expires_at);
CREATE INDEX IF NOT EXISTS material_units_batch_status_idx ON material_units (batch_id, status);

CREATE TABLE IF NOT EXISTS material_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id uuid NOT NULL REFERENCES materials(id),
    requester_id uuid REFERENCES users(id),
    requester text NOT NULL,
    group_name text NOT NULL DEFAULT '默认归属',
    batch_id uuid REFERENCES material_batches(id),
    unit_id uuid REFERENCES material_units(id),
    quantity integer NOT NULL CHECK (quantity > 0),
    purpose text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'outbound', 'cancelled')),
    decided_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE material_requests ADD COLUMN IF NOT EXISTS requester_id uuid REFERENCES users(id);
ALTER TABLE material_requests ADD COLUMN IF NOT EXISTS group_name text NOT NULL DEFAULT '默认归属';
ALTER TABLE material_requests ADD COLUMN IF NOT EXISTS batch_id uuid REFERENCES material_batches(id);
ALTER TABLE material_requests ADD COLUMN IF NOT EXISTS unit_id uuid REFERENCES material_units(id);
ALTER TABLE material_requests ADD COLUMN IF NOT EXISTS decided_at timestamptz;
ALTER TABLE material_requests ALTER COLUMN group_name SET DEFAULT '默认归属';

CREATE TABLE IF NOT EXISTS material_purchases (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id uuid REFERENCES materials(id),
    purchasable_material_id uuid REFERENCES purchasable_materials(id),
    purchase_id_no text NOT NULL DEFAULT '',
    purchase_sequence_no text NOT NULL DEFAULT '',
    purchase_project_name text NOT NULL DEFAULT '',
    purchase_item_name text NOT NULL DEFAULT '',
    purchase_brand text NOT NULL DEFAULT '',
    purchase_spec text NOT NULL DEFAULT '',
    purchase_unit text NOT NULL DEFAULT '',
    purchase_remark text NOT NULL DEFAULT '',
    purchase_technical_requirement text NOT NULL DEFAULT '',
    purchase_min_spec text NOT NULL DEFAULT '',
    requester_id uuid REFERENCES users(id),
    requester text NOT NULL,
    group_name text NOT NULL DEFAULT '默认归属',
    quantity integer NOT NULL CHECK (quantity > 0),
    estimated_unit_price numeric(12,2) NOT NULL DEFAULT 0 CHECK (estimated_unit_price >= 0),
    supplier text NOT NULL DEFAULT '',
    reason text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'ordered', 'received', 'cancelled')),
    decided_at timestamptz,
    ordered_at timestamptz,
    received_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE material_purchases ALTER COLUMN material_id DROP NOT NULL;
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchasable_material_id uuid REFERENCES purchasable_materials(id);
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_id_no text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_sequence_no text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_project_name text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_item_name text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_brand text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_spec text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_unit text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_remark text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_technical_requirement text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_min_spec text NOT NULL DEFAULT '';
UPDATE material_purchases mp
SET purchase_project_name = COALESCE(NULLIF(purchase_project_name, ''), m.name),
    purchase_item_name = COALESCE(NULLIF(purchase_item_name, ''), NULLIF(purchase_project_name, ''), m.name),
    purchase_brand = COALESCE(NULLIF(purchase_brand, ''), m.manufacturer, ''),
    purchase_spec = COALESCE(NULLIF(purchase_spec, ''), m.spec),
    purchase_unit = COALESCE(NULLIF(purchase_unit, ''), m.unit)
FROM materials m
WHERE mp.material_id = m.id
  AND (mp.purchase_project_name = '' OR mp.purchase_item_name = '' OR mp.purchase_spec = '' OR mp.purchase_unit = '');
UPDATE material_purchases mp
SET purchase_project_name = CASE
        WHEN pm.procurement_project <> '' AND (mp.purchase_project_name = '' OR mp.purchase_project_name = pm.project_name) THEN pm.procurement_project
        ELSE COALESCE(NULLIF(mp.purchase_project_name, ''), NULLIF(pm.procurement_project, ''), pm.project_name)
    END,
    purchase_item_name = COALESCE(NULLIF(mp.purchase_item_name, ''), pm.project_name)
FROM purchasable_materials pm
WHERE mp.purchasable_material_id = pm.id
  AND (mp.purchase_project_name = '' OR mp.purchase_item_name = '' OR mp.purchase_project_name = pm.project_name);
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS requester_id uuid REFERENCES users(id);
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS requester_phone text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS requester_email text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS group_name text NOT NULL DEFAULT '默认归属';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS estimated_unit_price numeric(12,2) NOT NULL DEFAULT 0;
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS supplier text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS decided_at timestamptz;
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS ordered_at timestamptz;
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS received_at timestamptz;
ALTER TABLE material_purchases ALTER COLUMN group_name SET DEFAULT '默认归属';
CREATE INDEX IF NOT EXISTS material_purchases_status_created_at_idx ON material_purchases (status, created_at DESC);
CREATE INDEX IF NOT EXISTS material_purchases_group_status_idx ON material_purchases (group_name, status);

CREATE TABLE IF NOT EXISTS inventory_ledger (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id uuid NOT NULL REFERENCES materials(id),
    request_id uuid REFERENCES material_requests(id),
    purchase_id uuid REFERENCES material_purchases(id),
    change_qty integer NOT NULL,
    reason text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE inventory_ledger ADD COLUMN IF NOT EXISTS purchase_id uuid REFERENCES material_purchases(id);

CREATE TABLE IF NOT EXISTS material_damage_reports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    material_id uuid NOT NULL REFERENCES materials(id),
    reporter_id uuid REFERENCES users(id),
    reporter text NOT NULL,
    group_name text NOT NULL DEFAULT '默认归属',
    quantity integer NOT NULL CHECK (quantity > 0),
    reason text NOT NULL,
    photo_url text NOT NULL DEFAULT '',
    attachment_url text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'processed', 'cancelled')),
    reviewer text NOT NULL DEFAULT '',
    review_comment text NOT NULL DEFAULT '',
    reviewed_at timestamptz,
    processed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS material_damage_reports_status_created_at_idx ON material_damage_reports (status, created_at DESC);
CREATE INDEX IF NOT EXISTS material_damage_reports_group_status_idx ON material_damage_reports (group_name, status);
ALTER TABLE material_damage_reports ADD COLUMN IF NOT EXISTS batch_id uuid REFERENCES material_batches(id);
ALTER TABLE material_damage_reports ADD COLUMN IF NOT EXISTS unit_id uuid REFERENCES material_units(id);
CREATE INDEX IF NOT EXISTS material_damage_reports_unit_idx ON material_damage_reports (unit_id);

ALTER TABLE inventory_ledger ADD COLUMN IF NOT EXISTS damage_id uuid REFERENCES material_damage_reports(id);

CREATE TABLE IF NOT EXISTS material_approval_actions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    material_request_id uuid NOT NULL REFERENCES material_requests(id),
    actor text NOT NULL DEFAULT 'system',
    action text NOT NULL,
    comment text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS material_purchase_actions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    material_purchase_id uuid NOT NULL REFERENCES material_purchases(id),
    actor text NOT NULL DEFAULT 'system',
    action text NOT NULL,
    comment text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

DROP TRIGGER IF EXISTS ledger_entries_immutable_update ON ledger_entries;
DROP TRIGGER IF EXISTS approval_actions_immutable_update ON approval_actions;
DROP TRIGGER IF EXISTS material_approval_actions_immutable_update ON material_approval_actions;
DROP TRIGGER IF EXISTS material_purchase_actions_immutable_update ON material_purchase_actions;

ALTER TABLE ledger_entries ADD COLUMN IF NOT EXISTS user_id uuid REFERENCES users(id);
ALTER TABLE ledger_entries ADD COLUMN IF NOT EXISTS user_name text NOT NULL DEFAULT '';
UPDATE ledger_entries le
SET user_id = r.user_id,
    user_name = r.user_name
FROM reservations r
WHERE le.reservation_id = r.id
  AND le.user_id IS NULL;

ALTER TABLE financial_accounts ADD COLUMN IF NOT EXISTS user_id uuid REFERENCES users(id);
ALTER TABLE financial_accounts ADD COLUMN IF NOT EXISTS user_name text NOT NULL DEFAULT '';

ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE users SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE users ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE users ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
DROP INDEX IF EXISTS users_tenant_email_unique;
CREATE UNIQUE INDEX IF NOT EXISTS users_tenant_email_unique ON users (tenant_id, lower(email));
CREATE UNIQUE INDEX IF NOT EXISTS users_tenant_dingtalk_user_unique ON users (tenant_id, dingtalk_user_id) WHERE dingtalk_user_id <> '';

ALTER TABLE instruments ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE instruments SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE instruments ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE instruments ALTER COLUMN tenant_id SET NOT NULL;
DROP INDEX IF EXISTS instruments_asset_code_unique;
CREATE UNIQUE INDEX IF NOT EXISTS instruments_tenant_asset_code_unique ON instruments (tenant_id, asset_code) WHERE asset_code <> '';

ALTER TABLE reservations ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE reservations r SET tenant_id = i.tenant_id FROM instruments i WHERE r.instrument_id = i.id AND r.tenant_id IS NULL;
UPDATE reservations SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE reservations ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE reservations ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE reservations ALTER COLUMN instrument_id DROP NOT NULL;
ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_instrument_id_fkey;
ALTER TABLE reservations ADD CONSTRAINT reservations_instrument_id_fkey FOREIGN KEY (instrument_id) REFERENCES instruments(id) ON DELETE SET NULL;

ALTER TABLE notifications ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'system';
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS publisher text NOT NULL DEFAULT '';
UPDATE notifications SET source = 'system' WHERE source = '';
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_source_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_source_check CHECK (source IN ('system', 'announcement'));
UPDATE notifications SET updated_at = created_at WHERE updated_at IS NULL;
UPDATE notifications SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE notifications ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE notifications ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE notification_reads ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE notification_reads nr SET tenant_id = u.tenant_id FROM users u WHERE nr.user_id = u.id AND nr.tenant_id IS NULL;
UPDATE notification_reads SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE notification_reads ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE notification_reads ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE ledger_entries ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE ledger_entries le SET tenant_id = r.tenant_id FROM reservations r WHERE le.reservation_id = r.id AND le.tenant_id IS NULL;
UPDATE ledger_entries SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE ledger_entries ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE ledger_entries ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE financial_accounts ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE financial_accounts SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE financial_accounts ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE financial_accounts ALTER COLUMN tenant_id SET NOT NULL;
UPDATE financial_accounts fa
SET user_id = u.id,
    user_name = u.name
FROM users u
WHERE fa.user_id IS NULL
  AND fa.tenant_id = u.tenant_id
  AND fa.group_name = u.group_name;
ALTER TABLE financial_accounts DROP CONSTRAINT IF EXISTS financial_accounts_group_name_key;
DROP INDEX IF EXISTS financial_accounts_tenant_group_unique;
CREATE UNIQUE INDEX IF NOT EXISTS financial_accounts_tenant_user_unique ON financial_accounts (tenant_id, user_id) WHERE user_id IS NOT NULL;

ALTER TABLE organization_units ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE organization_units SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE organization_units ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE organization_units ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE organization_units ADD COLUMN IF NOT EXISTS parent_name text NOT NULL DEFAULT '';
ALTER TABLE organization_units DROP CONSTRAINT IF EXISTS organization_units_kind_name_key;
DROP INDEX IF EXISTS organization_units_kind_name_unique;
CREATE UNIQUE INDEX IF NOT EXISTS organization_units_tenant_kind_name_unique ON organization_units (tenant_id, kind, name);

WITH instrument_team_parent AS (
    SELECT tenant_id, group_name, min(department) AS parent_name
    FROM instruments
    WHERE group_name <> ''
    GROUP BY tenant_id, group_name
    HAVING count(DISTINCT department) = 1
)
UPDATE organization_units item
SET parent_name = instrument_team_parent.parent_name,
    updated_at = now()
FROM instrument_team_parent
WHERE item.kind = 'group'
  AND item.tenant_id = instrument_team_parent.tenant_id
  AND item.name = instrument_team_parent.group_name
  AND item.parent_name = '';

WITH team_parent(name, parent_name) AS (
    VALUES
        ('李明课题组', '化学与分子工程学院'),
        ('李明团队', '化学与分子工程学院'),
        ('王敏课题组', '生命科学学院'),
        ('王敏团队', '生命科学学院'),
        ('先进材料课题组', '物理学院'),
        ('先进材料平台', '物理学院'),
        ('免疫工程课题组', '医学院'),
        ('免疫工程平台', '医学院'),
        ('系统管理组', '系统管理'),
        ('默认课题组', '系统管理'),
        ('默认归属', '系统管理'),
        ('未分配课题组', '系统管理'),
        ('未分配归属', '系统管理')
)
UPDATE organization_units item
SET parent_name = team_parent.parent_name,
    updated_at = now()
FROM team_parent
WHERE item.kind = 'group'
  AND item.name = team_parent.name
  AND item.parent_name = '';

WITH ownership_renames(old_name, new_name) AS (
    VALUES
        ('李明课题组', '李明团队'),
        ('王敏课题组', '王敏团队'),
        ('先进材料课题组', '先进材料平台'),
        ('免疫工程课题组', '免疫工程平台'),
        ('默认课题组', '默认归属'),
        ('未分配课题组', '未分配归属')
)
UPDATE instruments item
SET group_name = ownership_renames.new_name
FROM ownership_renames
WHERE item.group_name = ownership_renames.old_name;

WITH ownership_renames(old_name, new_name) AS (
    VALUES
        ('李明课题组', '李明团队'),
        ('王敏课题组', '王敏团队'),
        ('先进材料课题组', '先进材料平台'),
        ('免疫工程课题组', '免疫工程平台'),
        ('默认课题组', '默认归属'),
        ('未分配课题组', '未分配归属')
)
UPDATE users item
SET group_name = ownership_renames.new_name
FROM ownership_renames
WHERE item.group_name = ownership_renames.old_name;

WITH ownership_renames(old_name, new_name) AS (
    VALUES
        ('李明课题组', '李明团队'),
        ('王敏课题组', '王敏团队'),
        ('先进材料课题组', '先进材料平台'),
        ('免疫工程课题组', '免疫工程平台'),
        ('默认课题组', '默认归属'),
        ('未分配课题组', '未分配归属')
)
UPDATE reservations item
SET group_name = ownership_renames.new_name
FROM ownership_renames
WHERE item.group_name = ownership_renames.old_name;

WITH ownership_renames(old_name, new_name) AS (
    VALUES
        ('李明课题组', '李明团队'),
        ('王敏课题组', '王敏团队'),
        ('先进材料课题组', '先进材料平台'),
        ('免疫工程课题组', '免疫工程平台'),
        ('默认课题组', '默认归属'),
        ('未分配课题组', '未分配归属')
)
UPDATE notifications item
SET group_name = ownership_renames.new_name
FROM ownership_renames
WHERE item.group_name = ownership_renames.old_name;

UPDATE notifications
SET body = replace(body, '分配角色和课题组', '分配角色')
WHERE body LIKE '%分配角色和课题组%';

WITH ownership_renames(old_name, new_name) AS (
    VALUES
        ('李明课题组', '李明团队'),
        ('王敏课题组', '王敏团队'),
        ('先进材料课题组', '先进材料平台'),
        ('免疫工程课题组', '免疫工程平台'),
        ('默认课题组', '默认归属'),
        ('未分配课题组', '未分配归属')
)
UPDATE ledger_entries item
SET group_name = ownership_renames.new_name
FROM ownership_renames
WHERE item.group_name = ownership_renames.old_name;

WITH ownership_renames(old_name, new_name) AS (
    VALUES
        ('李明课题组', '李明团队'),
        ('王敏课题组', '王敏团队'),
        ('先进材料课题组', '先进材料平台'),
        ('免疫工程课题组', '免疫工程平台'),
        ('默认课题组', '默认归属'),
        ('未分配课题组', '未分配归属')
)
UPDATE financial_accounts item
SET group_name = ownership_renames.new_name
FROM ownership_renames
WHERE item.group_name = ownership_renames.old_name;

WITH ownership_renames(old_name, new_name) AS (
    VALUES
        ('李明课题组', '李明团队'),
        ('王敏课题组', '王敏团队'),
        ('先进材料课题组', '先进材料平台'),
        ('免疫工程课题组', '免疫工程平台'),
        ('默认课题组', '默认归属'),
        ('未分配课题组', '未分配归属')
)
UPDATE material_requests item
SET group_name = ownership_renames.new_name
FROM ownership_renames
WHERE item.group_name = ownership_renames.old_name;

WITH ownership_renames(old_name, new_name) AS (
    VALUES
        ('李明课题组', '李明团队'),
        ('王敏课题组', '王敏团队'),
        ('先进材料课题组', '先进材料平台'),
        ('免疫工程课题组', '免疫工程平台'),
        ('默认课题组', '默认归属'),
        ('未分配课题组', '未分配归属')
)
UPDATE material_purchases item
SET group_name = ownership_renames.new_name
FROM ownership_renames
WHERE item.group_name = ownership_renames.old_name;

WITH ownership_renames(old_name, new_name) AS (
    VALUES
        ('李明课题组', '李明团队'),
        ('王敏课题组', '王敏团队'),
        ('先进材料课题组', '先进材料平台'),
        ('免疫工程课题组', '免疫工程平台'),
        ('默认课题组', '默认归属'),
        ('未分配课题组', '未分配归属')
)
DELETE FROM organization_units old_unit
USING organization_units new_unit, ownership_renames
WHERE old_unit.kind = 'group'
  AND old_unit.name = ownership_renames.old_name
  AND new_unit.tenant_id = old_unit.tenant_id
  AND new_unit.kind = old_unit.kind
  AND new_unit.name = ownership_renames.new_name;

WITH ownership_renames(old_name, new_name) AS (
    VALUES
        ('李明课题组', '李明团队'),
        ('王敏课题组', '王敏团队'),
        ('先进材料课题组', '先进材料平台'),
        ('免疫工程课题组', '免疫工程平台'),
        ('默认课题组', '默认归属'),
        ('未分配课题组', '未分配归属')
)
UPDATE organization_units item
SET name = ownership_renames.new_name,
    updated_at = now()
FROM ownership_renames
WHERE item.kind = 'group'
  AND item.name = ownership_renames.old_name;

ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE audit_events SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE audit_events ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE audit_events ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE approval_actions ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE approval_actions aa SET tenant_id = r.tenant_id FROM reservations r WHERE aa.reservation_id = r.id AND aa.tenant_id IS NULL;
UPDATE approval_actions SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE approval_actions ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE approval_actions ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE maintenance_orders ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE maintenance_orders mo SET tenant_id = i.tenant_id FROM instruments i WHERE mo.instrument_id = i.id AND mo.tenant_id IS NULL;
UPDATE maintenance_orders SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE maintenance_orders ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE maintenance_orders ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE maintenance_orders ALTER COLUMN instrument_id DROP NOT NULL;
ALTER TABLE maintenance_orders DROP CONSTRAINT IF EXISTS maintenance_orders_instrument_id_fkey;
ALTER TABLE maintenance_orders ADD CONSTRAINT maintenance_orders_instrument_id_fkey FOREIGN KEY (instrument_id) REFERENCES instruments(id) ON DELETE SET NULL;

ALTER TABLE materials ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
ALTER TABLE materials ADD COLUMN IF NOT EXISTS catalog_no text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS tender_contract text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS contract_no text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS remark text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS product_type text NOT NULL DEFAULT 'consumable';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS subcategory text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS manufacturer text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS cas_no text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS grade text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS concentration text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS parent_material_id uuid REFERENCES materials(id);
ALTER TABLE materials ADD COLUMN IF NOT EXISTS dilution_factor text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS preparation_method text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS storage_condition text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS storage_room text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS storage_cabinet text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS storage_layer text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS storage_slot text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS certificate_url text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS standard_certificate_url text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS attachment_url text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS qr_code text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS opened_at date;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS open_expire_days integer NOT NULL DEFAULT 0;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS freeze_thaw_count integer NOT NULL DEFAULT 0;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS freeze_thaw_limit integer NOT NULL DEFAULT 0;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS approval_required boolean NOT NULL DEFAULT false;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS near_expiry_days integer NOT NULL DEFAULT 30;
UPDATE materials SET status = 'normal' WHERE status NOT IN ('normal', 'near_expiry', 'low', 'expired', 'open_expired', 'freeze_thaw_exceeded', 'damaged', 'disabled');
ALTER TABLE materials DROP CONSTRAINT IF EXISTS materials_status_check;
ALTER TABLE materials ADD CONSTRAINT materials_status_check CHECK (status IN ('normal', 'near_expiry', 'low', 'expired', 'open_expired', 'freeze_thaw_exceeded', 'damaged', 'disabled'));
ALTER TABLE materials DROP CONSTRAINT IF EXISTS materials_product_type_check;
UPDATE materials SET product_type = 'standard' WHERE product_type IN ('working_solution', 'mixed_standard') OR category LIKE '%标准%';
ALTER TABLE materials ADD CONSTRAINT materials_product_type_check CHECK (product_type IN ('consumable', 'reagent', 'standard'));
UPDATE materials SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE materials ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE materials ALTER COLUMN tenant_id SET NOT NULL;

CREATE TABLE IF NOT EXISTS material_categories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    name text NOT NULL,
    parent_name text NOT NULL DEFAULT '',
    display_order integer NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE IF NOT EXISTS material_alert_actions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    material_id uuid NOT NULL REFERENCES materials(id),
    alert_type text NOT NULL,
    action text NOT NULL CHECK (action IN ('handled', 'ignored')),
    comment text NOT NULL DEFAULT '',
    actor text NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS material_alert_actions_material_created_at_idx ON material_alert_actions (material_id, created_at DESC);
CREATE INDEX IF NOT EXISTS material_alert_actions_tenant_created_at_idx ON material_alert_actions (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS procurement_projects (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    name text NOT NULL,
    expires_at date,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS procurement_projects_tenant_status_idx ON procurement_projects (tenant_id, status, expires_at);

CREATE TABLE IF NOT EXISTS purchasable_materials (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    id_no text NOT NULL,
    sequence_no text NOT NULL,
    procurement_project_id uuid REFERENCES procurement_projects(id),
    procurement_project text NOT NULL DEFAULT '',
    project_name text NOT NULL,
    brand text NOT NULL,
    spec text NOT NULL,
    unit text NOT NULL,
    purchase_price numeric(12,2) NOT NULL CHECK (purchase_price >= 0),
    remark text NOT NULL DEFAULT '',
    technical_requirement text NOT NULL DEFAULT '',
    min_spec text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'deleted')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id_no)
);
ALTER TABLE purchasable_materials ADD COLUMN IF NOT EXISTS procurement_project_id uuid REFERENCES procurement_projects(id);
ALTER TABLE purchasable_materials ADD COLUMN IF NOT EXISTS procurement_project text NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS purchasable_materials_tenant_status_idx ON purchasable_materials (tenant_id, status, project_name, sequence_no);
INSERT INTO procurement_projects (tenant_id, name)
SELECT DISTINCT tenant_id, procurement_project
FROM purchasable_materials
WHERE procurement_project <> ''
ON CONFLICT (tenant_id, name) DO NOTHING;
UPDATE purchasable_materials pm
SET procurement_project_id = pp.id
FROM procurement_projects pp
WHERE pm.procurement_project <> ''
  AND pm.procurement_project_id IS NULL
  AND pp.tenant_id = pm.tenant_id
  AND pp.name = pm.procurement_project;

CREATE TABLE IF NOT EXISTS material_batches (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    material_id uuid NOT NULL REFERENCES materials(id),
    batch_no text NOT NULL,
    quantity integer NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    expires_at date,
    location text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'depleted', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, material_id, batch_no)
);
CREATE INDEX IF NOT EXISTS material_batches_material_status_idx ON material_batches (material_id, status, expires_at);

CREATE TABLE IF NOT EXISTS material_units (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    material_id uuid NOT NULL REFERENCES materials(id),
    batch_id uuid REFERENCES material_batches(id),
    unit_code text NOT NULL,
    expires_at date,
    location text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'reserved', 'used', 'damaged', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, unit_code)
);
CREATE INDEX IF NOT EXISTS material_units_material_status_idx ON material_units (material_id, status, expires_at);
CREATE INDEX IF NOT EXISTS material_units_batch_status_idx ON material_units (batch_id, status);

ALTER TABLE material_requests ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
ALTER TABLE material_requests ADD COLUMN IF NOT EXISTS batch_id uuid REFERENCES material_batches(id);
ALTER TABLE material_requests ADD COLUMN IF NOT EXISTS unit_id uuid REFERENCES material_units(id);
UPDATE material_requests mr SET tenant_id = m.tenant_id FROM materials m WHERE mr.material_id = m.id AND mr.tenant_id IS NULL;
UPDATE material_requests SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE material_requests ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE material_requests ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
ALTER TABLE material_purchases ALTER COLUMN material_id DROP NOT NULL;
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchasable_material_id uuid REFERENCES purchasable_materials(id);
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_id_no text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_sequence_no text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_project_name text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_item_name text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_brand text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_spec text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_unit text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_remark text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_technical_requirement text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS purchase_min_spec text NOT NULL DEFAULT '';
UPDATE material_purchases mp SET tenant_id = m.tenant_id FROM materials m WHERE mp.material_id = m.id AND mp.tenant_id IS NULL;
UPDATE material_purchases SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE material_purchases ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE material_purchases ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS requester_phone text NOT NULL DEFAULT '';
ALTER TABLE material_purchases ADD COLUMN IF NOT EXISTS requester_email text NOT NULL DEFAULT '';
UPDATE material_purchases mp
SET requester_phone = COALESCE(NULLIF(mp.requester_phone, ''), u.phone),
    requester_email = COALESCE(NULLIF(mp.requester_email, ''), u.email)
FROM users u
WHERE mp.requester_id = u.id
  AND (mp.requester_phone = '' OR mp.requester_email = '');
UPDATE material_purchases mp
SET purchase_project_name = COALESCE(NULLIF(purchase_project_name, ''), m.name),
    purchase_item_name = COALESCE(NULLIF(purchase_item_name, ''), NULLIF(purchase_project_name, ''), m.name),
    purchase_brand = COALESCE(NULLIF(purchase_brand, ''), m.manufacturer, ''),
    purchase_spec = COALESCE(NULLIF(purchase_spec, ''), m.spec),
    purchase_unit = COALESCE(NULLIF(purchase_unit, ''), m.unit)
FROM materials m
WHERE mp.material_id = m.id
  AND (mp.purchase_project_name = '' OR mp.purchase_item_name = '' OR mp.purchase_spec = '' OR mp.purchase_unit = '');
UPDATE material_purchases mp
SET purchase_project_name = CASE
        WHEN pm.procurement_project <> '' AND (mp.purchase_project_name = '' OR mp.purchase_project_name = pm.project_name) THEN pm.procurement_project
        ELSE COALESCE(NULLIF(mp.purchase_project_name, ''), NULLIF(pm.procurement_project, ''), pm.project_name)
    END,
    purchase_item_name = COALESCE(NULLIF(mp.purchase_item_name, ''), pm.project_name)
FROM purchasable_materials pm
WHERE mp.purchasable_material_id = pm.id
  AND (mp.purchase_project_name = '' OR mp.purchase_item_name = '' OR mp.purchase_project_name = pm.project_name);

CREATE TABLE IF NOT EXISTS material_damage_reports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    material_id uuid NOT NULL REFERENCES materials(id),
    reporter_id uuid REFERENCES users(id),
    reporter text NOT NULL,
    group_name text NOT NULL DEFAULT '默认归属',
    quantity integer NOT NULL CHECK (quantity > 0),
    reason text NOT NULL,
    photo_url text NOT NULL DEFAULT '',
    attachment_url text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'processed', 'cancelled')),
    reviewer text NOT NULL DEFAULT '',
    review_comment text NOT NULL DEFAULT '',
    reviewed_at timestamptz,
    processed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS material_damage_reports_status_created_at_idx ON material_damage_reports (status, created_at DESC);
CREATE INDEX IF NOT EXISTS material_damage_reports_group_status_idx ON material_damage_reports (group_name, status);
ALTER TABLE material_damage_reports ADD COLUMN IF NOT EXISTS batch_id uuid REFERENCES material_batches(id);
ALTER TABLE material_damage_reports ADD COLUMN IF NOT EXISTS unit_id uuid REFERENCES material_units(id);
CREATE INDEX IF NOT EXISTS material_damage_reports_unit_idx ON material_damage_reports (unit_id);

ALTER TABLE inventory_ledger ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
ALTER TABLE inventory_ledger ADD COLUMN IF NOT EXISTS damage_id uuid REFERENCES material_damage_reports(id);
UPDATE inventory_ledger il SET tenant_id = m.tenant_id FROM materials m WHERE il.material_id = m.id AND il.tenant_id IS NULL;
UPDATE inventory_ledger SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE inventory_ledger ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE inventory_ledger ALTER COLUMN tenant_id SET NOT NULL;

INSERT INTO material_batches (tenant_id, material_id, batch_no, quantity, expires_at, location, status)
SELECT m.tenant_id,
       m.id,
       COALESCE(NULLIF(m.batch_no, ''), '默认批次'),
       m.stock,
       m.expires_at,
       concat_ws(' / ', NULLIF(m.storage_room, ''), NULLIF(m.storage_cabinet, ''), NULLIF(m.storage_layer, ''), NULLIF(m.storage_slot, '')),
       CASE WHEN m.stock > 0 THEN 'active' ELSE 'depleted' END
FROM materials m
WHERE m.product_type = 'standard'
  AND m.stock > 0
  AND NOT EXISTS (SELECT 1 FROM material_batches mb WHERE mb.material_id = m.id);

UPDATE materials m
SET stock = COALESCE(batch_stock.quantity, 0)
FROM (
    SELECT material_id, SUM(quantity)::int AS quantity
    FROM material_batches
    WHERE status <> 'disabled'
    GROUP BY material_id
) AS batch_stock
WHERE m.id = batch_stock.material_id
  AND m.product_type = 'standard';

UPDATE materials m
SET stock = unit_stock.quantity
FROM (
    SELECT material_id, count(*)::int AS quantity
    FROM material_units
    WHERE status IN ('available', 'reserved')
    GROUP BY material_id
) AS unit_stock
WHERE m.id = unit_stock.material_id;

ALTER TABLE material_approval_actions ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE material_approval_actions maa SET tenant_id = mr.tenant_id FROM material_requests mr WHERE maa.material_request_id = mr.id AND maa.tenant_id IS NULL;
UPDATE material_approval_actions SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE material_approval_actions ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE material_approval_actions ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE material_purchase_actions ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES tenants(id);
UPDATE material_purchase_actions mpa SET tenant_id = mp.tenant_id FROM material_purchases mp WHERE mpa.material_purchase_id = mp.id AND mpa.tenant_id IS NULL;
UPDATE material_purchase_actions SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE material_purchase_actions ALTER COLUMN tenant_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE material_purchase_actions ALTER COLUMN tenant_id SET NOT NULL;

CREATE TABLE IF NOT EXISTS email_verification_codes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    email text NOT NULL,
    code_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS email_verification_codes_lookup_idx ON email_verification_codes (tenant_id, lower(email), expires_at DESC);

CREATE OR REPLACE FUNCTION reject_ledger_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'ledger_entries are immutable';
END;
$$;

DROP TRIGGER IF EXISTS ledger_entries_immutable_update ON ledger_entries;
CREATE TRIGGER ledger_entries_immutable_update
BEFORE UPDATE OR DELETE ON ledger_entries
FOR EACH ROW EXECUTE FUNCTION reject_ledger_mutation();

CREATE OR REPLACE FUNCTION reject_approval_action_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'approval_actions are immutable';
END;
$$;

DROP TRIGGER IF EXISTS approval_actions_immutable_update ON approval_actions;
CREATE TRIGGER approval_actions_immutable_update
BEFORE UPDATE OR DELETE ON approval_actions
FOR EACH ROW EXECUTE FUNCTION reject_approval_action_mutation();

DROP TRIGGER IF EXISTS material_approval_actions_immutable_update ON material_approval_actions;
CREATE TRIGGER material_approval_actions_immutable_update
BEFORE UPDATE OR DELETE ON material_approval_actions
FOR EACH ROW EXECUTE FUNCTION reject_approval_action_mutation();

DROP TRIGGER IF EXISTS material_purchase_actions_immutable_update ON material_purchase_actions;
CREATE TRIGGER material_purchase_actions_immutable_update
BEFORE UPDATE OR DELETE ON material_purchase_actions
FOR EACH ROW EXECUTE FUNCTION reject_approval_action_mutation();
`

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, migrationSQL); err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if err := backfillMaterialUnits(ctx, tx); err != nil {
		return err
	}
	if err := normalizeMaterialUnitCodes(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	_, err = pool.Exec(ctx, extensionMigrationSQL)
	return err
}
