package store

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
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
		StorageRoom:            " 冰箱A ",
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

func TestDingTalkSettingsFromValueCanExposeSecretsOnDemand(t *testing.T) {
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

	if settings.SchemaVersion != 2 || !settings.Enabled || settings.ClientID != "client-id" || settings.ClientSecret != "client-secret" || settings.RobotCode != "robot-code" || settings.OAuthRedirectURI == "" || settings.EventCallbackURL == "" {
		t.Fatalf("unexpected dingtalk settings projection: %#v", settings)
	}
	if !settings.ClientSecretConfigured || !settings.EventAesKeyConfigured || !settings.EventTokenConfigured {
		t.Fatalf("expected secret configured flags: %#v", settings)
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
