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

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

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

func TestRespondMasksDatabaseErrors(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/register", nil)

	respond(context, nil, &pgconn.PgError{Message: `duplicate key value violates unique constraint "users_email_key"`, Code: "23505"})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "internal server error" {
		t.Fatalf("expected masked error, got %q", payload["error"])
	}
}

func TestRespondKeepsClientSafeErrors(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/register", nil)

	respond(context, nil, errors.New("invalid registration input"))

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

	respond(context, nil, errors.New("graph mail test send failed: graph mail token failed: status=401 code=invalid_client message=凭证无效"))

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
