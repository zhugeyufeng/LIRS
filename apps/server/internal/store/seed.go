package store

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const seedSQL = `
INSERT INTO instruments (name, category, department, group_name, status, location, hourly_rate, brand, model, asset_code, description, technical_specs, booking_rule, maintenance_summary, booking_window_days, booking_interval_hours)
SELECT name, category, department, group_name, status, location, hourly_rate, brand, model, asset_code, description, technical_specs, booking_rule, maintenance_summary, booking_window_days, booking_interval_hours
FROM (VALUES
    ('高分辨率质谱仪', '分析测试', '化学与分子工程学院', '李明团队', 'available', 'A3-204', 380.00, 'Thermo Fisher', 'Orbitrap Exploris 480', 'LAB-2026-0001', '适用于复杂样品的高灵敏度定性与定量分析。', '分辨率 480k；ESI/APCI 离子源；支持 LC-MS/MS。', '最小预约 1 小时，最长 72 小时；需提前 2 小时提交；审批中时段锁定。', '2026-04-18 完成真空泵保养，运行稳定。', 14, 1),
    ('共聚焦显微镜', '成像平台', '生命科学学院', '王敏团队', 'busy', 'B1-118', 260.00, 'Zeiss', 'LSM 980 Airyscan 2', 'LAB-2026-0082', '支持多通道荧光成像和三维重建。', '405/488/561/640nm 激光器；63x 油镜；Airyscan 超分辨模块。', '最小预约 1 小时；活细胞实验需备注样本类型；审批通过后自动生成门禁授权。', '2026-05-02 更换载物台校准件。', 7, 1),
    ('X 射线衍射仪', '材料表征', '物理学院', '先进材料平台', 'maintenance', 'C2-310', 420.00, 'Bruker', 'D8 Advance', 'LAB-2026-0035', '用于晶体结构和薄膜取向分析。', 'Cu 靶；二维探测器；薄膜附件。', '维护期间不可预约；恢复后最小预约 2 小时。', '探测器年度校准中，预计 2026-05-08 恢复。', 21, 2),
    ('流式细胞仪', '细胞分析', '医学院', '免疫工程平台', 'available', 'M1-070', 180.00, 'BD', 'FACSymphony A5', 'LAB-2026-0044', '支持细胞分选、免疫表型和凋亡检测。', '5 激光 18 色；96 孔板进样；自动补偿。', '最小预约 1 小时；分选实验需管理员现场确认。', '2026-04-29 完成喷嘴清洁。', 10, 1)
) AS seed(name, category, department, group_name, status, location, hourly_rate, brand, model, asset_code, description, technical_specs, booking_rule, maintenance_summary, booking_window_days, booking_interval_hours)
WHERE NOT EXISTS (SELECT 1 FROM instruments WHERE instruments.name = seed.name);

INSERT INTO users (name, email, phone, department, group_name, password_hash, role, status, email_verified)
VALUES
('张华', 'zhanghua@univ.edu.cn', '13800000001', '物理学院', '先进材料平台', 'demo', 'lab_admin', 'active', true),
('李明', 'liming@univ.edu.cn', '13800000002', '生命科学学院', '王敏团队', 'demo', 'student', 'active', true),
('王敏', 'wangmin@univ.edu.cn', '13800000003', '化学与分子工程学院', '李明团队', 'demo', 'group_leader', 'active', true),
('测试普通用户', 'testuser@lirs.local', '13900000000', '生命科学学院', '王敏团队', 'demo', 'student', 'active', true)
ON CONFLICT (tenant_id, lower(email)) DO NOTHING;

INSERT INTO site_settings (setting_key, value, updated_by)
VALUES (
    'footer',
    jsonb_build_object(
        'brandName', 'LIRS 2026 实验室运营系统',
        'brandTagline', '仪器预约、审批、使用、耗材、财务与审计闭环平台',
        'baseUrl', '',
        'description', '系统数据统一写入 PostgreSQL，登录会话、审批、库存、财务流水和审计记录均从数据库读取；Redis 用于缓存与事件队列。',
        'sections', jsonb_build_array(
            jsonb_build_object(
                'title', '技术栈',
                'lines', jsonb_build_array(
                    'TypeScript / Next.js / React / Tailwind CSS / shadcn/ui / Lucide Icons',
                    'Go / Gin / Hono / Zod / Drizzle ORM / PostgreSQL 15+ / Redis 7+'
                )
            ),
            jsonb_build_object(
                'title', '运行信息',
                'lines', jsonb_build_array(
                    'Hono API Gateway: 8090',
                    'Go Core API: 8081'
                )
            )
        ),
        'copyright', '© 2026 LIRS. All rights reserved.'
    ),
    'system'
)
ON CONFLICT (setting_key) DO NOTHING;

INSERT INTO organization_units (kind, name, parent_name)
VALUES
('department', '化学与分子工程学院', ''),
('department', '生命科学学院', ''),
('department', '物理学院', ''),
('department', '工程学院', ''),
('department', '医学院', ''),
('department', '系统管理', ''),
('group', '李明团队', '化学与分子工程学院'),
('group', '王敏团队', '生命科学学院'),
('group', '先进材料平台', '物理学院'),
('group', '免疫工程平台', '医学院'),
('group', '系统管理组', '系统管理'),
('group', '默认归属', '系统管理'),
('group', '未分配归属', '系统管理')
ON CONFLICT (tenant_id, kind, name) DO NOTHING;

INSERT INTO financial_accounts (tenant_id, user_id, user_name, group_name, balance, credit_limit)
SELECT u.tenant_id, u.id, u.name, u.group_name, seed.balance, seed.credit_limit
FROM (VALUES
    ('zhanghua@univ.edu.cn', 64200.00, 120000.00),
    ('liming@univ.edu.cn', 128000.00, 200000.00),
    ('wangmin@univ.edu.cn', 86500.00, 150000.00),
    ('testuser@lirs.local', 10000.00, 30000.00)
) AS seed(email, balance, credit_limit)
JOIN users u ON lower(u.email) = seed.email
ON CONFLICT (tenant_id, user_id) WHERE user_id IS NOT NULL DO NOTHING;

INSERT INTO materials (
    name, product_type, category, subcategory, spec, unit, unit_price, stock, warning_line,
    supplier, manufacturer, batch_no, catalog_no, cas_no, grade, concentration,
    parent_material_id, dilution_factor, preparation_method,
    storage_condition, storage_room, storage_cabinet, storage_layer, storage_slot,
    tender_contract, contract_no, certificate_url, standard_certificate_url, attachment_url,
    qr_code, expires_at, opened_at, open_expire_days, freeze_thaw_count, freeze_thaw_limit,
    approval_required, near_expiry_days, status
)
SELECT name, product_type, category, subcategory, spec, unit, unit_price, stock, warning_line,
       supplier, manufacturer, batch_no, catalog_no, cas_no, grade, concentration,
       NULL, dilution_factor, preparation_method,
       storage_condition, storage_room, storage_cabinet, storage_layer, storage_slot,
       tender_contract, contract_no, certificate_url, standard_certificate_url, attachment_url,
       qr_code, expires_at::date, NULLIF(opened_at, '')::date, open_expire_days, freeze_thaw_count, freeze_thaw_limit,
       approval_required, near_expiry_days,
       CASE WHEN stock <= warning_line THEN 'low' ELSE 'normal' END
FROM (VALUES
    ('无菌移液吸头', 'consumable', '塑料耗材', '移液耗材', '10uL 盒装灭菌', '盒', 38.00, 8, 12, '赛默飞', 'Thermo Fisher', 'TIP-2604-A', '94052320', '', '灭菌级', '', '', '', '室温干燥', 'A库', '耗材架1', '二层', 'B03', '2026-通用耗材框架合同', 'TC-2026-001', '', '', '', 'MAT-TIP-2604-A', '2027-04-30', '', 0, 0, 0, false, 30),
    ('琼脂糖 Low EEO', 'reagent', '分子生物学试剂', '核酸电泳', '100g/瓶', '瓶', 560.00, 26, 6, 'BioWest', 'BioWest', 'AGR-2603-B', '111860', '9012-36-6', '分子生物学级', '', '', '', '室温干燥', '试剂库', '常温柜2', '三层', 'C12', '2026-分子试剂招标合同', 'TC-2026-018', '/files/certs/agarose.pdf', '', '', 'REA-AGR-2603-B', '2028-03-31', '', 0, 0, 0, false, 60),
    ('PBS 缓冲液', 'standard', '工作液', '缓冲液', '500mL/瓶', '瓶', 45.00, 52, 15, '索莱宝', 'Solarbio', 'PBS-2605-C', 'P1020', '', '分析纯', '1x', '1:10', '10x 母液稀释后过滤除菌', '2-8°C', '冷藏库', '冰箱1', '上层', 'D05', '2026-基础试剂招标合同', 'TC-2026-022', '', '', '', 'REA-PBS-2605-C', '2027-05-31', '2026-05-01', 90, 0, 0, false, 30),
    ('铅标准溶液', 'standard', '单元素标准品', '金属元素', '1000mg/L 50mL', '瓶', 860.00, 5, 2, '国家标准物质中心', 'NIM', 'STD-2604-PB', 'GBW(E)080129', '7439-92-1', 'CRM', '1000mg/L', '', '按证书要求保存标准品原液', '2-8°C 避光', '标准品库', '防爆冰箱', '二层', 'A08', '2026-标准品采购合同', 'TC-2026-066', '', '/files/certs/pb-standard.pdf', '', 'STD-PB-2604', '2027-02-28', '2026-04-20', 60, 1, 5, true, 45),
    ('离心管', 'consumable', '塑料耗材', '离心耗材', '1.5mL 500支/包', '包', 65.00, 18, 10, 'Axygen', 'Axygen', 'TUBE-2604-D', 'MCT-150-C', '', '无酶级', '', '', '', '室温干燥', 'A库', '耗材架2', '一层', 'A11', '2026-塑料耗材框架合同', 'TC-2026-031', '', '', '', 'MAT-TUBE-2604-D', '2029-01-31', '', 0, 0, 0, false, 30)
) AS seed(name, product_type, category, subcategory, spec, unit, unit_price, stock, warning_line, supplier, manufacturer, batch_no, catalog_no, cas_no, grade, concentration, dilution_factor, preparation_method, storage_condition, storage_room, storage_cabinet, storage_layer, storage_slot, tender_contract, contract_no, certificate_url, standard_certificate_url, attachment_url, qr_code, expires_at, opened_at, open_expire_days, freeze_thaw_count, freeze_thaw_limit, approval_required, near_expiry_days)
WHERE NOT EXISTS (SELECT 1 FROM materials WHERE materials.name = seed.name AND materials.batch_no = seed.batch_no);

INSERT INTO material_categories (name, parent_name, display_order, status)
SELECT name, parent_name, display_order, 'active'
FROM (VALUES
    ('塑料耗材', '', 10),
    ('移液耗材', '塑料耗材', 11),
    ('离心耗材', '塑料耗材', 12),
    ('分子生物学试剂', '', 20),
    ('核酸电泳', '分子生物学试剂', 21),
    ('单元素标准品', '', 30),
    ('金属元素', '单元素标准品', 31),
    ('工作液', '', 40),
    ('缓冲液', '工作液', 41),
    ('混标', '', 50)
) AS seed(name, parent_name, display_order)
ON CONFLICT (tenant_id, name) DO UPDATE
SET parent_name = EXCLUDED.parent_name,
    display_order = EXCLUDED.display_order,
    status = 'active',
    updated_at = now();
`

func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, seedSQL); err != nil {
		return err
	}
	if err := seedExtensionData(ctx, pool); err != nil {
		return err
	}
	if err := ensureInitialAdmin(ctx, pool); err != nil {
		return err
	}
	if err := ensureInitialNotification(ctx, pool); err != nil {
		return err
	}
	return upgradeDemoUserPasswords(ctx, pool)
}

func ensureInitialNotification(ctx context.Context, pool *pgxpool.Pool) error {
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notifications WHERE title = '系统已初始化')`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	repo := NewRepository(pool, nil)
	_, err := repo.createNotification(ctx, defaultTenantID, "", "", "", "global", "系统已初始化", "PostgreSQL schema 和演示数据已准备完成。", "success")
	return err
}

func ensureInitialAdmin(ctx context.Context, pool *pgxpool.Pool) error {
	email := strings.TrimSpace(strings.ToLower(os.Getenv("INITIAL_ADMIN_EMAIL")))
	password := os.Getenv("INITIAL_ADMIN_PASSWORD")
	name := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_NAME"))
	if email == "" {
		email = "admin@lirs.local"
	}
	if password == "" {
		return errors.New("INITIAL_ADMIN_PASSWORD is required")
	}
	if name == "" {
		name = "系统初始管理员"
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO users (name, email, phone, department, group_name, password_hash, role, status, email_verified)
VALUES ($1, $2, '00000000000', '系统管理', '系统管理组', $3, 'super_admin', 'active', true)
ON CONFLICT (tenant_id, lower(email)) DO UPDATE
SET role = 'super_admin',
    status = 'active',
    email_verified = true,
    group_name = '系统管理组',
    password_hash = EXCLUDED.password_hash,
    auth_epoch = users.auth_epoch + 1,
    updated_at = now()
`, name, email, passwordHash)
	return err
}

func upgradeDemoUserPasswords(ctx context.Context, pool *pgxpool.Pool) error {
	password := os.Getenv("INITIAL_DEMO_USER_PASSWORD")
	if password == "" {
		return errors.New("INITIAL_DEMO_USER_PASSWORD is required")
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
UPDATE users
SET password_hash = $1,
    auth_epoch = auth_epoch + 1,
    updated_at = now()
WHERE password_hash = 'demo' OR password_hash LIKE 'demo:%'
`, passwordHash)
	return err
}
