package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/xuri/excelize/v2"

	"lirs/apps/server/internal/store"
)

type fakeAuthRepo struct {
	user store.User
	err  error
}

func (f fakeAuthRepo) CurrentUser(context.Context, string) (store.User, error) {
	return f.user, f.err
}

func TestRequireAnyRoleRejectsMissingToken(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPatch, "/api/users/id/review", nil)

	if _, ok := requireAnyRole(context, fakeAuthRepo{}, "lab_admin"); ok {
		t.Fatal("expected missing role to be rejected")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestRequireAnyRoleAllowsMatchingRole(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPatch, "/api/users/id/review", nil)
	context.Request.Header.Set("Authorization", "Bearer token")
	repo := fakeAuthRepo{user: store.User{
		ID:            "user-1",
		Name:          "管理员",
		Role:          "lab_admin",
		Status:        "active",
		EmailVerified: true,
	}}

	if _, ok := requireAnyRole(context, repo, "lab_admin", "super_admin"); !ok {
		t.Fatal("expected matching role to be allowed")
	}
}

func TestRequireAnyRoleRejectsInvalidSession(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPatch, "/api/users/id/review", nil)
	context.Request.Header.Set("Authorization", "Bearer expired")

	if _, ok := requireAnyRole(context, fakeAuthRepo{err: errors.New("expired")}, "lab_admin"); ok {
		t.Fatal("expected invalid session to be rejected")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestFilterReservationsForActorScopesRows(t *testing.T) {
	t.Parallel()

	items := []store.Reservation{
		{ID: "own", UserID: "u1", GroupName: "A"},
		{ID: "group", UserID: "u2", GroupName: "A"},
		{ID: "other", UserID: "u3", GroupName: "B"},
	}

	student := filterReservationsForActor(store.Actor{UserID: "u1", Role: "student", GroupName: "A"}, items)
	if len(student) != 1 || student[0].ID != "own" {
		t.Fatalf("student should only see own reservation: %#v", student)
	}

	leader := filterReservationsForActor(store.Actor{UserID: "leader", Role: "group_leader", GroupName: "A"}, items)
	if len(leader) != 2 {
		t.Fatalf("group leader should see group reservations, got %#v", leader)
	}

	admin := filterReservationsForActor(store.Actor{UserID: "admin", Role: "lab_admin"}, items)
	if len(admin) != 3 {
		t.Fatalf("admin should see all reservations, got %#v", admin)
	}
}

func TestFilterLedgerForActorScopesRows(t *testing.T) {
	t.Parallel()

	items := []store.LedgerEntry{
		{ID: "own", UserID: "u1", GroupName: "A"},
		{ID: "group", UserID: "u2", GroupName: "A"},
		{ID: "other", UserID: "u3", GroupName: "B"},
	}

	student := filterLedgerForActor(store.Actor{UserID: "u1", Role: "student", GroupName: "A"}, items)
	if len(student) != 1 || student[0].ID != "own" {
		t.Fatalf("student should only see own ledger rows: %#v", student)
	}

	leader := filterLedgerForActor(store.Actor{UserID: "leader", Role: "group_leader", GroupName: "A"}, items)
	if len(leader) != 0 {
		t.Fatalf("finance is personal scoped; group leader should not see group rows, got %#v", leader)
	}
}

func TestFilterMaterialDamagesForActorScopesRows(t *testing.T) {
	t.Parallel()

	items := []store.MaterialDamage{
		{ID: "own", ReporterID: "u1", GroupName: "A"},
		{ID: "group", ReporterID: "u2", GroupName: "A"},
		{ID: "other", ReporterID: "u3", GroupName: "B"},
	}

	student := filterMaterialDamagesForActor(store.Actor{UserID: "u1", Role: "student", GroupName: "A"}, items)
	if len(student) != 1 || student[0].ID != "own" {
		t.Fatalf("student should only see own damage reports: %#v", student)
	}

	leader := filterMaterialDamagesForActor(store.Actor{UserID: "leader", Role: "group_leader", GroupName: "A"}, items)
	if len(leader) != 2 {
		t.Fatalf("group leader should see group damage reports, got %#v", leader)
	}

	admin := filterMaterialDamagesForActor(store.Actor{UserID: "admin", Role: "material_admin"}, items)
	if len(admin) != 3 {
		t.Fatalf("material admin should see all damage reports, got %#v", admin)
	}
}

func TestMaterialStatusAndLocationLabels(t *testing.T) {
	t.Parallel()

	item := store.Material{StorageRoom: "冰箱A", StorageCabinet: "二层", StorageLayer: "盒1", StorageSlot: "A01"}
	if got := materialLocation(item); got != "冰箱A / 二层 / 盒1 / A01" {
		t.Fatalf("unexpected material location: %q", got)
	}
	if got := materialStatusLabel("open_expired"); got != "开封超期" {
		t.Fatalf("unexpected material status label: %q", got)
	}
	if got := materialProductTypeLabel("standard"); got != "标准品/标准物质" {
		t.Fatalf("unexpected product type label: %q", got)
	}
}

func TestSaveMaterialCertificateUploadAcceptsPdf(t *testing.T) {
	uploadRoot := t.TempDir()
	context := materialCertificateUploadContext(t, "cert.pdf", []byte("%PDF-1.4\n证书内容"))

	url, err := saveMaterialCertificateUpload(context, uploadRoot)
	if err != nil {
		t.Fatalf("expected pdf upload to succeed: %v", err)
	}
	if !strings.HasPrefix(url, "/files/material-certificates/") || !strings.HasSuffix(url, ".pdf") {
		t.Fatalf("unexpected certificate url: %q", url)
	}
	content, err := os.ReadFile(filepath.Join(uploadRoot, strings.TrimPrefix(url, "/files/")))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Fatalf("saved certificate should keep pdf signature, got %q", string(content))
	}
}

func TestSaveMaterialCertificateUploadRejectsNonPdfExtension(t *testing.T) {
	context := materialCertificateUploadContext(t, "cert.png", []byte("%PDF-1.4\n证书内容"))

	if _, err := saveMaterialCertificateUpload(context, t.TempDir()); err == nil || !strings.Contains(err.Error(), "仅支持 PDF 文件") {
		t.Fatalf("expected pdf extension error, got %v", err)
	}
}

func TestSaveMaterialCertificateUploadRejectsInvalidPdfContent(t *testing.T) {
	context := materialCertificateUploadContext(t, "cert.pdf", []byte("不是 PDF"))

	if _, err := saveMaterialCertificateUpload(context, t.TempDir()); err == nil || !strings.Contains(err.Error(), "文件内容不是有效 PDF") {
		t.Fatalf("expected pdf content error, got %v", err)
	}
}

func materialCertificateUploadContext(t *testing.T, filename string, content []byte) *gin.Context {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/uploads/material-certificates", &body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return context
}

func TestMaterialBatchLabelsRemainChinese(t *testing.T) {
	t.Parallel()

	if got := materialRequestStatusLabel("outbound"); got != "已出库" {
		t.Fatalf("unexpected request status label: %q", got)
	}
}

func TestBindOptionalJSONRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPatch, "/api/reservations/id/approve", strings.NewReader("{"))
	context.Request.Header.Set("Content-Type", "application/json")

	var input struct {
		Comment string `json:"comment"`
	}
	if bindOptionalJSON(context, &input) {
		t.Fatal("非法 JSON 不应继续执行业务")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSaveMaterialCertificateUploadRejectsOversizedPdf(t *testing.T) {
	content := append([]byte("%PDF-"), bytes.Repeat([]byte("A"), 8<<20)...)
	context := materialCertificateUploadContext(t, "cert.pdf", content)

	if _, err := saveMaterialCertificateUpload(context, t.TempDir()); err == nil || !strings.Contains(err.Error(), "超过 8MB") {
		t.Fatalf("expected oversized pdf error, got %v", err)
	}
}

func TestBindOptionalJSONAllowsEmptyPayload(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPatch, "/api/reservations/id/approve", nil)

	var input struct {
		Comment string `json:"comment"`
	}
	if !bindOptionalJSON(context, &input) {
		t.Fatal("空请求体应被视为可选 JSON")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected response code: %d", recorder.Code)
	}
}

func TestMaterialRequestExportWorkbookMatchesStandardUsageTemplate(t *testing.T) {
	t.Parallel()

	file, err := materialRequestExportWorkbook("2026-01", []store.MaterialRequestExportRow{{
		MaterialRequest: store.MaterialRequest{
			MaterialName: "河豚毒素标准物质",
			Requester:    "张三",
			BatchNo:      "B-001",
			UnitCode:     "U-001",
			Location:     "4℃冰柜",
			Quantity:     1,
			CreatedAt:    time.Date(2026, 1, 5, 9, 30, 0, 0, time.UTC),
			Status:       "outbound",
		},
		StandardNo:   "GBW-001",
		Brand:        "国家标准物质中心",
		Spec:         "1mg/mL",
		Unit:         "支",
		ExpiresAt:    "2027.01.05",
		ApprovalInfo: "管理员",
	}})
	if err != nil {
		t.Fatal(err)
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	workbook, err := excelize.OpenReader(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer workbook.Close()
	if got := workbook.GetSheetName(0); got != "领用历史记录" {
		t.Fatalf("unexpected sheet name: %q", got)
	}
	if got, _ := workbook.GetCellValue("领用历史记录", "A1"); !strings.Contains(got, "标准物质领用记录表") {
		t.Fatalf("unexpected title: %q", got)
	}
	if got, _ := workbook.GetCellValue("领用历史记录", "A3"); got != "品名" {
		t.Fatalf("unexpected first header: %q", got)
	}
	if got, _ := workbook.GetCellValue("领用历史记录", "L3"); got != "审批信息" {
		t.Fatalf("unexpected last header: %q", got)
	}
	if got, _ := workbook.GetCellValue("领用历史记录", "A4"); got != "河豚毒素标准物质" {
		t.Fatalf("unexpected first material name: %q", got)
	}
}

func TestCanAccessNotificationScopes(t *testing.T) {
	t.Parallel()

	actor := store.Actor{UserID: "u1", Role: "student", GroupName: "A", Department: "Chem"}
	cases := []struct {
		name string
		item store.Notification
		want bool
	}{
		{name: "global", item: store.Notification{TargetScope: "global"}, want: true},
		{name: "personal match", item: store.Notification{TargetScope: "personal", UserID: "u1"}, want: true},
		{name: "personal other", item: store.Notification{TargetScope: "personal", UserID: "u2"}, want: false},
		{name: "group match", item: store.Notification{TargetScope: "group", GroupName: "A"}, want: true},
		{name: "group other", item: store.Notification{TargetScope: "group", GroupName: "B"}, want: false},
		{name: "department match", item: store.Notification{TargetScope: "department", Department: "Chem"}, want: true},
	}
	for _, tc := range cases {
		if got := canAccessNotification(actor, tc.item); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestRespondMapsKnownDatabaseErrors(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/register", nil)

	respond(context, nil, &pgconn.PgError{Message: `duplicate key value violates unique constraint "users_email_key"`, Code: "23505"})

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "resource already exists" {
		t.Fatalf("expected conflict error, got %q", payload["error"])
	}
}

func TestRespondReturnsUnknownDatabaseErrors(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/register", nil)

	respond(context, nil, &pgconn.PgError{Message: "database failure", Code: "XX000"})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "database failure (SQLSTATE XX000)" {
		t.Fatalf("expected real database error, got %q", payload["error"])
	}
}

func TestRespondKeepsClientSafeErrors(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/register", nil)

	respond(context, nil, newClientMessageError("invalid registration input"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "invalid registration input" {
		t.Fatalf("expected validation error, got %q", payload["error"])
	}
}

func TestRespondKeepsGraphMailErrors(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/notification-channel-settings/graph-mail/test", nil)

	respond(context, nil, newClientMessageError("graph mail test send failed: graph mail token failed: status=401 code=invalid_client message=凭证无效"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "graph mail test send failed: graph mail token failed: status=401 code=invalid_client message=凭证无效" {
		t.Fatalf("expected graph mail error, got %q", payload["error"])
	}
}
