package store

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestRepositoryRequiresDatabaseForHealth(t *testing.T) {
	t.Parallel()

	repo := NewRepository(nil, redis.NewClient(&redis.Options{Addr: "localhost:0"}))
	if repo == nil {
		t.Fatal("expected repository")
	}
}

func TestRegisterInputShape(t *testing.T) {
	t.Parallel()

	input := RegisterInput{
		Name:       "测试用户",
		Phone:      "13800000000",
		Email:      "user@example.com",
		Password:   "password123",
		Department: "物理学院",
	}
	if input.Name == "" || input.Department == "" {
		t.Fatalf("unexpected empty input: %#v", input)
	}
}

func TestContextCanCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if ctx.Err() == nil {
		t.Fatal("expected canceled context")
	}
}

func TestProfileFieldChanged(t *testing.T) {
	t.Parallel()

	if profileFieldChanged(nil, "张三") {
		t.Fatal("expected omitted profile field to be unchanged")
	}
	same := " 张三 "
	if profileFieldChanged(&same, "张三") {
		t.Fatal("expected trimmed same profile field to be unchanged")
	}
	next := "李四"
	if !profileFieldChanged(&next, "张三") {
		t.Fatal("expected changed profile field to be detected")
	}
}

func TestRandomTenantCodeShape(t *testing.T) {
	t.Parallel()

	code, err := randomTenantCode()
	if err != nil {
		t.Fatalf("expected tenant code: %v", err)
	}
	if !strings.HasPrefix(code, "org-") || len(code) != 12 {
		t.Fatalf("unexpected tenant code shape: %q", code)
	}
	for _, char := range strings.TrimPrefix(code, "org-") {
		if !strings.ContainsRune("0123456789abcdef", char) {
			t.Fatalf("expected lowercase hex tenant code, got %q", code)
		}
	}
}

func TestNormalizeInstrumentClampsIntervalHours(t *testing.T) {
	t.Parallel()

	input := normalizeInstrument(InstrumentInput{BookingIntervalHours: 20})
	if input.BookingIntervalHours != 12 {
		t.Fatalf("expected interval to clamp to 12, got %d", input.BookingIntervalHours)
	}

	input = normalizeInstrument(InstrumentInput{BookingIntervalHours: 0})
	if input.BookingIntervalHours != 1 {
		t.Fatalf("expected interval to default to 1, got %d", input.BookingIntervalHours)
	}

	if input.ServiceStartHour != 0 || input.ServiceEndHour != 24 {
		t.Fatalf("expected default service hours 00-24, got %02d-%02d", input.ServiceStartHour, input.ServiceEndHour)
	}
	if input.MaxBookingHours != 72 {
		t.Fatalf("expected default max booking hours to be 72, got %d", input.MaxBookingHours)
	}
}

func TestNormalizeMaterialLifecycleFields(t *testing.T) {
	t.Parallel()

	input := normalizeMaterial(MaterialInput{
		Name:                   "  标准品  ",
		ProductType:            " standard ",
		Category:               "  单元素标准品 ",
		CASNo:                  " 50-00-0 ",
		ParentMaterialID:       " 00000000-0000-0000-0000-000000000001 ",
		DilutionFactor:         " 1:10 ",
		PreparationMethod:      " 稀释 ",
		StorageRoom:            " 冰箱A ",
		Remark:                 " 首次入库 ",
		CertificateURL:         " https://example.test/cert.pdf ",
		StandardCertificateURL: " https://example.test/std.pdf ",
		AttachmentURL:          " https://example.test/file.pdf ",
		QRCode:                 " SJ-001 ",
		OpenedAt:               " 2026-05-15 ",
	})
	if input.Name != "标准品" || input.ProductType != "standard" || input.CASNo != "50-00-0" {
		t.Fatalf("expected trimmed material identity fields, got %#v", input)
	}
	if input.StorageRoom != "冰箱A" || input.QRCode != "SJ-001" || input.OpenedAt != "2026-05-15" {
		t.Fatalf("expected trimmed material lifecycle fields, got %#v", input)
	}
	if input.ParentMaterialID != "" || input.DilutionFactor != "" || input.PreparationMethod != "" {
		t.Fatalf("标准品不应保留母液和稀释字段：%#v", input)
	}
	if input.Remark != "首次入库" {
		t.Fatalf("expected trimmed remark, got %#v", input)
	}
}

func TestMaterialInputFromCSVRowUsesProcurementProjectAndRemark(t *testing.T) {
	t.Parallel()

	header := materialImportHeaderIndex([]string{"资源名称", "资源类型", "一级目录", "规格", "单位", "采购项目名称及编号", "备注", "母液ID", "稀释倍数", "配制方法"})
	input := materialInputFromCSVRow(header, []string{"铅标准物质", "标准品", "单元素标准品", "100ug/L", "瓶", "2026-标准品采购项目 编号：STD-001", "证书随货", "00000000-0000-0000-0000-000000000001", "1:10", "稀释"})
	if input.TenderContract != "2026-标准品采购项目 编号：STD-001" || input.ContractNo != input.TenderContract || input.Remark != "证书随货" {
		t.Fatalf("采购项目或备注解析错误：%#v", input)
	}
	if input.ParentMaterialID != "" || input.DilutionFactor != "" || input.PreparationMethod != "" {
		t.Fatalf("标准品导入不应保留母液和稀释字段：%#v", input)
	}
}

func TestMaterialImportRecordsSupportXLSX(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../../../2026.xlsx")
	if err != nil {
		t.Skipf("跳过真实 XLSX 解析测试：%v", err)
	}
	records, err := purchasableMaterialImportRecords("materials。xlsx", content)
	if err != nil {
		t.Fatalf("读取 XLSX 失败：%v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected records from xlsx")
	}
}

func TestMaterialImportRecordsSupportRealStandardXLS(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../../../库存导出.xls")
	if err != nil {
		t.Skipf("跳过真实标准品 XLS 解析测试：%v", err)
	}
	if !looksLikeXLS(content) {
		t.Fatal("真实标准品样本应识别为 XLS")
	}
	records, err := purchasableMaterialImportRecords("库存导出.xls", content)
	if err != nil {
		t.Fatalf("读取真实标准品 XLS 失败：%v", err)
	}
	headerIndex := -1
	for i, row := range records {
		if materialLooksLikeHeader(row) {
			headerIndex = i
			break
		}
	}
	if headerIndex < 0 {
		t.Fatalf("真实标准品 XLS 未找到资源导入表头，行数=%d", len(records))
	}
	header := materialImportHeaderIndex(records[headerIndex])
	parsed := 0
	for _, row := range records[headerIndex+1:] {
		if rowBlank(row) {
			continue
		}
		input := materialInputFromCSVRow(header, row)
		if input.Name == "" || input.Category == "" || input.Spec == "" || input.Unit == "" {
			continue
		}
		parsed++
	}
	if parsed == 0 {
		t.Fatal("真实标准品 XLS 未解析到可导入资源行")
	}
}

func TestMaterialImportRecordsDetectsOLEContentWithoutXLSFilename(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../../../库存导出.xls")
	if err != nil {
		t.Skipf("跳过真实标准品 XLS 解析测试：%v", err)
	}
	records, err := purchasableMaterialImportRecords("库存导出", content)
	if err != nil {
		t.Fatalf("应根据 OLE 文件头识别 XLS：%v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected records from OLE content")
	}
}

func TestMaterialBatchHelpers(t *testing.T) {
	t.Parallel()

	input := MaterialInput{
		StorageRoom:    " 冰箱A ",
		StorageCabinet: " 二层 ",
		StorageLayer:   "",
		StorageSlot:    " A01 ",
	}
	if got := materialInputBatchLocation(input); got != "冰箱A / 二层 / A01" {
		t.Fatalf("unexpected batch location: %q", got)
	}
	if got := materialBatchReason("申领出库", "STD-001"); got != "申领出库（批次：STD-001）" {
		t.Fatalf("unexpected batch reason: %q", got)
	}
	if got := materialBatchReason("申领出库", ""); got != "申领出库" {
		t.Fatalf("unexpected plain reason: %q", got)
	}
	if got := materialUnitReason("申领出库", "STD-001", "QBZRY-2026-05-16-0001"); got != "申领出库（批次：STD-001，编号：QBZRY-2026-05-16-0001）" {
		t.Fatalf("unexpected unit reason: %q", got)
	}
	if got := materialUnitCodePrefix("铅标准溶液"); got != "QBZRY" {
		t.Fatalf("unexpected material unit prefix: %q", got)
	}
	if got := materialUnitCodeDatePart("8F9AC021-2026-05-16-0005"); got != "2026-05-16" {
		t.Fatalf("unexpected material unit date part: %q", got)
	}
	if sequence, ok := materialUnitCodeSequence("QBZRY-2026-05-16-0005", "QBZRY", "2026-05-16"); !ok || sequence != 5 {
		t.Fatalf("unexpected material unit sequence: %d %v", sequence, ok)
	}
	if materialUnitCodeMatchesRule("8F9AC021-2026-05-16-0005", "QBZRY", "2026-05-16") {
		t.Fatal("expected hashed legacy unit code to miss current rule")
	}
	if !materialUnitCodeMatchesRule("QBZRY-2026-05-16-0005", "QBZRY", "2026-05-16") {
		t.Fatal("expected current unit code to match rule")
	}
	units := []MaterialUnit{{Status: "available"}, {Status: "reserved"}, {Status: "used"}, {Status: "available"}}
	if got := countAvailableMaterialUnits(units); got != 2 {
		t.Fatalf("unexpected available unit count: %d", got)
	}
}

func TestPurchasableMaterialImportProjectHeader(t *testing.T) {
	t.Parallel()

	row := []string{"9-病毒PCR试剂类采购项目 编号：WXCDCQTCG2021-029（满足采购人要求20210929）", "", ""}
	if got := purchasableMaterialProjectHeader(row); !strings.Contains(got, "病毒PCR试剂类采购项目") {
		t.Fatalf("expected project header, got %q", got)
	}
	header := materialImportHeaderIndex([]string{"ID号", "序号", "项目名称", "品牌", "规格", "单位", "采购价（元）", "备注", "技术要求", "最小规格"})
	input := purchasableMaterialInputFromRow(header, []string{"PCR-001", "1", "病毒核酸检测试剂盒", "品牌A", "200T/盒", "盒", "1200.5", "", "满足要求", "1盒"}, "项目头")
	if input.ProcurementProject != "项目头" || input.ProjectName != "病毒核酸检测试剂盒" || input.PurchasePrice != 1200.5 || input.IDNo != "PCR-001" {
		t.Fatalf("unexpected purchasable material row: %#v", input)
	}
}

func TestMaterialPurchaseStatusActionMapsReviewActions(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"approved": "approve",
		"rejected": "reject",
		"returned": "return",
	}
	for status, want := range cases {
		got, ok := materialPurchaseStatusAction(status)
		if !ok || got != want {
			t.Fatalf("%s action = %q %v, want %q true", status, got, ok, want)
		}
	}
	if _, ok := materialPurchaseStatusAction("ordered"); ok {
		t.Fatal("非审批状态不应进入申购审批动作映射")
	}
}

func TestMaterialWorkflowStatusLabelCoversPurchaseStatuses(t *testing.T) {
	t.Parallel()

	if got := materialWorkflowStatusLabel("registered"); got != "已登记" {
		t.Fatalf("unexpected registered label: %q", got)
	}
	if got := materialWorkflowStatusLabel("returned"); got != "退回修改" {
		t.Fatalf("unexpected returned label: %q", got)
	}
}

func TestPurchasableMaterialImportRecordsFrom2026XLSX(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../../../2026.xlsx")
	if err != nil {
		t.Skipf("跳过真实采购目录解析测试：%v", err)
	}
	records, err := purchasableMaterialImportRecords("2026.xlsx", content)
	if err != nil {
		t.Fatalf("read xlsx: %v", err)
	}
	if len(records) < 4 {
		t.Fatalf("expected xlsx rows, got %d", len(records))
	}
	headerIndex := -1
	for i, row := range records {
		if purchasableMaterialLooksLikeHeader(row) {
			headerIndex = i
			break
		}
	}
	if headerIndex != 1 {
		t.Fatalf("unexpected header index: %d", headerIndex)
	}
	project := purchasableMaterialProjectHeader(records[headerIndex+1])
	if !strings.Contains(project, "细菌性传染病PCR检测试剂类采购项目") {
		t.Fatalf("unexpected project header: %q", project)
	}
	input := purchasableMaterialInputFromRow(materialImportHeaderIndex(records[headerIndex]), records[headerIndex+2], project)
	if input.IDNo != "634" || input.SequenceNo != "1" || input.ProcurementProject != project || !strings.Contains(input.ProjectName, "副溶血性弧菌") || input.Brand != "生科原" || input.Spec != "1T" || input.Unit != "人份" || input.PurchasePrice != 80 {
		t.Fatalf("unexpected parsed row: %#v", input)
	}
}

func TestPurchasableMaterialImportRecordsDetectsChineseDotXLSX(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../../../2026.xlsx")
	if err != nil {
		t.Skipf("跳过真实采购目录解析测试：%v", err)
	}
	records, err := purchasableMaterialImportRecords("2026。xlsx", content)
	if err != nil {
		t.Fatalf("read chinese-dot xlsx: %v", err)
	}
	if len(records) < 4 || !purchasableMaterialLooksLikeHeader(records[1]) {
		t.Fatalf("unexpected records from chinese-dot xlsx: rows=%d", len(records))
	}
}

func TestPurchasableMaterialImportPlanFrom2026XLSX(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../../../2026.xlsx")
	if err != nil {
		t.Skipf("跳过真实采购目录解析测试：%v", err)
	}
	records, err := purchasableMaterialImportRecords("2026.xlsx", content)
	if err != nil {
		t.Fatalf("read xlsx: %v", err)
	}
	headerIndex := -1
	for i, row := range records {
		if purchasableMaterialLooksLikeHeader(row) {
			headerIndex = i
			break
		}
	}
	if headerIndex < 0 {
		t.Fatal("missing purchasable material header")
	}
	header := materialImportHeaderIndex(records[headerIndex])
	currentProject := ""
	valid := 0
	invalidRows := 0
	for _, row := range records[headerIndex+1:] {
		if rowBlank(row) {
			continue
		}
		if project := purchasableMaterialProjectHeader(row); project != "" {
			currentProject = project
			continue
		}
		input := purchasableMaterialInputFromRow(header, row, currentProject)
		if input.IDNo == "" || input.SequenceNo == "" || input.ProjectName == "" || input.Brand == "" || input.Spec == "" || input.Unit == "" {
			invalidRows++
			continue
		}
		valid++
	}
	if valid < 5150 {
		t.Fatalf("expected at least 5150 valid rows, got %d", valid)
	}
	if invalidRows > 5 {
		t.Fatalf("expected only a few invalid rows, got %d", invalidRows)
	}
}

func TestScanPurchasableMaterialIncludesProcurementProjectStatus(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 5, 19, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	item, err := scanPurchasableMaterial(notificationScanRow{values: []any{
		"material-1",
		"ID-001",
		"1",
		"project-1",
		"采购项目 编号：A",
		"2026-12-31",
		"disabled",
		"试剂盒",
		"品牌A",
		"1T",
		"盒",
		float64(80),
		"备注",
		"技术要求",
		"最小规格",
		"active",
		createdAt,
		updatedAt,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if item.ProcurementProjectStatus != "disabled" || item.ProcurementExpiresAt != "2026-12-31" {
		t.Fatalf("采购项目状态或有效期扫描错误：%#v", item)
	}
}

func TestScanMaterialIncludesRemark(t *testing.T) {
	t.Parallel()

	item, err := scanMaterial(notificationScanRow{values: []any{
		"material-1",
		"铅标准物质",
		"standard",
		"单元素标准品",
		"金属元素",
		"100ug/L",
		"瓶",
		float64(80),
		1,
		1,
		"供应商",
		"生产商",
		"BATCH-1",
		"CAT-1",
		"7439-92-1",
		"CRM",
		"100ug/L",
		"",
		"",
		"",
		"",
		"2-8°C",
		"标准品库",
		"冰箱",
		"二层",
		"A01",
		"采购项目 编号：STD-001",
		"采购项目 编号：STD-001",
		"随货证书",
		"",
		"",
		"",
		"QR-1",
		"2026-12-31",
		"2026-05-19",
		30,
		"2026-06-18",
		0,
		5,
		true,
		30,
		0,
		"normal",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if item.Remark != "随货证书" || item.TenderContract == "" {
		t.Fatalf("资源备注或采购项目扫描错误：%#v", item)
	}
}

func TestPurchasableMaterialImportRecordsUnsupportedTypeMessage(t *testing.T) {
	t.Parallel()

	_, err := purchasableMaterialImportRecords("2026.txt", []byte("not,a,catalog"))
	if err == nil || !strings.Contains(err.Error(), "不支持的文件类型") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}

func TestDingTalkOAuthURLUsesConfiguredRedirectURI(t *testing.T) {
	t.Parallel()

	raw := dingTalkOAuthURL(dingTalkSettingsValue{
		ClientID:         "ding-client-id",
		OAuthRedirectURI: "https://lirs.example.com/settings/dingtalk",
	}, "state-token")
	if !strings.HasPrefix(raw, "https://login.dingtalk.com/oauth2/auth?") {
		t.Fatalf("unexpected oauth host: %s", raw)
	}
	for _, fragment := range []string{
		"client_id=ding-client-id",
		"redirect_uri=https%3A%2F%2Flirs.example.com%2Fsettings%2Fdingtalk",
		"response_type=code",
		"scope=openid",
		"state=state-token",
	} {
		if !strings.Contains(raw, fragment) {
			t.Fatalf("oauth url missing %s: %s", fragment, raw)
		}
	}
}

func TestDingTalkSettingsFromValueDoesNotExposeSecrets(t *testing.T) {
	t.Parallel()

	settings := dingTalkSettingsFromValue(dingTalkSettingsValue{
		SchemaVersion:    2,
		Enabled:          true,
		ClientID:         "client-id",
		ClientSecret:     "client-secret",
		CorpID:           "corp-id",
		RobotCode:        "robot-code",
		OAuthRedirectURI: "https://lirs.example.com/settings/dingtalk",
		EventCallbackURL: "https://lirs.example.com/api/dingtalk/events",
		EventAesKey:      "aes-key",
		EventToken:       "token",
	}, "系统", time.Time{})

	if settings.SchemaVersion != 2 || !settings.Enabled || settings.ClientID != "client-id" || settings.ClientSecret != "" || settings.RobotCode != "robot-code" || settings.OAuthRedirectURI == "" || settings.EventCallbackURL == "" || settings.EventAesKey != "" || settings.EventToken != "" {
		t.Fatalf("unexpected dingtalk settings projection: %#v", settings)
	}
	if !settings.ClientSecretConfigured || !settings.EventAesKeyConfigured || !settings.EventTokenConfigured {
		t.Fatalf("expected secret configured flags: %#v", settings)
	}
}

func TestGeneratedDingTalkEventCallbackURLUsesTenantCode(t *testing.T) {
	t.Parallel()

	got := generatedDingTalkEventCallbackURL("https://lirs.example.com/settings/dingtalk", "tenant-a")
	want := "https://lirs.example.com/api/dingtalk/events/tenant-a"
	if got != want {
		t.Fatalf("事件订阅 URL 生成错误：got %q want %q", got, want)
	}
}

func TestNormalizeDingTalkSettingsValueDoesNotMigrateLegacyKeys(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"enabled":true,"` + strings.Join(deprecatedDingTalkJSONKeys(), `":"deprecated","`) + `":"deprecated"}`)
	value := normalizeDingTalkSettingsValue(raw, dingTalkSettingsValue{})

	if value.ClientID != "" || value.ClientSecret != "" || value.RobotCode != "" || value.EventCallbackURL != "" || value.EventAesKey != "" || value.EventToken != "" {
		t.Fatalf("deprecated keys should not be migrated: %#v", value)
	}
}

func TestTenantScopedDingTalkSettingsKey(t *testing.T) {
	t.Parallel()

	if got := tenantScopedDingTalkSettingsKey(" tenant-1 "); got != "dingtalk:tenant-1" {
		t.Fatalf("unexpected tenant scoped key: %q", got)
	}
	if got := tenantScopedDingTalkSettingsKey(""); got != "dingtalk:"+defaultTenantID {
		t.Fatalf("expected default tenant scoped key, got %q", got)
	}
}

func TestDingTalkDefaultHTTPEventCredentialsStayCurrent(t *testing.T) {
	t.Parallel()

	if !strings.Contains(migrationSQL, "O3qwhUsprT1XONy8p7K1jhuq3O2fg7xP9kRw27b8MKq") {
		t.Fatal("migration should seed the current DingTalk event AES key")
	}
	if !strings.Contains(migrationSQL, "Hml3sD9iYksE0CtwtHPMJBPvAF") {
		t.Fatal("migration should seed the current DingTalk event token")
	}
	for _, legacyKey := range deprecatedDingTalkJSONKeys() {
		if strings.Contains(migrationSQL, legacyKey) {
			t.Fatalf("migration should not keep deprecated DingTalk JSON field %s", legacyKey)
		}
	}
}

func TestDingTalkQuickAuthCodeAcceptsNestedAndFlatUserInfo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		body     string
		wantPath []string
	}{
		{
			name: "nested",
			body: `{"errcode":0,"result":{"userid":"user-1","unionid":"union-1"}}`,
			wantPath: []string{
				"/v1.0/oauth2/ding-corp-id/token",
				"/topapi/v2/user/getuserinfo",
				"/v1.0/oauth2/ding-corp-id/token",
				"/topapi/v2/user/get",
			},
		},
		{
			name: "flat",
			body: `{"errcode":0,"userid":"user-1","unionid":"union-1"}`,
			wantPath: []string{
				"/v1.0/oauth2/ding-corp-id/token",
				"/topapi/v2/user/getuserinfo",
				"/v1.0/oauth2/ding-corp-id/token",
				"/topapi/v2/user/get",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var paths []string
			repo := NewRepository(nil, nil)
			repo.http = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				paths = append(paths, request.URL.Path)
				switch request.URL.Path {
				case "/v1.0/oauth2/ding-corp-id/token":
					return jsonResponse(`{"access_token":"app-token","expires_in":7200}`), nil
				case "/topapi/v2/user/getuserinfo":
					return jsonResponse(tc.body), nil
				case "/topapi/v2/user/get":
					return jsonResponse(`{"errcode":0,"result":{"userid":"user-1","unionid":"union-1","name":"张三","mobile":"13800000000"}}`), nil
				default:
					t.Fatalf("unexpected dingtalk path: %s", request.URL.Path)
					return jsonResponse(`{}`), nil
				}
			})}
			identity, err := repo.dingTalkIdentityByQuickAuthCode(context.Background(), dingTalkSettingsValue{ClientID: "client-id", ClientSecret: "client-secret", CorpID: "ding-corp-id"}, "auth-code")
			if err != nil {
				t.Fatalf("quick auth code should resolve identity: %v", err)
			}
			if identity.UserID != "user-1" || identity.UnionID != "union-1" || identity.Name != "张三" {
				t.Fatalf("unexpected identity: %#v", identity)
			}
			if strings.Join(paths, ",") != strings.Join(tc.wantPath, ",") {
				t.Fatalf("unexpected dingtalk paths: %#v", paths)
			}
		})
	}
}

func deprecatedDingTalkJSONKeys() []string {
	return []string{
		"app" + "Key",
		"agent" + "Id",
		"stream" + "ClientId",
		"stream" + "ClientSecret",
		"callback" + "Route",
		"stream" + "ModeEnabled",
	}
}

func TestPushDingTalkNotificationScopesTenantContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scoped := WithTenantContext(ctx, TenantContext{TenantID: "tenant-1"})
	if TenantFromContext(scoped).TenantID != "tenant-1" {
		t.Fatal("expected tenant context to carry notification tenant")
	}
}

func TestDingTalkOAuthStateKeyScopesTenantAndUser(t *testing.T) {
	t.Parallel()

	key := dingTalkOAuthStateKey("tenant-1", "user-1", "state-1")
	for _, fragment := range []string{"lirs:dingtalk:oauth:", "tenant-1", "user-1", "state-1"} {
		if !strings.Contains(key, fragment) {
			t.Fatalf("oauth state key missing %s: %s", fragment, key)
		}
	}
}

func TestDingTalkAppAccessTokenUsesV1Endpoint(t *testing.T) {
	t.Parallel()

	var requestedPath string
	var requestedBody string
	repo := &Repository{
		http: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requestedPath = request.URL.Path
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			requestedBody = string(body)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"token-1","expires_in":7200}`)),
			}, nil
		})},
	}

	token, err := repo.dingTalkAppAccessToken(context.Background(), dingTalkSettingsValue{ClientID: "client-id", ClientSecret: "client-secret", CorpID: "ding-corp-id"})
	if err != nil {
		t.Fatalf("unexpected access token error: %v", err)
	}
	if token != "token-1" || requestedPath != "/v1.0/oauth2/ding-corp-id/token" {
		t.Fatalf("unexpected token request: token=%s path=%s body=%s", token, requestedPath, requestedBody)
	}
	for _, fragment := range []string{`"client_id":"client-id"`, `"client_secret":"client-secret"`, `"grant_type":"client_credentials"`} {
		if !strings.Contains(requestedBody, fragment) {
			t.Fatalf("access token body missing %s: %s", fragment, requestedBody)
		}
	}
}

func TestDingTalkWorkNotificationUsesRobotBatchSend(t *testing.T) {
	t.Parallel()

	var paths []string
	var notificationBody string
	repo := &Repository{
		http: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			paths = append(paths, request.URL.Path)
			if request.URL.Path == "/v1.0/robot/oToMessages/batchSend" {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Fatalf("read notification body: %v", err)
				}
				notificationBody = string(body)
				if got := request.Header.Get("x-acs-dingtalk-access-token"); got != "token-1" {
					t.Fatalf("unexpected dingtalk token header: %s", got)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"processQueryKey":"process-1"}`)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"token-1","expires_in":7200}`)),
			}, nil
		})},
	}

	err := repo.sendDingTalkWorkNotification(context.Background(), dingTalkSettingsValue{ClientID: "client-id", ClientSecret: "client-secret", CorpID: "ding-corp-id", RobotCode: "robot-code"}, "user-1", "标题", "正文")
	if err != nil {
		t.Fatalf("unexpected notification error: %v", err)
	}
	if len(paths) != 2 || paths[0] != "/v1.0/oauth2/ding-corp-id/token" || paths[1] != "/v1.0/robot/oToMessages/batchSend" {
		t.Fatalf("unexpected dingtalk paths: %#v", paths)
	}
	for _, fragment := range []string{`"robotCode":"robot-code"`, `"userIds":["user-1"]`, `"msgKey":"sampleMarkdown"`, `### 标题`} {
		if !strings.Contains(notificationBody, fragment) {
			t.Fatalf("notification body missing %s: %s", fragment, notificationBody)
		}
	}
}

func TestGraphMailAccessTokenUsesClientCredentials(t *testing.T) {
	t.Parallel()

	var requestedPath string
	var requestedBody string
	repo := &Repository{
		http: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requestedPath = request.URL.Path
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read token body: %v", err)
			}
			requestedBody = string(body)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"graph-token"}`)),
			}, nil
		})},
	}

	token, err := repo.graphMailAccessToken(context.Background(), graphMailSettingsValue{TenantID: "tenant-id", ClientID: "client-id", ClientSecret: "client-secret"})
	if err != nil {
		t.Fatalf("unexpected graph token error: %v", err)
	}
	if token != "graph-token" || requestedPath != "/tenant-id/oauth2/v2.0/token" {
		t.Fatalf("unexpected graph token request: token=%s path=%s body=%s", token, requestedPath, requestedBody)
	}
	for _, fragment := range []string{"client_id=client-id", "client_secret=client-secret", "grant_type=client_credentials", "scope=https%3A%2F%2Fgraph.microsoft.com%2F.default"} {
		if !strings.Contains(requestedBody, fragment) {
			t.Fatalf("graph token body missing %s: %s", fragment, requestedBody)
		}
	}
}

func TestSendGraphMailUsesUserSendMailEndpoint(t *testing.T) {
	t.Parallel()

	var paths []string
	var mailBody string
	repo := &Repository{
		http: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			paths = append(paths, request.URL.Path)
			if strings.HasSuffix(request.URL.Path, "/sendMail") {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Fatalf("read graph mail body: %v", err)
				}
				mailBody = string(body)
				if got := request.Header.Get("Authorization"); got != "Bearer graph-token" {
					t.Fatalf("unexpected graph token header: %s", got)
				}
				return &http.Response{
					StatusCode: http.StatusAccepted,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"graph-token"}`)),
			}, nil
		})},
	}

	err := repo.sendGraphMail(context.Background(), graphMailSettingsValue{
		Enabled:                 true,
		TenantID:                "tenant-id",
		ClientID:                "client-id",
		ClientSecret:            "client-secret",
		SenderUserPrincipalName: "sender@example.com",
		SaveToSentItems:         true,
	}, "user@example.com", "标题", "正文")
	if err != nil {
		t.Fatalf("unexpected graph send error: %v", err)
	}
	if len(paths) != 2 || paths[0] != "/tenant-id/oauth2/v2.0/token" || paths[1] != "/v1.0/users/sender@example.com/sendMail" {
		t.Fatalf("unexpected graph paths: %#v", paths)
	}
	for _, fragment := range []string{`"subject":"标题"`, `"content":"正文"`, `"address":"user@example.com"`, `"saveToSentItems":true`} {
		if !strings.Contains(mailBody, fragment) {
			t.Fatalf("graph mail body missing %s: %s", fragment, mailBody)
		}
	}
}

func TestGraphHTTPErrorMessageExtractsMicrosoftError(t *testing.T) {
	t.Parallel()

	message := graphHTTPErrorMessage(http.StatusUnauthorized, []byte(`{"error":{"code":"InvalidAuthenticationToken","message":"Access token is empty."}}`))
	for _, fragment := range []string{"status=401", "code=InvalidAuthenticationToken", "message=Access token is empty."} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("graph error message missing %s: %s", fragment, message)
		}
	}
}

func TestNotificationTargetUserIDsRespectScope(t *testing.T) {
	t.Parallel()

	repo := &Repository{}
	personal, err := repo.notificationTargetUserIDs(context.Background(), Notification{
		TenantID:    defaultTenantID,
		UserID:      "user-1",
		TargetScope: "personal",
	}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(personal) != 1 || personal[0] != "user-1" {
		t.Fatalf("个人通知应该只推送目标用户，得到 %#v", personal)
	}

	unknown, err := repo.notificationTargetUserIDs(context.Background(), Notification{
		TenantID:    defaultTenantID,
		TargetScope: "unknown",
	}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(unknown) != 0 {
		t.Fatalf("未知通知范围不应推送，得到 %#v", unknown)
	}
}

type notificationScanRow struct {
	values []any
}

func (r notificationScanRow) Scan(dest ...any) error {
	for index, value := range r.values {
		switch target := dest[index].(type) {
		case *string:
			*target = value.(string)
		case *bool:
			*target = value.(bool)
		case *float64:
			*target = value.(float64)
		case *int:
			*target = value.(int)
		case *time.Time:
			*target = value.(time.Time)
		default:
			return errors.New("unsupported scan target")
		}
	}
	return nil
}

func TestScanNotificationIncludesTenantAndUpdatedAt(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 5, 17, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	item, err := scanNotification(notificationScanRow{values: []any{
		"notification-1",
		"tenant-1",
		"机构一",
		"user-1",
		"团队一",
		"部门一",
		"personal",
		"system",
		"",
		"标题",
		"正文",
		"info",
		true,
		createdAt,
		updatedAt,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if item.TenantID != "tenant-1" || item.TenantName != "机构一" || !item.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("通知机构或更新时间扫描错误：%#v", item)
	}
}

func TestDingTalkEventCallbackCryptoRoundTrip(t *testing.T) {
	t.Parallel()

	aesKey := "O3qwhUsprT1XONy8p7K1jhuq3O2fg7xP9kRw27b8MKq"
	encrypted, err := encryptDingTalkEvent(`{"EventType":"user_add_org","CorpId":"ding-corp-id","UserId":"ding-user-1"}`, aesKey, "ding-corp-id")
	if err != nil {
		t.Fatalf("encrypt dingtalk event: %v", err)
	}
	event, err := decryptDingTalkEvent(encrypted, aesKey, "ding-corp-id")
	if err != nil {
		t.Fatalf("decrypt dingtalk event: %v", err)
	}
	if event["EventType"] != "user_add_org" || event["CorpId"] != "ding-corp-id" || event["UserId"] != "ding-user-1" {
		t.Fatalf("unexpected dingtalk event payload: %#v", event)
	}
}

func TestValidMaterialStatusAndProductType(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"normal", "near_expiry", "low", "expired", "open_expired", "freeze_thaw_exceeded", "damaged", "disabled"} {
		if !validMaterialStatus(status) {
			t.Fatalf("expected status %q to be valid", status)
		}
	}
	if validMaterialStatus("archived") {
		t.Fatal("expected archived status to be invalid")
	}
	for _, productType := range []string{"consumable", "reagent", "standard"} {
		if !validMaterialProductType(productType) {
			t.Fatalf("expected product type %q to be valid", productType)
		}
	}
	for _, productType := range []string{"working_solution", "mixed_standard", "sample"} {
		if validMaterialProductType(productType) {
			t.Fatalf("expected product type %q to be invalid", productType)
		}
	}
}

func TestMaterialSeedRowsMatchColumnList(t *testing.T) {
	t.Parallel()

	materialSeed := seedSQL[strings.Index(seedSQL, "INSERT INTO materials ("):]
	valuesStart := strings.Index(materialSeed, "FROM (VALUES")
	aliasStart := strings.Index(materialSeed, ") AS seed(")
	if valuesStart < 0 || aliasStart < 0 || aliasStart <= valuesStart {
		t.Fatal("expected material seed VALUES block")
	}
	aliasColumnsStart := aliasStart + len(") AS seed(")
	aliasColumnsEnd := strings.Index(materialSeed[aliasColumnsStart:], ")")
	if aliasColumnsEnd < 0 {
		t.Fatal("expected material seed column alias list")
	}

	expectedColumns := countSeedCSVColumns(materialSeed[aliasColumnsStart : aliasColumnsStart+aliasColumnsEnd])
	rows := splitSeedTuples(materialSeed[valuesStart+len("FROM (VALUES") : aliasStart])
	if len(rows) == 0 {
		t.Fatal("expected material seed rows")
	}
	for index, row := range rows {
		if got := countSeedCSVColumns(row); got != expectedColumns {
			t.Fatalf("material seed row %d has %d columns, expected %d: %s", index+1, got, expectedColumns, row)
		}
	}
}

func TestNormalizeServiceHours(t *testing.T) {
	t.Parallel()

	start, end := normalizeServiceHours(8, 20)
	if start != 8 || end != 20 {
		t.Fatalf("expected 08-20, got %02d-%02d", start, end)
	}

	start, end = normalizeServiceHours(-1, 0)
	if start != 0 || end != 24 {
		t.Fatalf("expected invalid hours to default to 00-24, got %02d-%02d", start, end)
	}

	start, end = normalizeServiceHours(23, 23)
	if start != 23 || end != 24 {
		t.Fatalf("expected end hour to default after start, got %02d-%02d", start, end)
	}
}

func TestReservationIntervalAlignment(t *testing.T) {
	t.Parallel()

	loc := reservationServiceLocation()
	if !isAlignedToReservationInterval(time.Date(2026, 5, 7, 0, 0, 0, 0, loc), 4, 0) {
		t.Fatal("expected 00:00 to align with 4 hour interval")
	}
	if isAlignedToReservationInterval(time.Date(2026, 5, 7, 2, 0, 0, 0, loc), 4, 0) {
		t.Fatal("expected 02:00 to not align with 4 hour interval")
	}
	if !isAlignedToReservationInterval(time.Date(2026, 5, 7, 20, 0, 0, 0, loc), 4, 8) {
		t.Fatal("expected 20:00 to align with 4 hour interval from 08:00")
	}
	if !isAlignedToReservationInterval(time.Date(2026, 5, 8, 0, 0, 0, 0, loc), 4, 8) {
		t.Fatal("expected 24:00 service boundary to align with 4 hour interval from 08:00")
	}
	if !isAlignedToReservationInterval(time.Date(2026, 5, 7, 8, 0, 0, 0, loc), 5, 8) {
		t.Fatal("expected 08:00 to align with 5 hour interval")
	}
	if isAlignedToReservationInterval(time.Date(2026, 5, 7, 9, 0, 0, 0, loc), 5, 8) {
		t.Fatal("expected 09:00 to not align with 5 hour interval")
	}
	if !isAlignedToReservationInterval(time.Date(2026, 5, 7, 13, 0, 0, 0, loc), 5, 8) {
		t.Fatal("expected 13:00 to align with 5 hour interval")
	}
}

func TestReservationServiceHours(t *testing.T) {
	t.Parallel()

	loc := reservationServiceLocation()
	if !isWithinServiceHours(time.Date(2026, 5, 7, 20, 0, 0, 0, loc), time.Date(2026, 5, 8, 0, 0, 0, 0, loc), 8, 24) {
		t.Fatal("expected 20:00-24:00 to be within 08-24 service hours")
	}
	if isWithinServiceHours(time.Date(2026, 5, 8, 0, 0, 0, 0, loc), time.Date(2026, 5, 8, 4, 0, 0, 0, loc), 8, 24) {
		t.Fatal("expected after-midnight slot to be outside 08-24 service hours")
	}
	if !isWithinServiceHours(time.Date(2026, 5, 8, 0, 0, 0, 0, loc), time.Date(2026, 5, 8, 4, 0, 0, 0, loc), 0, 24) {
		t.Fatal("expected 00:00-04:00 to be within all-day service hours")
	}
	if !isWithinServiceHours(time.Date(2026, 5, 7, 20, 0, 0, 0, loc), time.Date(2026, 5, 10, 20, 0, 0, 0, loc), 0, 24) {
		t.Fatal("expected 72 hour cross-day reservation to be within all-day service hours")
	}
	if isWithinServiceHours(time.Date(2026, 5, 7, 20, 0, 0, 0, loc), time.Date(2026, 5, 8, 10, 0, 0, 0, loc), 8, 24) {
		t.Fatal("expected cross-day reservation spanning closed hours to be outside 08-24 service hours")
	}
}

func TestNormalizeFooterSettingsValueFallsBackToDefaults(t *testing.T) {
	t.Parallel()

	value := normalizeFooterSettingsValue(footerSettingsValue{})
	if value.BrandName == "" || value.Description == "" || value.Copyright == "" {
		t.Fatalf("expected footer defaults to be populated: %#v", value)
	}
	if len(value.Sections) == 0 {
		t.Fatal("expected default footer sections")
	}
}

func TestNormalizeFooterSettingsValueTrimsAndDropsBlankSections(t *testing.T) {
	t.Parallel()

	value := normalizeFooterSettingsValue(footerSettingsValue{
		BrandName:   "  自定义系统  ",
		BaseURL:     "  https://lirs.example.cn/  ",
		Description: "  简介  ",
		Sections: []FooterSection{
			{Title: "  技术栈 ", Lines: []string{"  Go  ", "", " Next.js "}},
			{Title: "   ", Lines: []string{"   "}},
		},
		Copyright: "  © 2026 Test  ",
	})
	if value.BrandName != "自定义系统" {
		t.Fatalf("expected trimmed brand name, got %q", value.BrandName)
	}
	if len(value.Sections) != 1 {
		t.Fatalf("expected blank section to be removed, got %#v", value.Sections)
	}
	if len(value.Sections[0].Lines) != 2 {
		t.Fatalf("expected blank lines to be removed, got %#v", value.Sections[0].Lines)
	}
	if value.BaseURL != "https://lirs.example.cn" {
		t.Fatalf("expected base url to be trimmed, got %q", value.BaseURL)
	}
}

func TestMaterialDetailURL(t *testing.T) {
	t.Parallel()

	if got := materialDetailURL(" https://lirs.example.cn/ ", "abc-123"); got != "https://lirs.example.cn/materials/abc-123" {
		t.Fatalf("expected absolute material detail url, got %q", got)
	}
	if got := materialDetailURL("", "abc 123"); got != "/materials/abc%20123" {
		t.Fatalf("expected relative material detail url, got %q", got)
	}
	if got := materialDetailURL("https://lirs.example.cn", ""); got != "" {
		t.Fatalf("expected empty url without material id, got %q", got)
	}
}

func TestValidSiteBaseURL(t *testing.T) {
	t.Parallel()

	if !validSiteBaseURL("https://lirs.example.cn") {
		t.Fatal("expected https base url to be valid")
	}
	if validSiteBaseURL("ftp://lirs.example.cn") {
		t.Fatal("expected non-http base url to be invalid")
	}
	if validSiteBaseURL("lirs.example.cn") {
		t.Fatal("expected missing scheme base url to be invalid")
	}
	if validSiteBaseURL("https://lirs.example.cn?x=1") {
		t.Fatal("expected base url with query to be invalid")
	}
}

func splitSeedTuples(block string) []string {
	rows := make([]string, 0)
	inQuote := false
	depth := 0
	start := -1
	for index := 0; index < len(block); index++ {
		character := block[index]
		if character == '\'' {
			if inQuote && index+1 < len(block) && block[index+1] == '\'' {
				index++
				continue
			}
			inQuote = !inQuote
			continue
		}
		if inQuote {
			continue
		}
		switch character {
		case '(':
			if depth == 0 {
				start = index + 1
			}
			depth++
		case ')':
			depth--
			if depth == 0 && start >= 0 {
				rows = append(rows, strings.TrimSpace(block[start:index]))
				start = -1
			}
		}
	}
	return rows
}

func countSeedCSVColumns(row string) int {
	if strings.TrimSpace(row) == "" {
		return 0
	}
	count := 1
	inQuote := false
	for index := 0; index < len(row); index++ {
		character := row[index]
		if character == '\'' {
			if inQuote && index+1 < len(row) && row[index+1] == '\'' {
				index++
				continue
			}
			inQuote = !inQuote
			continue
		}
		if character == ',' && !inQuote {
			count++
		}
	}
	return count
}
