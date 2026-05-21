package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const extensionSeedSQL = `
INSERT INTO training_courses (tenant_id, title, category, instrument_id, instructor, delivery_mode, duration_hours, required_for_booking, status, description)
SELECT '00000000-0000-0000-0000-000000000001'::uuid, title, category,
       (SELECT id FROM instruments WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND name = instrument_name LIMIT 1),
       instructor, delivery_mode, duration_hours, required_for_booking, seed.status, seed.description
FROM (VALUES
    ('高分辨率质谱仪准入培训', '仪器培训', '高分辨率质谱仪', '平台工程师', 'blended', 2.0, true, 'active', '覆盖预约规则、样品前处理、上机安全和异常上报。'),
    ('实验室安全基础培训', '安全培训', '', '安全管理员', 'online', 1.5, true, 'active', '覆盖化学品、样本、门禁和应急处理基础要求。')
) AS seed(title, category, instrument_name, instructor, delivery_mode, duration_hours, required_for_booking, status, description)
WHERE NOT EXISTS (
    SELECT 1 FROM training_courses tc
    WHERE tc.tenant_id = '00000000-0000-0000-0000-000000000001' AND tc.title = seed.title
);

INSERT INTO training_authorizations (tenant_id, user_id, user_name, course_id, instrument_id, status, expires_at, notes)
SELECT '00000000-0000-0000-0000-000000000001'::uuid,
       u.id,
       u.name,
       tc.id,
       i.id,
       'active',
       now() + interval '180 days',
       '演示授权：培训通过后可预约对应仪器。'
FROM users u
JOIN training_courses tc ON tc.tenant_id = u.tenant_id AND tc.title = '高分辨率质谱仪准入培训'
JOIN instruments i ON i.tenant_id = u.tenant_id AND i.name = '高分辨率质谱仪'
WHERE lower(u.email) = 'wangmin@univ.edu.cn'
  AND NOT EXISTS (
      SELECT 1 FROM training_authorizations ta
      WHERE ta.tenant_id = u.tenant_id AND ta.user_id = u.id AND ta.course_id = tc.id
  );

INSERT INTO training_questions (tenant_id, title, question_type, options, correct_answer, explanation, status)
SELECT '00000000-0000-0000-0000-000000000001'::uuid, seed.title, seed.question_type, seed.options, seed.correct_answer, seed.explanation, seed.status
FROM (VALUES
    ('预约前是否需要确认仪器状态？', 'single', 'A. 需要\nB. 不需要', 'A', '预约前应确认仪器是否可用及是否需要准入。', 'active'),
    ('仪器维护中是否允许普通预约？', 'judge', 'A. 允许\nB. 不允许', 'B', '维护中的仪器应暂停预约，避免冲突。', 'active')
) AS seed(title, question_type, options, correct_answer, explanation, status)
WHERE NOT EXISTS (
    SELECT 1 FROM training_questions tq
    WHERE tq.tenant_id = '00000000-0000-0000-0000-000000000001' AND tq.title = seed.title
);

INSERT INTO training_exams (tenant_id, user_id, user_name, course_id, score, passed, answers, status, notes, exam_at)
SELECT u.tenant_id, u.id, u.name, tc.id, 92, true, 'A;B;实验步骤与数据留痕', 'graded', '演示在线考试记录。', now() - interval '1 day'
FROM users u
JOIN training_courses tc ON tc.tenant_id = u.tenant_id AND tc.title = '高分辨率质谱仪准入培训'
WHERE lower(u.email) = 'wangmin@univ.edu.cn'
  AND NOT EXISTS (
      SELECT 1 FROM training_exams te
      WHERE te.tenant_id = u.tenant_id AND te.user_id = u.id AND te.course_id = tc.id
  );

INSERT INTO training_practical_assessments (tenant_id, user_id, user_name, instrument_id, assessor, score, result, notes, assessment_at)
SELECT u.tenant_id, u.id, u.name, i.id, '平台工程师', 95, 'pass', '演示线下实操考核。', now() - interval '2 days'
FROM users u
JOIN instruments i ON i.tenant_id = u.tenant_id AND i.name = '高分辨率质谱仪'
WHERE lower(u.email) = 'wangmin@univ.edu.cn'
  AND NOT EXISTS (
      SELECT 1 FROM training_practical_assessments tp
      WHERE tp.tenant_id = u.tenant_id AND tp.user_id = u.id AND tp.instrument_id = i.id
  );

INSERT INTO training_rules (tenant_id, instrument_id, require_training, require_exam, require_approval, min_score, status, notes)
SELECT i.tenant_id, i.id, true, true, true, 80, 'active', '预约前需完成培训、考试和管理员审批。'
FROM instruments i
WHERE i.tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
  AND i.name IN ('高分辨率质谱仪', '流式细胞仪')
  AND NOT EXISTS (
      SELECT 1 FROM training_rules tr
      WHERE tr.tenant_id = i.tenant_id AND tr.instrument_id = i.id
  );

INSERT INTO spaces (tenant_id, name, kind, department, location, capacity, status, access_control_point, description)
SELECT '00000000-0000-0000-0000-000000000001'::uuid, seed.name, seed.kind, seed.department, seed.location, seed.capacity, seed.status, seed.access_control_point, seed.description
FROM (VALUES
    ('样品前处理室', 'lab', '化学与分子工程学院', 'A3-102', 6, 'available', 'A3-102-DOOR', '用于样品前处理、称量和预实验操作。'),
    ('公共会议室', 'meeting_room', '系统管理', 'A1-201', 18, 'available', 'A1-201-DOOR', '用于培训、评审和跨平台协调会议。'),
    ('低温样本库', 'storage', '生命科学学院', 'B1-010', 4, 'maintenance', 'B1-010-DOOR', '样本低温存储区域，当前进行温控巡检。')
) AS seed(name, kind, department, location, capacity, status, access_control_point, description)
WHERE NOT EXISTS (
    SELECT 1 FROM spaces sp
    WHERE sp.tenant_id = '00000000-0000-0000-0000-000000000001' AND sp.name = seed.name
);

INSERT INTO space_reservations (tenant_id, space_id, requester_id, requester, purpose, period, status)
SELECT sp.tenant_id, sp.id, u.id, u.name, '例会与样品排队协调',
       tstzrange(now() + interval '1 day', now() + interval '1 day 2 hours', '[)'),
       'approved'
FROM spaces sp
JOIN users u ON u.tenant_id = sp.tenant_id AND lower(u.email) = 'zhanghua@univ.edu.cn'
WHERE sp.name = '公共会议室'
  AND NOT EXISTS (
      SELECT 1 FROM space_reservations sr
      WHERE sr.tenant_id = sp.tenant_id AND sr.purpose = '例会与样品排队协调'
  );

INSERT INTO samples (tenant_id, code, name, owner_id, owner_name, department, group_name, location, status, hazard_level, storage_condition, description)
SELECT '00000000-0000-0000-0000-000000000001'::uuid,
       seed.code, seed.name,
       (SELECT id FROM users WHERE lower(email) = owner_email LIMIT 1),
       seed.owner_name, seed.department, seed.group_name, seed.location, seed.status, seed.hazard_level, seed.storage_condition, seed.description
FROM (VALUES
    ('SMP-2026-0001', '血清细胞因子面板', 'wangmin@univ.edu.cn', '王敏', '生命科学学院', '王敏团队', 'B1-010-02', 'stored', 'warning', '-80°C', '用于流式细胞仪和 ELISA 联合检测。'),
    ('SMP-2026-0002', '高分子薄膜样品', 'liming@univ.edu.cn', '李明', '物理学院', '先进材料平台', 'C2-210-05', 'testing', 'normal', '室温避光', '待进行 XRD 与表面形貌检测。')
) AS seed(code, name, owner_email, owner_name, department, group_name, location, status, hazard_level, storage_condition, description)
WHERE NOT EXISTS (
    SELECT 1 FROM samples s
    WHERE s.tenant_id = '00000000-0000-0000-0000-000000000001' AND s.code = seed.code
);

INSERT INTO sample_movements (tenant_id, sample_id, movement_type, from_location, to_location, reason)
SELECT s.tenant_id, s.id, 'in', '外部登记', s.location, '样本入库登记'
FROM samples s
WHERE s.code IN ('SMP-2026-0001', 'SMP-2026-0002')
  AND NOT EXISTS (
      SELECT 1 FROM sample_movements sm
      WHERE sm.sample_id = s.id AND sm.reason = '样本入库登记'
  );

INSERT INTO iot_devices (tenant_id, instrument_id, name, vendor, device_code, online, status, last_seen_at, telemetry, notes)
SELECT i.tenant_id, i.id, i.name || ' 采集终端', 'LIRS-IoT', 'IOT-' || right(replace(i.id::text, '-', ''), 8),
       i.status <> 'maintenance',
       CASE WHEN i.status = 'maintenance' THEN 'warning' ELSE 'online' END,
       now(),
       jsonb_build_object('temperature', '22.4°C', 'humidity', '46%', 'power', 'stable'),
       '预留 IoT 采集设备，可对接实际网关。'
FROM instruments i
WHERE i.name IN ('高分辨率质谱仪', '共聚焦显微镜')
  AND NOT EXISTS (
      SELECT 1 FROM iot_devices d
      WHERE d.tenant_id = i.tenant_id AND d.instrument_id = i.id
  );

INSERT INTO assistant_queries (tenant_id, requester, question, answer, context)
SELECT '00000000-0000-0000-0000-000000000001'::uuid,
       'system',
       '如何查看今日预约和待审批？',
       '可进入管理中心工作概览或预约记录查看今日预约、待审批、使用中和已履约状态。',
       'seed'
WHERE NOT EXISTS (
    SELECT 1 FROM assistant_queries
    WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND question = '如何查看今日预约和待审批？'
);
`

func seedExtensionData(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, extensionSeedSQL)
	return err
}
