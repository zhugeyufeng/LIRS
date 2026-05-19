package store

const extensionMigrationSQL = `
CREATE TABLE IF NOT EXISTS training_courses (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    title text NOT NULL,
    category text NOT NULL DEFAULT '仪器培训',
    instrument_id uuid REFERENCES instruments(id) ON DELETE SET NULL,
    instructor text NOT NULL DEFAULT '',
    delivery_mode text NOT NULL DEFAULT 'blended' CHECK (delivery_mode IN ('online', 'offline', 'blended')),
    duration_hours numeric(6,2) NOT NULL DEFAULT 0 CHECK (duration_hours >= 0),
    required_for_booking boolean NOT NULL DEFAULT false,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'archived')),
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS training_courses_tenant_status_idx ON training_courses (tenant_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS training_authorizations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    user_id uuid REFERENCES users(id),
    user_name text NOT NULL DEFAULT '',
    course_id uuid REFERENCES training_courses(id),
    instrument_id uuid REFERENCES instruments(id) ON DELETE SET NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'expired', 'revoked')),
    expires_at timestamptz NOT NULL DEFAULT (now() + interval '180 days'),
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS training_authorizations_tenant_status_idx ON training_authorizations (tenant_id, status, expires_at);

CREATE TABLE IF NOT EXISTS training_questions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    title text NOT NULL,
    question_type text NOT NULL DEFAULT 'single' CHECK (question_type IN ('single', 'multiple', 'judge', 'short')),
    options text NOT NULL DEFAULT '',
    correct_answer text NOT NULL DEFAULT '',
    explanation text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'draft', 'archived')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS training_questions_tenant_status_idx ON training_questions (tenant_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS training_exams (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    user_id uuid REFERENCES users(id),
    user_name text NOT NULL DEFAULT '',
    course_id uuid REFERENCES training_courses(id),
    score numeric(6,2) NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 100),
    passed boolean NOT NULL DEFAULT false,
    answers text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'graded', 'archived')),
    notes text NOT NULL DEFAULT '',
    exam_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS training_exams_tenant_status_idx ON training_exams (tenant_id, status, exam_at DESC);

CREATE TABLE IF NOT EXISTS training_practical_assessments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    user_id uuid REFERENCES users(id),
    user_name text NOT NULL DEFAULT '',
    instrument_id uuid REFERENCES instruments(id) ON DELETE SET NULL,
    assessor text NOT NULL DEFAULT '',
    score numeric(6,2) NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 100),
    result text NOT NULL DEFAULT 'pending' CHECK (result IN ('pending', 'pass', 'fail')),
    notes text NOT NULL DEFAULT '',
    assessment_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS training_practical_assessments_tenant_result_idx ON training_practical_assessments (tenant_id, result, assessment_at DESC);

CREATE TABLE IF NOT EXISTS training_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    instrument_id uuid NOT NULL REFERENCES instruments(id) ON DELETE CASCADE,
    require_training boolean NOT NULL DEFAULT true,
    require_exam boolean NOT NULL DEFAULT false,
    require_approval boolean NOT NULL DEFAULT true,
    min_score numeric(6,2) NOT NULL DEFAULT 80 CHECK (min_score >= 0 AND min_score <= 100),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS training_rules_tenant_instrument_idx ON training_rules (tenant_id, instrument_id);
CREATE INDEX IF NOT EXISTS training_rules_tenant_status_idx ON training_rules (tenant_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS spaces (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    name text NOT NULL,
    kind text NOT NULL DEFAULT 'lab' CHECK (kind IN ('lab', 'meeting_room', 'workspace', 'storage', 'other')),
    department text NOT NULL DEFAULT '',
    location text NOT NULL DEFAULT '',
    capacity integer NOT NULL DEFAULT 1 CHECK (capacity >= 0),
    status text NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'busy', 'maintenance', 'disabled')),
    access_control_point text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS spaces_tenant_kind_idx ON spaces (tenant_id, kind, status);

CREATE TABLE IF NOT EXISTS space_reservations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    space_id uuid NOT NULL REFERENCES spaces(id),
    requester_id uuid REFERENCES users(id),
    requester text NOT NULL,
    purpose text NOT NULL,
    period tstzrange NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'in_use', 'completed', 'cancelled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (lower(period) < upper(period))
);
CREATE INDEX IF NOT EXISTS space_reservations_tenant_status_idx ON space_reservations (tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS space_reservations_space_status_idx ON space_reservations (space_id, status);

CREATE TABLE IF NOT EXISTS samples (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    code text NOT NULL,
    name text NOT NULL,
    owner_id uuid REFERENCES users(id),
    owner_name text NOT NULL DEFAULT '',
    department text NOT NULL DEFAULT '',
    group_name text NOT NULL DEFAULT '',
    location text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'stored' CHECK (status IN ('stored', 'testing', 'checked_out', 'archived', 'disposed')),
    hazard_level text NOT NULL DEFAULT 'normal' CHECK (hazard_level IN ('normal', 'warning', 'danger')),
    storage_condition text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS samples_tenant_status_idx ON samples (tenant_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS sample_movements (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    sample_id uuid NOT NULL REFERENCES samples(id),
    movement_type text NOT NULL DEFAULT 'transfer' CHECK (movement_type IN ('in', 'out', 'transfer', 'test', 'dispose')),
    from_location text NOT NULL DEFAULT '',
    to_location text NOT NULL DEFAULT '',
    reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS sample_movements_tenant_created_idx ON sample_movements (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS lims_tasks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    sample_id uuid REFERENCES samples(id),
    instrument_id uuid REFERENCES instruments(id) ON DELETE SET NULL,
    title text NOT NULL,
    assay_type text NOT NULL DEFAULT '',
    priority text NOT NULL DEFAULT 'normal' CHECK (priority IN ('normal', 'high', 'urgent')),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'assigned', 'running', 'completed', 'cancelled')),
    requester_id uuid REFERENCES users(id),
    requester_name text NOT NULL DEFAULT '',
    due_at timestamptz NOT NULL DEFAULT (now() + interval '3 days'),
    result_summary text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS lims_tasks_tenant_status_idx ON lims_tasks (tenant_id, status, due_at);

CREATE TABLE IF NOT EXISTS eln_records (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    title text NOT NULL,
    author_id uuid REFERENCES users(id),
    author_name text NOT NULL DEFAULT '',
    project text NOT NULL DEFAULT '',
    linked_task_id uuid REFERENCES lims_tasks(id),
    content text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'signed', 'archived')),
    signed_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS eln_records_tenant_status_idx ON eln_records (tenant_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS iot_devices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    instrument_id uuid REFERENCES instruments(id) ON DELETE SET NULL,
    name text NOT NULL,
    vendor text NOT NULL DEFAULT '',
    device_code text NOT NULL DEFAULT '',
    online boolean NOT NULL DEFAULT false,
    status text NOT NULL DEFAULT 'offline' CHECK (status IN ('online', 'offline', 'warning', 'disabled')),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    telemetry jsonb NOT NULL DEFAULT '{}'::jsonb,
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS iot_devices_tenant_status_idx ON iot_devices (tenant_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS assistant_queries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    requester_id uuid REFERENCES users(id),
    requester text NOT NULL DEFAULT '',
    question text NOT NULL,
    answer text NOT NULL,
    context text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS assistant_queries_tenant_created_idx ON assistant_queries (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS business_configs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    domain text NOT NULL CHECK (domain IN ('workflow', 'billing')),
    kind text NOT NULL,
    title text NOT NULL,
    category text NOT NULL DEFAULT '',
    scope text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'disabled', 'archived')),
    description text NOT NULL DEFAULT '',
    config_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_by text NOT NULL DEFAULT 'system',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS business_configs_tenant_domain_kind_idx ON business_configs (tenant_id, domain, kind, status, updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS business_configs_tenant_domain_kind_title_idx ON business_configs (tenant_id, domain, kind, title);

ALTER TABLE training_courses DROP CONSTRAINT IF EXISTS training_courses_instrument_id_fkey;
ALTER TABLE training_courses ADD CONSTRAINT training_courses_instrument_id_fkey FOREIGN KEY (instrument_id) REFERENCES instruments(id) ON DELETE SET NULL;
ALTER TABLE training_authorizations DROP CONSTRAINT IF EXISTS training_authorizations_instrument_id_fkey;
ALTER TABLE training_authorizations ADD CONSTRAINT training_authorizations_instrument_id_fkey FOREIGN KEY (instrument_id) REFERENCES instruments(id) ON DELETE SET NULL;
ALTER TABLE training_practical_assessments DROP CONSTRAINT IF EXISTS training_practical_assessments_instrument_id_fkey;
ALTER TABLE training_practical_assessments ADD CONSTRAINT training_practical_assessments_instrument_id_fkey FOREIGN KEY (instrument_id) REFERENCES instruments(id) ON DELETE SET NULL;
ALTER TABLE training_rules DROP CONSTRAINT IF EXISTS training_rules_instrument_id_fkey;
ALTER TABLE training_rules ADD CONSTRAINT training_rules_instrument_id_fkey FOREIGN KEY (instrument_id) REFERENCES instruments(id) ON DELETE CASCADE;
ALTER TABLE lims_tasks DROP CONSTRAINT IF EXISTS lims_tasks_instrument_id_fkey;
ALTER TABLE lims_tasks ADD CONSTRAINT lims_tasks_instrument_id_fkey FOREIGN KEY (instrument_id) REFERENCES instruments(id) ON DELETE SET NULL;
ALTER TABLE iot_devices DROP CONSTRAINT IF EXISTS iot_devices_instrument_id_fkey;
ALTER TABLE iot_devices ADD CONSTRAINT iot_devices_instrument_id_fkey FOREIGN KEY (instrument_id) REFERENCES instruments(id) ON DELETE SET NULL;
`
