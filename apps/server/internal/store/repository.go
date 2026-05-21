package store

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/mail"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	xls "github.com/extrame/xls"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	db    *pgxpool.Pool
	redis *redis.Client
	http  *http.Client
}

var appLocation = time.FixedZone("CST", 8*3600)

const (
	defaultTenantID          = "00000000-0000-0000-0000-000000000001"
	footerSettingsKey        = "footer"
	copySettingsKey          = "copy"
	graphMailSettingsKey     = "graph_mail"
	wechatSettingsKey        = "wechat"
	dingTalkSettingsKey      = "dingtalk"
	accessControlSettingsKey = "access_control"
)

func tenantScopedDingTalkSettingsKey(tenantID string) string {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	return dingTalkSettingsKey + ":" + tenantID
}

type tenantContextKey struct{}
type notificationSourceContextKey struct{}

type TenantContext struct {
	TenantID       string
	TenantName     string
	FinanceEnabled bool
	AllTenants     bool
	Actor          Actor
}

func WithTenantContext(ctx context.Context, tenant TenantContext) context.Context {
	if tenant.TenantID == "" {
		tenant.TenantID = defaultTenantID
	}
	return context.WithValue(ctx, tenantContextKey{}, tenant)
}

func TenantFromContext(ctx context.Context) TenantContext {
	tenant, _ := ctx.Value(tenantContextKey{}).(TenantContext)
	if tenant.TenantID == "" {
		tenant.TenantID = defaultTenantID
	}
	return tenant
}

func appNow() time.Time {
	return time.Now().In(appLocation)
}

func appToday() time.Time {
	now := appNow()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, appLocation)
}

func appDateString() string {
	return appDateStringAt(appNow())
}

func appDateStringAt(t time.Time) string {
	return t.In(appLocation).Format("2006-01-02")
}

func appDateSQL() string {
	return "((now() AT TIME ZONE 'Asia/Shanghai')::date)"
}

func WithNotificationSourceContext(ctx context.Context, source string) context.Context {
	return context.WithValue(ctx, notificationSourceContextKey{}, strings.TrimSpace(source))
}

func notificationSourceFromContext(ctx context.Context) string {
	source, _ := ctx.Value(notificationSourceContextKey{}).(string)
	return strings.TrimSpace(source)
}

func dashboardCacheKey(ctx context.Context) string {
	tenant := TenantFromContext(ctx)
	if tenant.AllTenants {
		return "lirs:dashboard:all:v3"
	}
	return "lirs:dashboard:" + tenant.TenantID + ":v3"
}

type footerSettingsValue struct {
	BrandName    string          `json:"brandName"`
	BrandTagline string          `json:"brandTagline"`
	BaseURL      string          `json:"baseUrl"`
	Description  string          `json:"description"`
	Sections     []FooterSection `json:"sections"`
	Copyright    string          `json:"copyright"`
}

type copySettingsValue struct {
	Entries []CopyEntry `json:"entries"`
}

type graphMailSettingsValue struct {
	Enabled                 bool   `json:"enabled"`
	TenantID                string `json:"tenantId"`
	ClientID                string `json:"clientId"`
	ClientSecret            string `json:"clientSecret"`
	SenderUserPrincipalName string `json:"senderUserPrincipalName"`
	SaveToSentItems         bool   `json:"saveToSentItems"`
}

type wechatSettingsValue struct {
	Enabled            bool   `json:"enabled"`
	AccountType        string `json:"accountType"`
	AppID              string `json:"appId"`
	AppSecret          string `json:"appSecret"`
	ServiceAccountName string `json:"serviceAccountName"`
	TemplateID         string `json:"templateId"`
	Token              string `json:"token"`
	EncodingAESKey     string `json:"encodingAesKey"`
}

type dingTalkSettingsValue struct {
	SchemaVersion    int    `json:"schemaVersion"`
	Enabled          bool   `json:"enabled"`
	ClientID         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret"`
	CorpID           string `json:"corpId"`
	RobotCode        string `json:"robotCode"`
	OAuthRedirectURI string `json:"oauthRedirectUri"`
	EventCallbackURL string `json:"eventCallbackUrl"`
	EventAesKey      string `json:"eventAesKey"`
	EventToken       string `json:"eventToken"`
}

type accessControlSettingsValue struct {
	Enabled                bool   `json:"enabled"`
	Vendor                 string `json:"vendor"`
	Endpoint               string `json:"endpoint"`
	ClientID               string `json:"clientId"`
	ClientSecret           string `json:"clientSecret"`
	AccessGroup            string `json:"accessGroup"`
	AutoGrantOnApproval    bool   `json:"autoGrantOnApproval"`
	AutoRevokeOnCompletion bool   `json:"autoRevokeOnCompletion"`
}

func NewRepository(db *pgxpool.Pool, redisClient *redis.Client) *Repository {
	startNotificationDeliveryWorkers()
	return &Repository{db: db, redis: redisClient, http: &http.Client{Timeout: 10 * time.Second}}
}

func (r *Repository) httpClient() *http.Client {
	if r.http != nil {
		return r.http
	}
	return http.DefaultClient
}

func (r *Repository) Health(ctx context.Context) error {
	if r.db == nil {
		return errors.New("postgres is not configured")
	}
	if err := r.db.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}
	if r.redis == nil {
		return errors.New("redis is not configured")
	}
	if err := r.redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

func (r *Repository) Tenants(ctx context.Context) ([]Tenant, error) {
	rows, err := r.db.Query(ctx, `
SELECT id::text, name, code, finance_enabled, status, created_at, updated_at
FROM tenants
ORDER BY created_at DESC, name
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Tenant, 0)
	for rows.Next() {
		var item Tenant
		if err := rows.Scan(&item.ID, &item.Name, &item.Code, &item.FinanceEnabled, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveTenant(ctx context.Context, id string, input TenantInput) (Tenant, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.TrimSpace(strings.ToLower(input.Code))
	input.Status = strings.TrimSpace(input.Status)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Name == "" || !validTenantStatus(input.Status) {
		return Tenant{}, clientError("invalid tenant input")
	}

	var item Tenant
	if id == "" {
		code, err := r.generateTenantCode(ctx)
		if err != nil {
			return Tenant{}, err
		}
		err = r.db.QueryRow(ctx, `
INSERT INTO tenants (name, code, finance_enabled, status)
VALUES ($1, $2, $3, $4)
RETURNING id::text, name, code, finance_enabled, status, created_at, updated_at
`, input.Name, code, input.FinanceEnabled, input.Status).Scan(&item.ID, &item.Name, &item.Code, &item.FinanceEnabled, &item.Status, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return Tenant{}, err
		}
		r.audit(ctx, input.Actor, "tenant.create", "tenant", item.ID, "", item.Name)
		return item, nil
	}

	err := r.db.QueryRow(ctx, `
UPDATE tenants
SET name = $2, finance_enabled = $3, status = $4, updated_at = now()
WHERE id = $1
RETURNING id::text, name, code, finance_enabled, status, created_at, updated_at
`, id, input.Name, input.FinanceEnabled, input.Status).Scan(&item.ID, &item.Name, &item.Code, &item.FinanceEnabled, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return Tenant{}, err
	}
	r.audit(ctx, input.Actor, "tenant.update", "tenant", item.ID, "", fmt.Sprintf("%s/%t/%s", item.Name, item.FinanceEnabled, item.Status))
	return item, nil
}

func (r *Repository) generateTenantCode(ctx context.Context) (string, error) {
	for attempt := 0; attempt < 20; attempt++ {
		code, err := randomTenantCode()
		if err != nil {
			return "", err
		}
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenants WHERE lower(code) = lower($1))`, code).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", clientError("tenant code generation failed")
}

func (r *Repository) resolveActiveTenant(ctx context.Context, tenantID string, tenantCode string) (Tenant, error) {
	tenantID = strings.TrimSpace(tenantID)
	tenantCode = strings.TrimSpace(strings.ToLower(tenantCode))
	if tenantID == "" && tenantCode == "" {
		tenantID = defaultTenantID
	}
	rows, err := r.db.Query(ctx, `
SELECT id::text, name, code, finance_enabled, status, created_at, updated_at
FROM tenants
WHERE ($1 <> '' AND id::text = $1)
   OR ($2 <> '' AND lower(code) = $2)
ORDER BY CASE WHEN id::text = $1 THEN 0 ELSE 1 END
LIMIT 2
`, tenantID, tenantCode)
	if err != nil {
		return Tenant{}, err
	}
	defer rows.Close()

	var item Tenant
	count := 0
	for rows.Next() {
		var current Tenant
		if err := rows.Scan(&current.ID, &current.Name, &current.Code, &current.FinanceEnabled, &current.Status, &current.CreatedAt, &current.UpdatedAt); err != nil {
			return Tenant{}, err
		}
		count++
		if count == 1 {
			item = current
		}
	}
	if err := rows.Err(); err != nil {
		return Tenant{}, err
	}
	if count == 0 {
		return Tenant{}, clientError("tenant not found")
	}
	if item.Status != "active" {
		return Tenant{}, clientError("tenant is disabled")
	}
	return item, nil
}

func (r *Repository) Dashboard(ctx context.Context) (Dashboard, error) {
	tenant := TenantFromContext(ctx)
	if r.redis != nil {
		cached, err := r.redis.Get(ctx, dashboardCacheKey(ctx)).Bytes()
		if err == nil {
			var dashboard Dashboard
			if err := json.Unmarshal(cached, &dashboard); err == nil {
				return dashboard, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			slog.Warn("read dashboard cache", "error", err)
		}
	}

	var dashboard Dashboard
	err := r.db.QueryRow(ctx, `
WITH reservation_counts AS (
    SELECT
        count(*) FILTER (WHERE (lower(period) AT TIME ZONE 'Asia/Shanghai')::date = `+appDateSQL()+`)::int AS today_reservations,
        count(*) FILTER (WHERE status = 'pending')::int AS pending_approvals,
        count(*) FILTER (WHERE status = 'in_use')::int AS in_use_reservations,
        count(*) FILTER (WHERE status = 'completed')::int AS completed_reservations,
        count(*) FILTER (WHERE status IN ('completed', 'cancelled'))::int AS fulfillment_base
    FROM reservations
    WHERE ($1::boolean OR tenant_id = $2::uuid)
)
SELECT
    today_reservations,
    pending_approvals,
    in_use_reservations,
    completed_reservations,
    CASE WHEN fulfillment_base = 0 THEN 100 ELSE round(completed_reservations::numeric / fulfillment_base * 100, 1)::float8 END,
    (SELECT count(*) FROM instruments WHERE status IN ('available', 'busy') AND ($1::boolean OR tenant_id = $2::uuid)),
    COALESCE((SELECT sum(amount) FROM ledger_entries WHERE date_trunc('month', created_at) = date_trunc('month', now()) AND ($1::boolean OR tenant_id = $2::uuid)), 0)
FROM reservation_counts
`, tenant.AllTenants, tenant.TenantID).Scan(&dashboard.TodayReservations, &dashboard.PendingApprovals, &dashboard.InUseReservations, &dashboard.CompletedReservations, &dashboard.FulfillmentRate, &dashboard.ActiveInstruments, &dashboard.MonthlyRevenue)
	if err != nil {
		return dashboard, err
	}
	r.cacheDashboard(ctx, dashboard)
	return dashboard, nil
}

func (r *Repository) FooterSettings(ctx context.Context) (FooterSettings, error) {
	var key string
	var raw []byte
	var updatedBy string
	var updatedAt time.Time
	err := r.db.QueryRow(ctx, `
SELECT setting_key, value, updated_by, updated_at
FROM site_settings
WHERE setting_key = $1
`, footerSettingsKey).Scan(&key, &raw, &updatedBy, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultFooterSettings(), nil
	}
	if err != nil {
		return FooterSettings{}, err
	}

	value := defaultFooterSettingsValue()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &value); err != nil {
			slog.Warn("unmarshal footer settings", "error", err)
			value = defaultFooterSettingsValue()
		}
	}
	return footerSettingsFromValue(key, normalizeFooterSettingsValue(value), updatedBy, updatedAt), nil
}

func (r *Repository) SaveFooterSettings(ctx context.Context, input FooterSettingsInput) (FooterSettings, error) {
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	oldSettings, err := r.FooterSettings(ctx)
	if err != nil {
		return FooterSettings{}, err
	}

	value := normalizeFooterSettingsValue(footerSettingsValue{
		BrandName:    input.BrandName,
		BrandTagline: input.BrandTagline,
		BaseURL:      input.BaseURL,
		Description:  input.Description,
		Sections:     input.Sections,
		Copyright:    input.Copyright,
	})
	if value.BaseURL != "" && !validSiteBaseURL(value.BaseURL) {
		return FooterSettings{}, clientError("invalid footer settings base url")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return FooterSettings{}, err
	}

	var key string
	var raw []byte
	var updatedBy string
	var updatedAt time.Time
	err = r.db.QueryRow(ctx, `
INSERT INTO site_settings (setting_key, value, updated_by)
VALUES ($1, $2::jsonb, $3)
ON CONFLICT (setting_key) DO UPDATE
SET value = EXCLUDED.value,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
RETURNING setting_key, value, updated_by, updated_at
`, footerSettingsKey, string(payload), input.Actor).Scan(&key, &raw, &updatedBy, &updatedAt)
	if err != nil {
		return FooterSettings{}, err
	}

	savedValue := defaultFooterSettingsValue()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &savedValue); err != nil {
			return FooterSettings{}, err
		}
	}
	item := footerSettingsFromValue(key, normalizeFooterSettingsValue(savedValue), updatedBy, updatedAt)
	oldPayload, _ := json.Marshal(footerSettingsValueFromSettings(oldSettings))
	r.audit(ctx, input.Actor, "site_settings.update", "site_setting", footerSettingsKey, string(oldPayload), string(payload))
	return item, nil
}

func (r *Repository) CopySettings(ctx context.Context) (CopySettings, error) {
	var raw []byte
	var updatedBy string
	var updatedAt time.Time
	err := r.db.QueryRow(ctx, `
SELECT value, updated_by, updated_at
FROM site_settings
WHERE setting_key = $1
`, copySettingsKey).Scan(&raw, &updatedBy, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultCopySettings(), nil
	}
	if err != nil {
		return CopySettings{}, err
	}

	value := defaultCopySettingsValue()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &value); err != nil {
			slog.Warn("unmarshal copy settings", "error", err)
			value = defaultCopySettingsValue()
		}
	}
	return copySettingsFromValue(copySettingsKey, normalizeCopySettingsValue(value), updatedBy, updatedAt), nil
}

func (r *Repository) SaveCopySettings(ctx context.Context, input CopySettingsInput) (CopySettings, error) {
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	oldSettings, err := r.CopySettings(ctx)
	if err != nil {
		return CopySettings{}, err
	}

	value := normalizeCopySettingsValue(copySettingsValue{
		Entries: input.Entries,
	})
	payload, err := json.Marshal(value)
	if err != nil {
		return CopySettings{}, err
	}

	var raw []byte
	var updatedBy string
	var updatedAt time.Time
	err = r.db.QueryRow(ctx, `
INSERT INTO site_settings (setting_key, value, updated_by)
VALUES ($1, $2::jsonb, $3)
ON CONFLICT (setting_key) DO UPDATE
SET value = EXCLUDED.value,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
RETURNING value, updated_by, updated_at
`, copySettingsKey, string(payload), input.Actor).Scan(&raw, &updatedBy, &updatedAt)
	if err != nil {
		return CopySettings{}, err
	}

	savedValue := defaultCopySettingsValue()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &savedValue); err != nil {
			return CopySettings{}, err
		}
	}
	item := copySettingsFromValue(copySettingsKey, normalizeCopySettingsValue(savedValue), updatedBy, updatedAt)
	oldPayload, _ := json.Marshal(copySettingsValueFromSettings(oldSettings))
	r.audit(ctx, input.Actor, "site_settings.update", "site_setting", copySettingsKey, string(oldPayload), string(payload))
	return item, nil
}

func (r *Repository) NotificationChannelSettings(ctx context.Context) (NotificationChannelSettings, error) {
	graphMailValue, graphMailMeta, err := r.readGraphMailSettings(ctx)
	if err != nil {
		return NotificationChannelSettings{}, err
	}
	wechatValue, wechatMeta, err := r.readWeChatSettings(ctx)
	if err != nil {
		return NotificationChannelSettings{}, err
	}
	dingTalkValue, dingTalkMeta, err := r.readDingTalkSettings(ctx)
	if err != nil {
		return NotificationChannelSettings{}, err
	}
	return NotificationChannelSettings{
		GraphMail: graphMailSettingsFromValue(graphMailValue, graphMailMeta.updatedBy, graphMailMeta.updatedAt),
		WeChat:    wechatSettingsFromValue(wechatValue, wechatMeta.updatedBy, wechatMeta.updatedAt),
		DingTalk:  dingTalkSettingsFromValue(dingTalkValue, dingTalkMeta.updatedBy, dingTalkMeta.updatedAt),
	}, nil
}

func (r *Repository) DingTalkSettings(ctx context.Context) (DingTalkSettings, error) {
	value, meta, err := r.readDingTalkSettings(ctx)
	if err != nil {
		return DingTalkSettings{}, err
	}
	return dingTalkSettingsFromValue(value, meta.updatedBy, meta.updatedAt), nil
}

func (r *Repository) SaveGraphMailSettings(ctx context.Context, input GraphMailSettingsInput) (GraphMailSettings, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ClientSecret = strings.TrimSpace(input.ClientSecret)
	input.SenderUserPrincipalName = strings.TrimSpace(input.SenderUserPrincipalName)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	oldValue, _, err := r.readGraphMailSettings(ctx)
	if err != nil {
		return GraphMailSettings{}, err
	}
	if input.ClientSecret == "" {
		input.ClientSecret = oldValue.ClientSecret
	}
	if input.Enabled {
		if input.TenantID == "" || input.ClientID == "" || input.ClientSecret == "" || input.SenderUserPrincipalName == "" {
			return GraphMailSettings{}, clientError("graph mail tenant, client, secret, and sender are required")
		}
		if _, err := mail.ParseAddress(input.SenderUserPrincipalName); err != nil {
			return GraphMailSettings{}, clientError("graph mail sender email is invalid")
		}
	}
	value := graphMailSettingsValue{
		Enabled:                 input.Enabled,
		TenantID:                input.TenantID,
		ClientID:                input.ClientID,
		ClientSecret:            input.ClientSecret,
		SenderUserPrincipalName: input.SenderUserPrincipalName,
		SaveToSentItems:         input.SaveToSentItems,
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return GraphMailSettings{}, err
	}
	meta, err := r.saveJSONSetting(ctx, graphMailSettingsKey, raw, input.Actor)
	if err != nil {
		return GraphMailSettings{}, err
	}
	r.audit(ctx, input.Actor, "notification.graph_mail_settings", "site_setting", graphMailSettingsKey, "", input.TenantID+"/"+input.SenderUserPrincipalName)
	return graphMailSettingsFromValue(value, meta.updatedBy, meta.updatedAt), nil
}

func (r *Repository) TestGraphMailSettings(ctx context.Context, input GraphMailTestInput) (GraphMailTestResult, error) {
	input.To = strings.TrimSpace(input.To)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	settings, err := r.graphMailSettingsValue(ctx)
	if err != nil {
		return GraphMailTestResult{}, err
	}
	if input.To == "" {
		return GraphMailTestResult{}, clientError("graph mail test recipient is required")
	}
	if _, err := mail.ParseAddress(input.To); err != nil {
		return GraphMailTestResult{}, clientError("graph mail test recipient email is invalid")
	}
	if !settings.Enabled {
		return GraphMailTestResult{}, clientError("graph mail is not enabled")
	}
	if settings.TenantID == "" || settings.ClientID == "" || settings.ClientSecret == "" || settings.SenderUserPrincipalName == "" {
		return GraphMailTestResult{}, clientError("graph mail tenant, client, secret, and sender are required")
	}
	if err := r.sendGraphMail(ctx, settings, input.To, "实验室运营系统 Microsoft Graph 邮件测试", fmt.Sprintf("这是一封 Microsoft Graph API 发送测试邮件。\n\n发送人：%s\n发送时间：%s", input.Actor, time.Now().UTC().Format(time.RFC3339))); err != nil {
		return GraphMailTestResult{}, WrapClientError("graph mail test send failed", err)
	}
	r.audit(ctx, input.Actor, "notification.graph_mail_test", "site_setting", graphMailSettingsKey, "", input.To)
	return GraphMailTestResult{Sent: true, Message: "Microsoft Graph 测试邮件已发送。"}, nil
}

func (r *Repository) SaveWeChatSettings(ctx context.Context, input WeChatSettingsInput) (WeChatSettings, error) {
	input.AccountType = strings.TrimSpace(input.AccountType)
	input.AppID = strings.TrimSpace(input.AppID)
	input.AppSecret = strings.TrimSpace(input.AppSecret)
	input.ServiceAccountName = strings.TrimSpace(input.ServiceAccountName)
	input.TemplateID = strings.TrimSpace(input.TemplateID)
	input.Token = strings.TrimSpace(input.Token)
	input.EncodingAESKey = strings.TrimSpace(input.EncodingAESKey)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.AccountType == "" {
		input.AccountType = "service_account"
	}
	oldValue, _, err := r.readWeChatSettings(ctx)
	if err != nil {
		return WeChatSettings{}, err
	}
	if input.AppSecret == "" {
		input.AppSecret = oldValue.AppSecret
	}
	if input.Enabled && input.AppID == "" {
		return WeChatSettings{}, clientError("wechat app id is required")
	}
	value := wechatSettingsValue{
		Enabled:            input.Enabled,
		AccountType:        input.AccountType,
		AppID:              input.AppID,
		AppSecret:          input.AppSecret,
		ServiceAccountName: input.ServiceAccountName,
		TemplateID:         input.TemplateID,
		Token:              input.Token,
		EncodingAESKey:     input.EncodingAESKey,
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return WeChatSettings{}, err
	}
	meta, err := r.saveJSONSetting(ctx, wechatSettingsKey, raw, input.Actor)
	if err != nil {
		return WeChatSettings{}, err
	}
	r.audit(ctx, input.Actor, "notification.wechat_settings", "site_setting", wechatSettingsKey, "", input.AppID)
	return wechatSettingsFromValue(value, meta.updatedBy, meta.updatedAt), nil
}

func (r *Repository) SaveDingTalkSettings(ctx context.Context, input DingTalkSettingsInput) (DingTalkSettings, error) {
	tenant := TenantFromContext(ctx)
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ClientSecret = strings.TrimSpace(input.ClientSecret)
	input.CorpID = strings.TrimSpace(input.CorpID)
	input.RobotCode = strings.TrimSpace(input.RobotCode)
	input.OAuthRedirectURI = strings.TrimSpace(input.OAuthRedirectURI)
	input.EventCallbackURL = strings.TrimSpace(input.EventCallbackURL)
	input.EventAesKey = strings.TrimSpace(input.EventAesKey)
	input.EventToken = strings.TrimSpace(input.EventToken)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if tenant.TenantID == "" {
		tenant.TenantID = defaultTenantID
	}
	oldValue, _, err := r.readDingTalkSettings(ctx)
	if err != nil {
		return DingTalkSettings{}, err
	}
	if input.ClientSecret == "" {
		input.ClientSecret = oldValue.ClientSecret
	}
	if input.EventAesKey == "" {
		input.EventAesKey = oldValue.EventAesKey
	}
	if input.EventToken == "" {
		input.EventToken = oldValue.EventToken
	}
	if input.Enabled && (input.ClientID == "" || input.ClientSecret == "" || input.CorpID == "" || input.RobotCode == "") {
		return DingTalkSettings{}, clientError("dingtalk client id, client secret, corp id and robot code are required")
	}
	if input.Enabled && input.OAuthRedirectURI == "" {
		return DingTalkSettings{}, clientError("dingtalk oauth redirect uri is required")
	}
	if input.Enabled && (input.EventCallbackURL == "" || input.EventAesKey == "" || input.EventToken == "") {
		return DingTalkSettings{}, clientError("dingtalk event callback url, aes key and token are required")
	}
	if input.EventAesKey != "" && len(input.EventAesKey) != 43 {
		return DingTalkSettings{}, clientError("dingtalk event aes key must be 43 characters")
	}
	tenantCode := tenant.TenantID
	if currentTenant, err := r.resolveActiveTenant(ctx, tenant.TenantID, ""); err == nil && currentTenant.Code != "" {
		tenantCode = currentTenant.Code
	}
	input.EventCallbackURL = firstNonEmpty(generatedDingTalkEventCallbackURL(input.OAuthRedirectURI, tenantCode), input.EventCallbackURL)
	value := dingTalkSettingsValue{
		SchemaVersion:    2,
		Enabled:          input.Enabled,
		ClientID:         input.ClientID,
		ClientSecret:     input.ClientSecret,
		CorpID:           input.CorpID,
		RobotCode:        input.RobotCode,
		OAuthRedirectURI: input.OAuthRedirectURI,
		EventCallbackURL: input.EventCallbackURL,
		EventAesKey:      input.EventAesKey,
		EventToken:       input.EventToken,
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return DingTalkSettings{}, err
	}
	settingKey := tenantScopedDingTalkSettingsKey(tenant.TenantID)
	meta, err := r.saveJSONSetting(ctx, settingKey, raw, input.Actor)
	if err != nil {
		return DingTalkSettings{}, err
	}
	r.audit(WithTenantContext(ctx, TenantContext{TenantID: tenant.TenantID, TenantName: tenant.TenantName, FinanceEnabled: tenant.FinanceEnabled}), input.Actor, "notification.dingtalk_settings", "site_setting", settingKey, "", input.ClientID)
	return dingTalkSettingsFromValue(value, meta.updatedBy, meta.updatedAt), nil
}

func (r *Repository) TestDingTalkSettings(ctx context.Context, input DingTalkTestInput) (DingTalkTestResult, error) {
	tenant := TenantFromContext(ctx)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if tenant.TenantID == "" {
		tenant.TenantID = defaultTenantID
	}
	if input.UserID == "" {
		return DingTalkTestResult{}, clientError("dingtalk test user is required")
	}
	settings, err := r.dingTalkSettingsValue(ctx)
	if err != nil {
		return DingTalkTestResult{}, err
	}
	if !settings.Enabled {
		return DingTalkTestResult{}, clientError("dingtalk notification is not enabled")
	}
	if settings.ClientID == "" || settings.ClientSecret == "" || settings.CorpID == "" || settings.RobotCode == "" {
		return DingTalkTestResult{}, clientError("dingtalk client id, client secret, corp id and robot code are required")
	}
	target, err := r.dingTalkBoundUser(ctx, tenant.TenantID, input.UserID)
	if err != nil {
		return DingTalkTestResult{}, err
	}
	title := "实验室运营系统钉钉测试推送"
	body := fmt.Sprintf("这是一条钉钉企业应用测试推送。\n\n接收人：%s\n发送人：%s\n发送时间：%s", target.Name, input.Actor, time.Now().UTC().Format(time.RFC3339))
	if err := r.sendDingTalkWorkNotification(ctx, settings, target.DingTalkUserID, title, body); err != nil {
		return DingTalkTestResult{}, WrapClientError("dingtalk test send failed", err)
	}
	settingKey := tenantScopedDingTalkSettingsKey(tenant.TenantID)
	r.audit(WithTenantContext(ctx, TenantContext{TenantID: tenant.TenantID, TenantName: tenant.TenantName, FinanceEnabled: tenant.FinanceEnabled}), input.Actor, "notification.dingtalk_test", "site_setting", settingKey, "", input.UserID)
	return DingTalkTestResult{Sent: true, Message: "钉钉测试推送已发送。"}, nil
}

func (r *Repository) AccessControlSettings(ctx context.Context) (AccessControlSettings, error) {
	value, meta, err := r.readAccessControlSettings(ctx)
	if err != nil {
		return AccessControlSettings{}, err
	}
	return accessControlSettingsFromValue(value, meta.updatedBy, meta.updatedAt), nil
}

func (r *Repository) SaveAccessControlSettings(ctx context.Context, input AccessControlSettingsInput) (AccessControlSettings, error) {
	input.Vendor = strings.TrimSpace(strings.ToLower(input.Vendor))
	input.Endpoint = strings.TrimSpace(input.Endpoint)
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.ClientSecret = strings.TrimSpace(input.ClientSecret)
	input.AccessGroup = strings.TrimSpace(input.AccessGroup)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Vendor == "" {
		input.Vendor = "hikvision"
	}

	oldValue, _, err := r.readAccessControlSettings(ctx)
	if err != nil {
		return AccessControlSettings{}, err
	}
	if input.ClientSecret == "" {
		input.ClientSecret = oldValue.ClientSecret
	}
	if input.Enabled {
		if input.Endpoint == "" || input.ClientID == "" {
			return AccessControlSettings{}, clientError("access control endpoint and client id are required")
		}
		if input.ClientSecret == "" {
			return AccessControlSettings{}, clientError("access control client secret is required")
		}
	}

	value := accessControlSettingsValue{
		Enabled:                input.Enabled,
		Vendor:                 input.Vendor,
		Endpoint:               input.Endpoint,
		ClientID:               input.ClientID,
		ClientSecret:           input.ClientSecret,
		AccessGroup:            input.AccessGroup,
		AutoGrantOnApproval:    input.AutoGrantOnApproval,
		AutoRevokeOnCompletion: input.AutoRevokeOnCompletion,
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return AccessControlSettings{}, err
	}
	meta, err := r.saveJSONSetting(ctx, accessControlSettingsKey, raw, input.Actor)
	if err != nil {
		return AccessControlSettings{}, err
	}
	r.audit(ctx, input.Actor, "access_control_settings.update", "site_setting", accessControlSettingsKey, "", input.Vendor)
	return accessControlSettingsFromValue(value, meta.updatedBy, meta.updatedAt), nil
}

func (r *Repository) Instruments(ctx context.Context, filter InstrumentFilter) ([]Instrument, error) {
	tenant := TenantFromContext(ctx)
	if filter.Limit <= 0 || filter.Limit > 1000 {
		filter.Limit = 1000
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Category = strings.TrimSpace(filter.Category)
	filter.Department = strings.TrimSpace(filter.Department)
	filter.GroupName = strings.TrimSpace(filter.GroupName)
	filter.Status = strings.TrimSpace(filter.Status)

	rows, err := r.db.Query(ctx, `
SELECT i.id::text, i.tenant_id::text, i.name, i.category, i.department, i.group_name, i.status, i.location,
       i.hourly_rate::float8, i.brand, i.model, i.asset_code,
       i.access_control_enabled, i.access_control_group, i.access_control_point,
       i.description,
       i.technical_specs, i.booking_rule, i.maintenance_summary,
       i.max_booking_hours, i.min_advance_hours, i.cancel_cutoff_hours, i.checkin_window_minutes,
       i.booking_window_days, i.booking_interval_hours, i.service_start_hour, i.service_end_hour,
       COALESCE(completed_reservations.usage_count, 0) AS usage_count
FROM instruments i
LEFT JOIN (
    SELECT instrument_id, count(*)::int AS usage_count
    FROM reservations
    WHERE status = 'completed'
      AND ($8::boolean OR tenant_id = $9::uuid)
    GROUP BY instrument_id
) completed_reservations ON completed_reservations.instrument_id = i.id
WHERE ($1 = '' OR i.name ILIKE '%' || $1 || '%' OR i.model ILIKE '%' || $1 || '%' OR i.asset_code ILIKE '%' || $1 || '%' OR i.category ILIKE '%' || $1 || '%')
  AND ($2 = '' OR i.category = $2)
  AND ($3 = '' OR i.department = $3)
  AND ($4 = '' OR i.group_name = $4)
  AND ($5 = '' OR i.status = $5)
  AND ($8::boolean OR i.tenant_id = $9::uuid)
ORDER BY i.created_at, i.name
LIMIT $6 OFFSET $7
`, filter.Search, filter.Category, filter.Department, filter.GroupName, filter.Status, filter.Limit, filter.Offset, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Instrument, 0)
	for rows.Next() {
		item, err := scanInstrument(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Instrument(ctx context.Context, id string) (Instrument, error) {
	tenant := TenantFromContext(ctx)
	row := r.db.QueryRow(ctx, `
SELECT i.id::text, i.tenant_id::text, i.name, i.category, i.department, i.group_name, i.status, i.location,
       i.hourly_rate::float8, i.brand, i.model, i.asset_code,
       i.access_control_enabled, i.access_control_group, i.access_control_point,
       i.description,
       i.technical_specs, i.booking_rule, i.maintenance_summary,
       i.max_booking_hours, i.min_advance_hours, i.cancel_cutoff_hours, i.checkin_window_minutes,
       i.booking_window_days, i.booking_interval_hours, i.service_start_hour, i.service_end_hour,
       (
           SELECT count(*)::int
           FROM reservations r
           WHERE r.instrument_id = i.id AND r.status = 'completed'
       ) AS usage_count
FROM instruments i
WHERE i.id = $1
  AND ($2::boolean OR i.tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID)
	return scanInstrument(row)
}

func (r *Repository) SaveInstrument(ctx context.Context, id string, input InstrumentInput) (Instrument, error) {
	tenant := TenantFromContext(ctx)
	input = normalizeInstrument(input)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Name == "" || input.Category == "" || input.Department == "" || input.Location == "" || input.HourlyRate < 0 {
		return Instrument{}, clientError("invalid instrument input")
	}
	if input.Status == "" {
		input.Status = "available"
	}
	if !validInstrumentStatus(input.Status) {
		return Instrument{}, clientError("invalid instrument status")
	}
	if input.GroupName != "" {
		var teamExists bool
		if err := r.db.QueryRow(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM organization_units
    WHERE kind = 'group'
      AND name = $1
      AND (parent_name = '' OR parent_name = $2)
      AND ($3::boolean OR tenant_id = $4::uuid)
)
`, input.GroupName, input.Department, tenant.AllTenants, tenant.TenantID).Scan(&teamExists); err != nil {
			return Instrument{}, err
		}
		if !teamExists {
			return Instrument{}, clientError("instrument team must belong to selected department")
		}
	}

	if id == "" {
		var item Instrument
		err := r.db.QueryRow(ctx, `
INSERT INTO instruments (tenant_id, name, category, department, group_name, status, location, hourly_rate, brand, model, asset_code, access_control_enabled, access_control_group, access_control_point, description, technical_specs, booking_rule, maintenance_summary, max_booking_hours, min_advance_hours, cancel_cutoff_hours, checkin_window_minutes, booking_window_days, booking_interval_hours, service_start_hour, service_end_hour)
VALUES ($26, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
RETURNING id::text, tenant_id::text, name, category, department, group_name, status, location, hourly_rate::float8, brand, model, asset_code, access_control_enabled, access_control_group, access_control_point, description, technical_specs, booking_rule, maintenance_summary, max_booking_hours, min_advance_hours, cancel_cutoff_hours, checkin_window_minutes, booking_window_days, booking_interval_hours, service_start_hour, service_end_hour, 0
`, input.Name, input.Category, input.Department, input.GroupName, input.Status, input.Location, input.HourlyRate, input.Brand, input.Model, input.AssetCode, input.AccessControlEnabled, input.AccessControlGroup, input.AccessControlPoint, input.Description, input.TechnicalSpecs, input.BookingRule, input.MaintenanceSummary, input.MaxBookingHours, input.MinAdvanceHours, input.CancelCutoffHours, input.CheckinWindowMins, input.BookingWindowDays, input.BookingIntervalHours, input.ServiceStartHour, input.ServiceEndHour, tenant.TenantID).Scan(
			&item.ID, &item.TenantID, &item.Name, &item.Category, &item.Department, &item.GroupName, &item.Status, &item.Location,
			&item.HourlyRate, &item.Brand, &item.Model, &item.AssetCode, &item.AccessControlEnabled, &item.AccessControlGroup, &item.AccessControlPoint, &item.Description,
			&item.TechnicalSpecs, &item.BookingRule, &item.MaintenanceSummary,
			&item.MaxBookingHours, &item.MinAdvanceHours, &item.CancelCutoffHours, &item.CheckinWindowMins,
			&item.BookingWindowDays, &item.BookingIntervalHours, &item.ServiceStartHour, &item.ServiceEndHour,
			&item.UsageCount,
		)
		if err != nil {
			return Instrument{}, err
		}
		r.audit(ctx, input.Actor, "instrument.create", "instrument", item.ID, "", item.Name)
		r.invalidateDashboard(ctx)
		return item, nil
	}

	oldItem, err := r.Instrument(ctx, id)
	if err != nil {
		return Instrument{}, err
	}
	var item Instrument
	err = r.db.QueryRow(ctx, `
UPDATE instruments
SET name = $2, category = $3, department = $4, group_name = $5, status = $6, location = $7,
    hourly_rate = $8, brand = $9, model = $10, asset_code = $11,
    access_control_enabled = $12, access_control_group = $13, access_control_point = $14,
    description = $15, technical_specs = $16, booking_rule = $17, maintenance_summary = $18,
    max_booking_hours = $19, min_advance_hours = $20, cancel_cutoff_hours = $21, checkin_window_minutes = $22,
    booking_window_days = $23, booking_interval_hours = $24, service_start_hour = $25, service_end_hour = $26
WHERE id = $1 AND ($27::boolean OR tenant_id = $28::uuid)
RETURNING id::text, tenant_id::text, name, category, department, group_name, status, location, hourly_rate::float8, brand, model, asset_code, access_control_enabled, access_control_group, access_control_point, description, technical_specs, booking_rule, maintenance_summary,
          max_booking_hours, min_advance_hours, cancel_cutoff_hours, checkin_window_minutes, booking_window_days, booking_interval_hours, service_start_hour, service_end_hour,
          (SELECT count(*)::int FROM reservations WHERE instrument_id = instruments.id AND status = 'completed')
`, id, input.Name, input.Category, input.Department, input.GroupName, input.Status, input.Location, input.HourlyRate, input.Brand, input.Model, input.AssetCode, input.AccessControlEnabled, input.AccessControlGroup, input.AccessControlPoint, input.Description, input.TechnicalSpecs, input.BookingRule, input.MaintenanceSummary, input.MaxBookingHours, input.MinAdvanceHours, input.CancelCutoffHours, input.CheckinWindowMins, input.BookingWindowDays, input.BookingIntervalHours, input.ServiceStartHour, input.ServiceEndHour, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Category, &item.Department, &item.GroupName, &item.Status, &item.Location,
		&item.HourlyRate, &item.Brand, &item.Model, &item.AssetCode, &item.AccessControlEnabled, &item.AccessControlGroup, &item.AccessControlPoint, &item.Description,
		&item.TechnicalSpecs, &item.BookingRule, &item.MaintenanceSummary,
		&item.MaxBookingHours, &item.MinAdvanceHours, &item.CancelCutoffHours, &item.CheckinWindowMins,
		&item.BookingWindowDays, &item.BookingIntervalHours, &item.ServiceStartHour, &item.ServiceEndHour,
		&item.UsageCount,
	)
	if err != nil {
		return Instrument{}, err
	}
	r.audit(ctx, input.Actor, "instrument.update", "instrument", item.ID, oldItem.Status, item.Status)
	r.invalidateDashboard(ctx)
	return item, nil
}

func (r *Repository) DeleteInstrument(ctx context.Context, id string, actor string) (Instrument, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	oldItem, err := r.Instrument(ctx, id)
	if err != nil {
		return Instrument{}, err
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Instrument{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	type instrumentDeletionNotice struct {
		tenantID  string
		userID    string
		userName  string
		groupName string
	}
	notices := make([]instrumentDeletionNotice, 0)
	reservationRows, err := tx.Query(ctx, `
UPDATE reservations
SET status = 'cancelled',
    cancel_reason = '关联仪器已删除',
    cancelled_at = now(),
    instrument_id = NULL
WHERE instrument_id = $1
  AND status IN ('pending', 'approved', 'in_use')
  AND ($2::boolean OR tenant_id = $3::uuid)
RETURNING tenant_id::text, COALESCE(user_id::text, ''), user_name, group_name
`, id, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return Instrument{}, err
	}
	for reservationRows.Next() {
		var reservationTenantID string
		var userID string
		var userName string
		var groupName string
		if err := reservationRows.Scan(&reservationTenantID, &userID, &userName, &groupName); err != nil {
			reservationRows.Close()
			return Instrument{}, err
		}
		if userID == "" {
			continue
		}
		notices = append(notices, instrumentDeletionNotice{tenantID: reservationTenantID, userID: userID, userName: userName, groupName: groupName})
	}
	reservationRows.Close()
	if err := reservationRows.Err(); err != nil {
		return Instrument{}, err
	}
	notifications := make([]Notification, 0, len(notices))
	for _, notice := range notices {
		notification, err := r.createNotificationTx(ctx, tx, notice.tenantID, notice.userID, notice.groupName, "", "personal", "预约状态更新", fmt.Sprintf("%s 的 %s 预约状态已更新为已取消，原因：关联仪器已删除。", notice.userName, oldItem.Name), "warning")
		if err != nil {
			return Instrument{}, err
		}
		notifications = append(notifications, notification)
	}
	if _, err := tx.Exec(ctx, `
UPDATE reservations
SET instrument_id = NULL
WHERE instrument_id = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID); err != nil {
		return Instrument{}, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE maintenance_orders
SET instrument_id = NULL,
    status = CASE WHEN status IN ('reported', 'assigned', 'in_progress') THEN 'cancelled' ELSE status END,
    result = CASE WHEN status IN ('reported', 'assigned', 'in_progress') AND result = '' THEN '关联仪器已删除' ELSE result END
WHERE instrument_id = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID); err != nil {
		return Instrument{}, err
	}
	for _, query := range []string{
		`UPDATE training_courses SET instrument_id = NULL, updated_at = now() WHERE instrument_id = $1 AND ($2::boolean OR tenant_id = $3::uuid)`,
		`UPDATE training_authorizations SET instrument_id = NULL, updated_at = now() WHERE instrument_id = $1 AND ($2::boolean OR tenant_id = $3::uuid)`,
		`UPDATE training_practical_assessments SET instrument_id = NULL, updated_at = now() WHERE instrument_id = $1 AND ($2::boolean OR tenant_id = $3::uuid)`,
		`UPDATE lims_tasks SET instrument_id = NULL, updated_at = now() WHERE instrument_id = $1 AND ($2::boolean OR tenant_id = $3::uuid)`,
		`UPDATE iot_devices SET instrument_id = NULL, updated_at = now() WHERE instrument_id = $1 AND ($2::boolean OR tenant_id = $3::uuid)`,
		`DELETE FROM training_rules WHERE instrument_id = $1 AND ($2::boolean OR tenant_id = $3::uuid)`,
	} {
		if _, err := tx.Exec(ctx, query, id, tenant.AllTenants, tenant.TenantID); err != nil {
			return Instrument{}, err
		}
	}
	tag, err := tx.Exec(ctx, `
DELETE FROM instruments
WHERE id = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return Instrument{}, err
	}
	if tag.RowsAffected() == 0 {
		return Instrument{}, pgx.ErrNoRows
	}
	if err := tx.Commit(ctx); err != nil {
		return Instrument{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	r.invalidateDashboard(ctx)
	r.audit(ctx, actor, "instrument.delete", "instrument", oldItem.ID, oldItem.Status, "deleted")
	return oldItem, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanInstrument(row scanner) (Instrument, error) {
	var item Instrument
	err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Category,
		&item.Department,
		&item.GroupName,
		&item.Status,
		&item.Location,
		&item.HourlyRate,
		&item.Brand,
		&item.Model,
		&item.AssetCode,
		&item.AccessControlEnabled,
		&item.AccessControlGroup,
		&item.AccessControlPoint,
		&item.Description,
		&item.TechnicalSpecs,
		&item.BookingRule,
		&item.MaintenanceSummary,
		&item.MaxBookingHours,
		&item.MinAdvanceHours,
		&item.CancelCutoffHours,
		&item.CheckinWindowMins,
		&item.BookingWindowDays,
		&item.BookingIntervalHours,
		&item.ServiceStartHour,
		&item.ServiceEndHour,
		&item.UsageCount,
	)
	return item, err
}

func scanReservation(row scanner) (Reservation, error) {
	var item Reservation
	err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.UserID,
		&item.InstrumentID,
		&item.InstrumentName,
		&item.UserName,
		&item.GroupName,
		&item.Purpose,
		&item.StartTime,
		&item.EndTime,
		&item.Status,
		&item.Fee,
	)
	return item, err
}

func (r *Repository) Reservations(ctx context.Context) ([]Reservation, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT r.id::text, r.tenant_id::text, COALESCE(r.user_id::text, ''), COALESCE(r.instrument_id::text, ''), COALESCE(i.name, '已删除仪器'),
       r.user_name, r.group_name, r.purpose, lower(r.period), upper(r.period), r.status, r.fee::float8
FROM reservations r
LEFT JOIN instruments i ON i.id = r.instrument_id
WHERE ($1::boolean OR r.tenant_id = $2::uuid)
ORDER BY lower(r.period) DESC
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Reservation, 0)
	for rows.Next() {
		item, err := scanReservation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Reservation(ctx context.Context, id string) (Reservation, error) {
	tenant := TenantFromContext(ctx)
	return scanReservation(r.db.QueryRow(ctx, `
SELECT r.id::text, r.tenant_id::text, COALESCE(r.user_id::text, ''), COALESCE(r.instrument_id::text, ''), COALESCE(i.name, '已删除仪器'),
       r.user_name, r.group_name, r.purpose, lower(r.period), upper(r.period), r.status, r.fee::float8
FROM reservations r
LEFT JOIN instruments i ON i.id = r.instrument_id
WHERE r.id = $1
  AND ($2::boolean OR r.tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID))
}

func (r *Repository) Users(ctx context.Context) ([]User, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT u.id::text, u.tenant_id::text, t.name, t.code, u.name, u.email, u.phone, u.department, u.group_name, u.role, u.status, u.email_verified,
       u.dingtalk_user_id, u.dingtalk_union_id, u.dingtalk_name, u.dingtalk_user_id <> '',
       t.finance_enabled, u.auth_epoch
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE ($1::boolean OR u.tenant_id = $2::uuid)
  AND u.status <> 'deleted'
ORDER BY u.created_at DESC
`, tenant.AllTenants, tenant.TenantID)
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

func (r *Repository) OrganizationUnits(ctx context.Context, kind string) ([]OrganizationUnit, error) {
	tenant := TenantFromContext(ctx)
	kind = strings.TrimSpace(kind)
	if kind != "" && !validOrganizationUnitKind(kind) {
		return nil, clientError("invalid organization unit kind")
	}
	rows, err := r.db.Query(ctx, `
SELECT id::text, kind, name, parent_name, created_at, updated_at
FROM organization_units
WHERE ($1 = '' OR kind = $1)
  AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY kind, parent_name, name
`, kind, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]OrganizationUnit, 0)
	for rows.Next() {
		var item OrganizationUnit
		if err := rows.Scan(&item.ID, &item.Kind, &item.Name, &item.ParentName, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveOrganizationUnit(ctx context.Context, id string, input OrganizationUnitInput) (OrganizationUnit, error) {
	tenant := TenantFromContext(ctx)
	input.Kind = strings.TrimSpace(input.Kind)
	input.Name = strings.TrimSpace(input.Name)
	input.ParentName = strings.TrimSpace(input.ParentName)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if !validOrganizationUnitKind(input.Kind) || input.Name == "" {
		return OrganizationUnit{}, clientError("invalid organization unit input")
	}
	if input.Kind == "department" {
		input.ParentName = ""
	}
	if input.Kind == "group" && input.ParentName == "" {
		return OrganizationUnit{}, clientError("organization unit parent department is required")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return OrganizationUnit{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if input.Kind == "group" {
		var parentExists bool
		if err := tx.QueryRow(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM organization_units
    WHERE kind = 'department'
      AND name = $1
      AND ($2::boolean OR tenant_id = $3::uuid)
)
`, input.ParentName, tenant.AllTenants, tenant.TenantID).Scan(&parentExists); err != nil {
			return OrganizationUnit{}, err
		}
		if !parentExists {
			return OrganizationUnit{}, clientError("organization unit parent department is required")
		}
	}

	if id == "" {
		var item OrganizationUnit
		err := tx.QueryRow(ctx, `
INSERT INTO organization_units (tenant_id, kind, name, parent_name)
VALUES ($4, $1, $2, $3)
RETURNING id::text, kind, name, parent_name, created_at, updated_at
`, input.Kind, input.Name, input.ParentName, tenant.TenantID).Scan(&item.ID, &item.Kind, &item.Name, &item.ParentName, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return OrganizationUnit{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return OrganizationUnit{}, err
		}
		r.audit(ctx, input.Actor, "organization_unit.create", "organization_unit", item.ID, "", item.Kind+":"+item.Name)
		return item, nil
	}

	var oldUnit OrganizationUnit
	if err := tx.QueryRow(ctx, `
SELECT id::text, kind, name, parent_name, created_at, updated_at
FROM organization_units
WHERE id = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
FOR UPDATE
`, id, tenant.AllTenants, tenant.TenantID).Scan(&oldUnit.ID, &oldUnit.Kind, &oldUnit.Name, &oldUnit.ParentName, &oldUnit.CreatedAt, &oldUnit.UpdatedAt); err != nil {
		return OrganizationUnit{}, err
	}
	if oldUnit.Kind != input.Kind {
		return OrganizationUnit{}, clientError("organization unit kind cannot be changed")
	}

	var item OrganizationUnit
	err = tx.QueryRow(ctx, `
UPDATE organization_units
SET name = $2, parent_name = $3, updated_at = now()
WHERE id = $1
RETURNING id::text, kind, name, parent_name, created_at, updated_at
`, id, input.Name, input.ParentName).Scan(&item.ID, &item.Kind, &item.Name, &item.ParentName, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return OrganizationUnit{}, err
	}
	if oldUnit.Name != item.Name || oldUnit.ParentName != item.ParentName {
		if err := r.updateOrganizationUnitReferences(ctx, tx, oldUnit, item); err != nil {
			return OrganizationUnit{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return OrganizationUnit{}, err
	}
	r.audit(ctx, input.Actor, "organization_unit.update", "organization_unit", item.ID, oldUnit.Kind+":"+oldUnit.Name, item.Kind+":"+item.Name)
	return item, nil
}

func (r *Repository) DeleteOrganizationUnit(ctx context.Context, id string, actor string) (OrganizationUnit, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return OrganizationUnit{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var item OrganizationUnit
	if err := tx.QueryRow(ctx, `
SELECT id::text, kind, name, parent_name, created_at, updated_at
FROM organization_units
WHERE id = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
FOR UPDATE
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &item.Kind, &item.Name, &item.ParentName, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return OrganizationUnit{}, err
	}

	instrumentCount, dependentCount, err := r.organizationUnitDeletionUsage(ctx, tx, item)
	if err != nil {
		return OrganizationUnit{}, err
	}
	if instrumentCount > 0 {
		return OrganizationUnit{}, fmt.Errorf("organization unit still has %d instruments", instrumentCount)
	}
	if dependentCount > 0 {
		return OrganizationUnit{}, fmt.Errorf("organization unit still has %d dependent records", dependentCount)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM organization_units WHERE id = $1`, item.ID); err != nil {
		return OrganizationUnit{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return OrganizationUnit{}, err
	}
	r.audit(ctx, actor, "organization_unit.delete", "organization_unit", item.ID, item.Kind+":"+item.Name, "")
	return item, nil
}

func (r *Repository) ReviewUser(ctx context.Context, id string, input UserReviewInput) (User, error) {
	tenant := TenantFromContext(ctx)
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.Role = strings.TrimSpace(input.Role)
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.Department = strings.TrimSpace(input.Department)
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Status = strings.TrimSpace(input.Status)
	input.Actor = strings.TrimSpace(input.Actor)
	input.ActorRole = strings.TrimSpace(input.ActorRole)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if !validRole(input.Role) || !validUserStatus(input.Status) {
		return User{}, clientError("invalid user review input")
	}
	if input.Status == "deleted" {
		return User{}, clientError("user status cannot be changed to deleted")
	}

	var oldTenantID, oldRole, oldStatus, oldGroup, oldDepartment, oldEmail, oldPhone string
	if err := r.db.QueryRow(ctx, `SELECT tenant_id::text, role, status, group_name, department, email, phone FROM users WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)`, id, tenant.AllTenants, tenant.TenantID).Scan(&oldTenantID, &oldRole, &oldStatus, &oldGroup, &oldDepartment, &oldEmail, &oldPhone); err != nil {
		return User{}, err
	}
	if input.TenantID == "" {
		input.TenantID = oldTenantID
	}
	if input.TenantID != oldTenantID && input.ActorRole != "super_admin" {
		return User{}, clientError("only system super admin can change tenant")
	}
	if !canActorManageUserRole(input.ActorRole, oldRole, input.Role) {
		return User{}, clientError("only system super admin can manage administrator roles")
	}
	if input.GroupName == "" {
		input.GroupName = oldGroup
	}
	if input.Department == "" {
		input.Department = oldDepartment
	}
	if input.Email == "" {
		input.Email = oldEmail
	}
	if input.Phone == "" {
		input.Phone = oldPhone
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return User{}, clientError("user email is invalid")
	}
	if input.Phone == "" {
		return User{}, clientError("user phone is required")
	}
	targetTenant, err := r.resolveActiveTenant(ctx, input.TenantID, "")
	if err != nil {
		return User{}, err
	}

	var duplicateEmailCount int
	if err := r.db.QueryRow(ctx, `
SELECT count(*)
FROM users
WHERE lower(email) = lower($1) AND id <> $2
  AND tenant_id = $3::uuid
`, input.Email, id, targetTenant.ID).Scan(&duplicateEmailCount); err != nil {
		return User{}, err
	}
	if duplicateEmailCount > 0 {
		return User{}, clientError("user email already exists")
	}

	var user User
	err = r.db.QueryRow(ctx, `
UPDATE users
SET tenant_id = $2,
    role = $3,
    group_name = $4,
    department = $5,
    status = $6,
    email = $7,
    phone = $8,
    email_verified = CASE WHEN $6 = 'active' THEN true ELSE email_verified END,
    auth_epoch = auth_epoch + 1,
    updated_at = now()
WHERE id = $1
  AND ($9::boolean OR tenant_id = $10::uuid)
RETURNING id::text, tenant_id::text, (SELECT name FROM tenants WHERE id = users.tenant_id), (SELECT code FROM tenants WHERE id = users.tenant_id), name, email, phone, department, group_name, role, status, email_verified,
          dingtalk_user_id, dingtalk_union_id, dingtalk_name, dingtalk_user_id <> '',
          (SELECT finance_enabled FROM tenants WHERE id = users.tenant_id), auth_epoch
`, id, targetTenant.ID, input.Role, input.GroupName, input.Department, input.Status, input.Email, input.Phone, tenant.AllTenants, tenant.TenantID).Scan(
		&user.ID,
		&user.TenantID,
		&user.TenantName,
		&user.TenantCode,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.Department,
		&user.GroupName,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.DingTalkUserID,
		&user.DingTalkUnionID,
		&user.DingTalkName,
		&user.DingTalkBound,
		&user.FinanceEnabled,
		&user.AuthEpoch,
	)
	if err != nil {
		return User{}, err
	}
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: user.TenantID, TenantName: user.TenantName, FinanceEnabled: user.FinanceEnabled})
	r.audit(auditCtx, input.Actor, "user.review", "user", user.ID, fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", oldTenantID, oldRole, oldStatus, oldGroup, oldDepartment, oldEmail, oldPhone), fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", user.TenantID, user.Role, user.Status, user.GroupName, user.Department, user.Email, user.Phone))
	_, err = r.createNotification(ctx, user.TenantID, user.ID, user.GroupName, user.Department, "personal", "账号状态更新", fmt.Sprintf("%s 的账号状态已更新为%s，角色为%s。", user.Name, userStatusLabel(user.Status), roleName(user.Role)), notificationLevelForStatus(user.Status))
	if err != nil {
		return User{}, err
	}
	r.invalidateDashboard(ctx)
	return user, nil
}

func (r *Repository) SaveUserMembership(ctx context.Context, id string, input UserMembershipInput) (User, error) {
	tenant := TenantFromContext(ctx)
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.Role = strings.TrimSpace(input.Role)
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.Department = strings.TrimSpace(input.Department)
	input.Status = strings.TrimSpace(input.Status)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.TenantID == "" || !validRole(input.Role) || !validUserStatus(input.Status) {
		return User{}, clientError("invalid user membership input")
	}
	if input.Role == "super_admin" {
		return User{}, clientError("user membership cannot use system super admin role")
	}

	var sourceTenantID, sourceName, sourceEmail, sourcePhone, sourceDepartment, sourceGroup, sourcePasswordHash string
	var sourceEmailVerified bool
	if err := r.db.QueryRow(ctx, `
SELECT tenant_id::text, name, email, phone, department, group_name, password_hash, email_verified
FROM users
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID).Scan(&sourceTenantID, &sourceName, &sourceEmail, &sourcePhone, &sourceDepartment, &sourceGroup, &sourcePasswordHash, &sourceEmailVerified); err != nil {
		return User{}, err
	}
	if _, err := mail.ParseAddress(sourceEmail); err != nil {
		return User{}, clientError("user email is invalid")
	}
	if input.Department == "" {
		input.Department = sourceDepartment
	}
	if input.GroupName == "" {
		input.GroupName = sourceGroup
	}
	if input.GroupName == "" {
		input.GroupName = "未分配归属"
	}
	targetTenant, err := r.resolveActiveTenant(ctx, input.TenantID, "")
	if err != nil {
		return User{}, err
	}

	var user User
	err = r.db.QueryRow(ctx, `
INSERT INTO users (tenant_id, name, email, phone, department, group_name, password_hash, role, status, email_verified)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (tenant_id, lower(email)) DO UPDATE
SET name = EXCLUDED.name,
    phone = EXCLUDED.phone,
    department = EXCLUDED.department,
    group_name = EXCLUDED.group_name,
    password_hash = EXCLUDED.password_hash,
    role = EXCLUDED.role,
    status = EXCLUDED.status,
    email_verified = users.email_verified OR EXCLUDED.email_verified,
    auth_epoch = users.auth_epoch + 1,
    updated_at = now()
RETURNING id::text, tenant_id::text, (SELECT name FROM tenants WHERE id = users.tenant_id), (SELECT code FROM tenants WHERE id = users.tenant_id), name, email, phone, department, group_name, role, status, email_verified,
          dingtalk_user_id, dingtalk_union_id, dingtalk_name, dingtalk_user_id <> '',
          (SELECT finance_enabled FROM tenants WHERE id = users.tenant_id), auth_epoch
`, targetTenant.ID, sourceName, sourceEmail, sourcePhone, input.Department, input.GroupName, sourcePasswordHash, input.Role, input.Status, sourceEmailVerified).Scan(
		&user.ID,
		&user.TenantID,
		&user.TenantName,
		&user.TenantCode,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.Department,
		&user.GroupName,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.DingTalkUserID,
		&user.DingTalkUnionID,
		&user.DingTalkName,
		&user.DingTalkBound,
		&user.FinanceEnabled,
		&user.AuthEpoch,
	)
	if err != nil {
		return User{}, err
	}
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: user.TenantID, TenantName: user.TenantName, FinanceEnabled: user.FinanceEnabled})
	r.audit(auditCtx, input.Actor, "user.membership.save", "user", user.ID, sourceTenantID+"/"+sourceEmail, fmt.Sprintf("%s/%s/%s", user.TenantID, user.Role, user.Status))
	_, err = r.createNotification(ctx, user.TenantID, user.ID, user.GroupName, user.Department, "personal", "机构权限更新", fmt.Sprintf("%s 在%s的机构权限已更新为%s，账号状态为%s。", user.Name, user.TenantName, roleName(user.Role), userStatusLabel(user.Status)), notificationLevelForStatus(user.Status))
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *Repository) CreateUser(ctx context.Context, input UserCreateInput) (User, error) {
	tenant := TenantFromContext(ctx)
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)
	input.Department = strings.TrimSpace(input.Department)
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.Role = strings.TrimSpace(input.Role)
	input.Status = strings.TrimSpace(input.Status)
	input.Actor = strings.TrimSpace(input.Actor)
	input.ActorRole = strings.TrimSpace(input.ActorRole)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.TenantID == "" {
		input.TenantID = tenant.TenantID
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.GroupName == "" {
		input.GroupName = "未分配归属"
	}
	if input.Name == "" || input.Email == "" || input.Phone == "" || input.Department == "" || len(input.Password) < 8 || !validRole(input.Role) || !validUserStatus(input.Status) {
		return User{}, clientError("invalid user create input")
	}
	if input.Status == "deleted" {
		return User{}, clientError("invalid user create status")
	}
	if input.ActorRole != "super_admin" && input.TenantID != tenant.TenantID {
		return User{}, clientError("only system super admin can create users for another tenant")
	}
	if !canActorManageUserRole(input.ActorRole, "unassigned", input.Role) {
		return User{}, clientError("only system super admin can manage administrator roles")
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return User{}, clientError("user email is invalid")
	}
	targetTenant, err := r.resolveActiveTenant(ctx, input.TenantID, "")
	if err != nil {
		return User{}, err
	}
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return User{}, err
	}

	var user User
	err = r.db.QueryRow(ctx, `
INSERT INTO users (tenant_id, name, email, phone, department, group_name, password_hash, role, status, email_verified)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true)
RETURNING id::text, tenant_id::text, (SELECT name FROM tenants WHERE id = users.tenant_id), (SELECT code FROM tenants WHERE id = users.tenant_id), name, email, phone, department, group_name, role, status, email_verified,
          dingtalk_user_id, dingtalk_union_id, dingtalk_name, dingtalk_user_id <> '',
          (SELECT finance_enabled FROM tenants WHERE id = users.tenant_id), auth_epoch
`, targetTenant.ID, input.Name, input.Email, input.Phone, input.Department, input.GroupName, passwordHash, input.Role, input.Status).Scan(
		&user.ID,
		&user.TenantID,
		&user.TenantName,
		&user.TenantCode,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.Department,
		&user.GroupName,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.DingTalkUserID,
		&user.DingTalkUnionID,
		&user.DingTalkName,
		&user.DingTalkBound,
		&user.FinanceEnabled,
		&user.AuthEpoch,
	)
	if err != nil {
		return User{}, err
	}
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: user.TenantID, TenantName: user.TenantName, FinanceEnabled: user.FinanceEnabled})
	r.audit(auditCtx, input.Actor, "user.create", "user", user.ID, "", fmt.Sprintf("%s/%s/%s", user.Email, user.Role, user.Status))
	_, err = r.createNotification(ctx, user.TenantID, user.ID, user.GroupName, user.Department, "personal", "账号已创建", fmt.Sprintf("%s 的账号已由管理员创建，角色为%s，账号状态为%s。", user.Name, roleName(user.Role), userStatusLabel(user.Status)), notificationLevelForStatus(user.Status))
	if err != nil {
		return User{}, err
	}
	r.invalidateDashboard(ctx)
	return user, nil
}

func (r *Repository) DeleteUser(ctx context.Context, id string, actor string) (User, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return User{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var oldStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM users WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid) FOR UPDATE`, id, tenant.AllTenants, tenant.TenantID).Scan(&oldStatus); err != nil {
		return User{}, err
	}

	var user User
	err = tx.QueryRow(ctx, `
UPDATE users
SET status = 'deleted',
    dingtalk_user_id = '',
    dingtalk_union_id = '',
    dingtalk_name = '',
    dingtalk_bound_at = NULL,
    auth_epoch = auth_epoch + 1,
    updated_at = now()
WHERE id = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
RETURNING id::text, tenant_id::text, (SELECT name FROM tenants WHERE id = users.tenant_id), (SELECT code FROM tenants WHERE id = users.tenant_id), name, email, phone, department, group_name, role, status, email_verified,
          dingtalk_user_id, dingtalk_union_id, dingtalk_name, dingtalk_user_id <> '',
          (SELECT finance_enabled FROM tenants WHERE id = users.tenant_id), auth_epoch
`, id, tenant.AllTenants, tenant.TenantID).Scan(
		&user.ID,
		&user.TenantID,
		&user.TenantName,
		&user.TenantCode,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.Department,
		&user.GroupName,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.DingTalkUserID,
		&user.DingTalkUnionID,
		&user.DingTalkName,
		&user.DingTalkBound,
		&user.FinanceEnabled,
		&user.AuthEpoch,
	)
	if err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, id); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	r.audit(ctx, actor, "user.delete", "user", user.ID, oldStatus, user.Status)
	return user, nil
}

func (r *Repository) UpdateCurrentUserProfile(ctx context.Context, id string, input UserProfileInput) (User, error) {
	tenant := TenantFromContext(ctx)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}

	var oldUser User
	if err := r.db.QueryRow(ctx, `
SELECT u.id::text, u.tenant_id::text, t.name, t.code, u.name, u.email, u.phone, u.department, u.group_name, u.role, u.status, u.email_verified,
       u.dingtalk_user_id, u.dingtalk_union_id, u.dingtalk_name, u.dingtalk_user_id <> '',
       t.finance_enabled, u.auth_epoch
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE u.id = $1 AND ($2::boolean OR u.tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID).Scan(&oldUser.ID, &oldUser.TenantID, &oldUser.TenantName, &oldUser.TenantCode, &oldUser.Name, &oldUser.Email, &oldUser.Phone, &oldUser.Department, &oldUser.GroupName, &oldUser.Role, &oldUser.Status, &oldUser.EmailVerified, &oldUser.DingTalkUserID, &oldUser.DingTalkUnionID, &oldUser.DingTalkName, &oldUser.DingTalkBound, &oldUser.FinanceEnabled, &oldUser.AuthEpoch); err != nil {
		return User{}, err
	}

	if profileFieldChanged(input.Name, oldUser.Name) ||
		profileFieldChanged(input.Phone, oldUser.Phone) ||
		profileFieldChanged(input.Department, oldUser.Department) ||
		profileFieldChanged(input.GroupName, oldUser.GroupName) {
		return User{}, clientError("user profile identity fields are managed by administrators")
	}

	return oldUser, nil
}

func (r *Repository) CurrentUserDingTalkBinding(ctx context.Context, id string) (DingTalkBinding, error) {
	tenant := TenantFromContext(ctx)
	var binding DingTalkBinding
	err := r.db.QueryRow(ctx, `
SELECT dingtalk_user_id <> '', dingtalk_user_id, dingtalk_union_id, dingtalk_name, COALESCE(dingtalk_bound_at, '0001-01-01 00:00:00+00'::timestamptz), updated_at
FROM users
WHERE id = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID).Scan(&binding.Bound, &binding.UserID, &binding.UnionID, &binding.Name, &binding.BoundAt, &binding.UpdatedAt)
	if err != nil {
		return DingTalkBinding{}, err
	}
	settings, err := r.dingTalkSettingsValue(ctx)
	if err == nil && settings.Enabled && settings.ClientID != "" && settings.ClientSecret != "" && settings.OAuthRedirectURI != "" {
		state, tokenErr := randomToken()
		if tokenErr == nil {
			if err := r.saveDingTalkOAuthState(ctx, tenant.TenantID, id, state); err == nil {
				binding.State = state
				binding.AuthURL = dingTalkOAuthURL(settings, state)
			} else {
				slog.Warn("save dingtalk oauth state", "userId", id, "error", err)
			}
		}
	}
	return binding, nil
}

func (r *Repository) BindCurrentUserDingTalk(ctx context.Context, id string, input DingTalkBindingInput) (DingTalkBinding, error) {
	tenant := TenantFromContext(ctx)
	input.AuthCode = strings.TrimSpace(input.AuthCode)
	input.State = strings.TrimSpace(input.State)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	settings, err := r.dingTalkSettingsValue(ctx)
	if err != nil {
		return DingTalkBinding{}, err
	}
	if !settings.Enabled || settings.ClientID == "" || settings.ClientSecret == "" {
		return DingTalkBinding{}, clientError("dingtalk application is not configured")
	}

	if input.AuthCode == "" {
		return DingTalkBinding{}, clientError("dingtalk auth code is required")
	}
	if err := r.consumeDingTalkOAuthState(ctx, tenant.TenantID, id, input.State); err != nil {
		return DingTalkBinding{}, err
	}
	identity, err := r.dingTalkIdentityByAuthCode(ctx, settings, input.AuthCode)
	if err != nil {
		return DingTalkBinding{}, err
	}
	identity.UserID = strings.TrimSpace(identity.UserID)
	identity.UnionID = strings.TrimSpace(identity.UnionID)
	identity.Name = strings.TrimSpace(identity.Name)
	if identity.UserID == "" {
		return DingTalkBinding{}, clientError("dingtalk user id is required")
	}

	var binding DingTalkBinding
	err = r.db.QueryRow(ctx, `
UPDATE users
SET dingtalk_user_id = $2,
    dingtalk_union_id = $3,
    dingtalk_name = $4,
    dingtalk_bound_at = now(),
    updated_at = now()
WHERE id = $1
  AND ($5::boolean OR tenant_id = $6::uuid)
RETURNING true, dingtalk_user_id, dingtalk_union_id, dingtalk_name, dingtalk_bound_at, updated_at
`, id, identity.UserID, identity.UnionID, firstNonEmpty(identity.Name, input.Actor), tenant.AllTenants, tenant.TenantID).Scan(&binding.Bound, &binding.UserID, &binding.UnionID, &binding.Name, &binding.BoundAt, &binding.UpdatedAt)
	if err != nil {
		return DingTalkBinding{}, err
	}
	r.audit(ctx, input.Actor, "user.dingtalk_bind", "user", id, "", binding.UserID)
	return binding, nil
}

func (r *Repository) UnbindCurrentUserDingTalk(ctx context.Context, id string, actor string) (DingTalkBinding, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var oldUserID string
	var updatedAt time.Time
	err := r.db.QueryRow(ctx, `
WITH old_user AS (
    SELECT dingtalk_user_id
    FROM users
    WHERE id = $1
      AND ($2::boolean OR tenant_id = $3::uuid)
),
updated AS (
UPDATE users
SET dingtalk_user_id = '',
    dingtalk_union_id = '',
    dingtalk_name = '',
    dingtalk_bound_at = NULL,
    updated_at = now()
WHERE id = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
RETURNING updated_at
)
SELECT COALESCE((SELECT dingtalk_user_id FROM old_user), ''), updated_at FROM updated
`, id, tenant.AllTenants, tenant.TenantID).Scan(&oldUserID, &updatedAt)
	if err != nil {
		return DingTalkBinding{}, err
	}
	r.audit(ctx, actor, "user.dingtalk_unbind", "user", id, oldUserID, "")
	return DingTalkBinding{Bound: false, UpdatedAt: updatedAt}, nil
}

func (r *Repository) ChangePassword(ctx context.Context, id string, input PasswordChangeInput) error {
	input.CurrentPassword = strings.TrimSpace(input.CurrentPassword)
	input.NewPassword = strings.TrimSpace(input.NewPassword)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.CurrentPassword == "" || len(input.NewPassword) < 8 {
		return clientError("invalid password input")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var name, email, passwordHash string
	if err := tx.QueryRow(ctx, `SELECT name, email, password_hash FROM users WHERE id = $1`, id).Scan(&name, &email, &passwordHash); err != nil {
		return err
	}
	if !passwordMatches(passwordHash, input.CurrentPassword) {
		return clientError("current password is incorrect")
	}

	newHash, err := hashPassword(input.NewPassword)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
UPDATE users
SET password_hash = $2,
    auth_epoch = auth_epoch + 1,
    updated_at = now()
WHERE lower(email) = lower($1)
`, email, newHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
UPDATE sessions
SET revoked_at = now()
WHERE user_id IN (SELECT id FROM users WHERE lower(email) = lower($1))
  AND revoked_at IS NULL
`, email); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	r.audit(ctx, input.Actor, "user.password_change", "user", id, name, "password_updated")
	return nil
}

func (r *Repository) Register(ctx context.Context, input RegisterInput) (User, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.TenantCode = strings.TrimSpace(strings.ToLower(input.TenantCode))
	input.AccountType = strings.TrimSpace(input.AccountType)
	if input.AccountType == "" {
		input.AccountType = "user"
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)
	input.Department = strings.TrimSpace(input.Department)
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)
	if input.Name == "" || input.Email == "" || input.Phone == "" || input.Department == "" || len(input.Password) < 8 || input.VerificationCode == "" || !validRegisterAccountType(input.AccountType) {
		return User{}, clientError("invalid registration input")
	}
	tenant, err := r.resolveActiveTenant(ctx, input.TenantID, input.TenantCode)
	if err != nil {
		return User{}, err
	}
	input.TenantID = tenant.ID
	requestedRole := "unassigned"
	requestLabel := "普通用户"

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return User{}, err
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return User{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var codeID string
	err = tx.QueryRow(ctx, `
SELECT id::text
FROM email_verification_codes
WHERE tenant_id = $1
  AND lower(email) = lower($2)
  AND code_hash = $3
  AND consumed_at IS NULL
  AND expires_at > now()
ORDER BY created_at DESC
LIMIT 1
FOR UPDATE
`, input.TenantID, input.Email, verificationCodeHash(input.VerificationCode)).Scan(&codeID)
	if err != nil {
		return User{}, clientError("email verification code is invalid or expired")
	}

	var user User
	err = tx.QueryRow(ctx, `
INSERT INTO users (tenant_id, name, email, phone, department, password_hash, role, email_verified, email_verification_token)
VALUES ($1, $2, $3, $4, $5, $6, $7, true, '')
RETURNING id::text, tenant_id::text, (SELECT name FROM tenants WHERE id = users.tenant_id), (SELECT code FROM tenants WHERE id = users.tenant_id), name, email, phone, department, group_name, role, status, email_verified,
          dingtalk_user_id, dingtalk_union_id, dingtalk_name, dingtalk_user_id <> '',
          (SELECT finance_enabled FROM tenants WHERE id = users.tenant_id), auth_epoch
`, input.TenantID, input.Name, input.Email, input.Phone, input.Department, passwordHash, requestedRole).Scan(
		&user.ID,
		&user.TenantID,
		&user.TenantName,
		&user.TenantCode,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.Department,
		&user.GroupName,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.DingTalkUserID,
		&user.DingTalkUnionID,
		&user.DingTalkName,
		&user.DingTalkBound,
		&user.FinanceEnabled,
		&user.AuthEpoch,
	)
	if err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE email_verification_codes SET consumed_at = now() WHERE id = $1`, codeID); err != nil {
		return User{}, err
	}

	notification, err := r.createNotificationTx(ctx, tx, user.TenantID, "", "", "", "global", "账号状态更新", fmt.Sprintf("%s 已提交%s注册申请，所属机构：%s，当前状态：待审核。", user.Name, requestLabel, tenant.Name), "info")
	if err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	r.enqueueNotificationDelivery(notification)
	r.invalidateDashboard(ctx)
	r.enqueueEvent(ctx, "user.registered", map[string]any{
		"userId":      user.ID,
		"tenantId":    user.TenantID,
		"accountType": input.AccountType,
		"name":        user.Name,
		"email":       user.Email,
		"status":      user.Status,
	})
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: user.TenantID, TenantName: user.TenantName, FinanceEnabled: user.FinanceEnabled})
	r.audit(auditCtx, "system", "user.register", "user", user.ID, "", user.Status)
	return user, nil
}

func (r *Repository) RequestEmailVerificationCode(ctx context.Context, input EmailVerificationCodeInput) (EmailVerificationCodeResponse, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.TenantCode = strings.TrimSpace(strings.ToLower(input.TenantCode))
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if input.Email == "" {
		return EmailVerificationCodeResponse{}, clientError("invalid email verification input")
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return EmailVerificationCodeResponse{}, clientError("email address is invalid")
	}
	tenant, err := r.resolveActiveTenant(ctx, input.TenantID, input.TenantCode)
	if err != nil {
		return EmailVerificationCodeResponse{}, err
	}
	code, err := randomNumericCode(6)
	if err != nil {
		return EmailVerificationCodeResponse{}, err
	}
	_, err = r.db.Exec(ctx, `
INSERT INTO email_verification_codes (tenant_id, email, code_hash, expires_at)
VALUES ($1, $2, $3, now() + interval '10 minutes')
`, tenant.ID, input.Email, verificationCodeHash(code))
	if err != nil {
		return EmailVerificationCodeResponse{}, err
	}
	settings, err := r.graphMailSettingsValue(ctx)
	if err != nil {
		return EmailVerificationCodeResponse{}, err
	}
	if !graphMailReady(settings) {
		slog.Warn("email verification code generated but graph mail is not enabled", "email", input.Email, "code", code)
		return EmailVerificationCodeResponse{Sent: false, Message: "验证码已生成，但 Microsoft Graph 邮件尚未启用，请在后台配置 Graph API 邮件通道。"}, nil
	}
	if err := r.sendGraphMail(ctx, settings, input.Email, "实验室运营系统注册验证码", fmt.Sprintf("您的注册验证码是：%s。10 分钟内有效。", code)); err != nil {
		return EmailVerificationCodeResponse{}, WrapClientError("email verification send failed", err)
	}
	return EmailVerificationCodeResponse{Sent: true, Message: "验证码已发送，请检查邮箱。"}, nil
}

type dingTalkIdentity struct {
	UserID  string
	UnionID string
	Name    string
	Mobile  string
}

func (r *Repository) VerifyEmail(ctx context.Context, token string) (User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return User{}, clientError("invalid verification token")
	}
	var user User
	err := r.db.QueryRow(ctx, `
UPDATE users
SET email_verified = true, email_verification_token = '', updated_at = now()
WHERE email_verification_token = $1
RETURNING id::text, tenant_id::text, (SELECT name FROM tenants WHERE id = users.tenant_id), (SELECT code FROM tenants WHERE id = users.tenant_id), name, email, phone, department, group_name, role, status, email_verified,
          dingtalk_user_id, dingtalk_union_id, dingtalk_name, dingtalk_user_id <> '',
          (SELECT finance_enabled FROM tenants WHERE id = users.tenant_id), auth_epoch
`, token).Scan(&user.ID, &user.TenantID, &user.TenantName, &user.TenantCode, &user.Name, &user.Email, &user.Phone, &user.Department, &user.GroupName, &user.Role, &user.Status, &user.EmailVerified, &user.DingTalkUserID, &user.DingTalkUnionID, &user.DingTalkName, &user.DingTalkBound, &user.FinanceEnabled, &user.AuthEpoch)
	if err != nil {
		return User{}, err
	}
	r.audit(ctx, user.Name, "user.email_verified", "user", user.ID, "false", "true")
	return user, nil
}

func (r *Repository) Login(ctx context.Context, input LoginInput) (AuthResponse, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.TenantCode = strings.TrimSpace(strings.ToLower(input.TenantCode))
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Device = strings.TrimSpace(input.Device)
	if input.Email == "" || input.Password == "" {
		return AuthResponse{}, clientError("invalid login input")
	}

	rows, err := r.db.Query(ctx, `
SELECT u.id::text, u.tenant_id::text, t.name, t.code, u.name, u.email, u.phone, u.department, u.group_name, u.role, u.status, u.email_verified,
       u.dingtalk_user_id, u.dingtalk_union_id, u.dingtalk_name, u.dingtalk_user_id <> '',
       t.finance_enabled, u.auth_epoch, u.password_hash
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE lower(u.email) = lower($1)
  AND ($2 = '' OR u.tenant_id = $2::uuid)
  AND ($3 = '' OR lower(t.code) = $3)
  AND t.status = 'active'
ORDER BY u.created_at DESC
LIMIT 2
`, input.Email, input.TenantID, input.TenantCode)
	if err != nil {
		return AuthResponse{}, err
	}
	defer rows.Close()

	var user User
	var passwordHash string
	count := 0
	for rows.Next() {
		count++
		if count == 1 {
			if err := rows.Scan(&user.ID, &user.TenantID, &user.TenantName, &user.TenantCode, &user.Name, &user.Email, &user.Phone, &user.Department, &user.GroupName, &user.Role, &user.Status, &user.EmailVerified, &user.DingTalkUserID, &user.DingTalkUnionID, &user.DingTalkName, &user.DingTalkBound, &user.FinanceEnabled, &user.AuthEpoch, &passwordHash); err != nil {
				return AuthResponse{}, err
			}
			continue
		}
		var discard User
		var discardHash string
		if err := rows.Scan(&discard.ID, &discard.TenantID, &discard.TenantName, &discard.TenantCode, &discard.Name, &discard.Email, &discard.Phone, &discard.Department, &discard.GroupName, &discard.Role, &discard.Status, &discard.EmailVerified, &discard.DingTalkUserID, &discard.DingTalkUnionID, &discard.DingTalkName, &discard.DingTalkBound, &discard.FinanceEnabled, &discard.AuthEpoch, &discardHash); err != nil {
			return AuthResponse{}, err
		}
	}
	if err := rows.Err(); err != nil {
		return AuthResponse{}, err
	}
	if count == 0 {
		return AuthResponse{}, clientError("invalid email, password, or institution")
	}
	if count > 1 && input.TenantID == "" && input.TenantCode == "" {
		return AuthResponse{}, clientError("tenant is required when this email exists in multiple tenants")
	}
	if !passwordMatches(passwordHash, input.Password) {
		return AuthResponse{}, clientError("invalid email or password")
	}
	if user.Status == "disabled" {
		return AuthResponse{}, clientError("account is disabled")
	}

	return r.createUserSession(ctx, user, firstNonEmpty(input.Device, "web"), "auth.login")
}

func (r *Repository) DingTalkQuickLogin(ctx context.Context, input DingTalkQuickLoginInput) (AuthResponse, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.TenantCode = strings.TrimSpace(strings.ToLower(input.TenantCode))
	input.AuthCode = strings.TrimSpace(input.AuthCode)
	input.CorpID = strings.TrimSpace(input.CorpID)
	input.Device = strings.TrimSpace(input.Device)
	if input.AuthCode == "" {
		return AuthResponse{}, clientError("dingtalk auth code is required")
	}

	tenant, settings, err := r.dingTalkQuickLoginTenantSettings(ctx, input)
	if err != nil {
		return AuthResponse{}, err
	}
	if !settings.Enabled || settings.ClientID == "" || settings.ClientSecret == "" || settings.CorpID == "" {
		return AuthResponse{}, clientError("dingtalk application is not configured")
	}

	identity, err := r.dingTalkIdentityByQuickAuthCode(ctx, settings, input.AuthCode)
	if err != nil {
		return AuthResponse{}, err
	}
	user, err := r.userByDingTalkIdentity(ctx, tenant.ID, identity)
	if err != nil {
		return AuthResponse{}, err
	}
	if user.Status == "disabled" {
		return AuthResponse{}, clientError("account is disabled")
	}
	return r.createUserSession(ctx, user, firstNonEmpty(input.Device, "dingtalk"), "auth.dingtalk_quick_login")
}

func (r *Repository) DingTalkWebLoginIntent(ctx context.Context, input DingTalkWebLoginIntentInput) (DingTalkWebLoginIntent, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.TenantCode = strings.TrimSpace(strings.ToLower(input.TenantCode))
	input.RedirectURI = strings.TrimSpace(input.RedirectURI)
	if input.RedirectURI == "" {
		return DingTalkWebLoginIntent{}, clientError("dingtalk redirect uri is required")
	}
	tenant, settings, err := r.dingTalkQuickLoginTenantSettings(ctx, DingTalkQuickLoginInput{
		TenantID:   input.TenantID,
		TenantCode: input.TenantCode,
	})
	if err != nil {
		return DingTalkWebLoginIntent{}, err
	}
	settings.OAuthRedirectURI = input.RedirectURI
	state, err := randomToken()
	if err != nil {
		return DingTalkWebLoginIntent{}, err
	}
	if err := r.saveDingTalkWebLoginState(ctx, tenant.ID, state); err != nil {
		return DingTalkWebLoginIntent{}, err
	}
	return DingTalkWebLoginIntent{
		AuthURL:    dingTalkOAuthURL(settings, state),
		State:      state,
		TenantID:   tenant.ID,
		TenantCode: tenant.Code,
	}, nil
}

func (r *Repository) DingTalkWebLogin(ctx context.Context, input DingTalkWebLoginInput) (DingTalkWebLoginResult, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.TenantCode = strings.TrimSpace(strings.ToLower(input.TenantCode))
	input.AuthCode = strings.TrimSpace(input.AuthCode)
	input.State = strings.TrimSpace(input.State)
	input.Device = strings.TrimSpace(input.Device)
	if input.AuthCode == "" {
		return DingTalkWebLoginResult{}, clientError("dingtalk auth code is required")
	}

	stateTenantID, err := r.consumeDingTalkWebLoginState(ctx, input.State)
	if err != nil {
		return DingTalkWebLoginResult{}, err
	}
	tenant, settings, err := r.dingTalkQuickLoginTenantSettings(ctx, DingTalkQuickLoginInput{
		TenantID:   firstNonEmpty(input.TenantID, stateTenantID),
		TenantCode: input.TenantCode,
	})
	if err != nil {
		return DingTalkWebLoginResult{}, err
	}
	identity, err := r.dingTalkIdentityByAuthCode(ctx, settings, input.AuthCode)
	if err != nil {
		return DingTalkWebLoginResult{}, err
	}
	user, err := r.userByDingTalkIdentity(ctx, tenant.ID, identity)
	if err != nil {
		if err.Error() != "dingtalk account is not bound to a LIRS user" {
			return DingTalkWebLoginResult{}, err
		}
		token, tokenErr := r.saveDingTalkLoginBindingIntent(ctx, dingTalkLoginBindingIntentValue{
			TenantID:     tenant.ID,
			TenantCode:   tenant.Code,
			Identity:     identity,
			DingTalkName: firstNonEmpty(identity.Name, identity.UserID, identity.UnionID),
		})
		if tokenErr != nil {
			return DingTalkWebLoginResult{}, tokenErr
		}
		return DingTalkWebLoginResult{
			Bound:        false,
			BindingToken: token,
			TenantID:     tenant.ID,
			TenantCode:   tenant.Code,
			DingTalkName: firstNonEmpty(identity.Name, identity.UserID, identity.UnionID),
		}, nil
	}
	if user.Status == "disabled" {
		return DingTalkWebLoginResult{}, clientError("account is disabled")
	}
	auth, err := r.createUserSession(ctx, user, firstNonEmpty(input.Device, "dingtalk-web"), "auth.dingtalk_web_login")
	if err != nil {
		return DingTalkWebLoginResult{}, err
	}
	return DingTalkWebLoginResult{Bound: true, Auth: &auth, TenantID: tenant.ID, TenantCode: tenant.Code, DingTalkName: firstNonEmpty(identity.Name, user.DingTalkName)}, nil
}

func (r *Repository) BindDingTalkLoginToExistingUser(ctx context.Context, input DingTalkLoginBindExistingInput) (AuthResponse, error) {
	input.BindingToken = strings.TrimSpace(input.BindingToken)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Device = strings.TrimSpace(input.Device)
	if input.BindingToken == "" || input.Email == "" || input.Password == "" {
		return AuthResponse{}, clientError("invalid dingtalk binding input")
	}
	intent, err := r.consumeDingTalkLoginBindingIntent(ctx, input.BindingToken)
	if err != nil {
		return AuthResponse{}, err
	}
	user, passwordHash, err := r.loginUserForTenant(ctx, intent.TenantID, input.Email)
	if err != nil {
		return AuthResponse{}, err
	}
	if !passwordMatches(passwordHash, input.Password) {
		return AuthResponse{}, clientError("invalid email or password")
	}
	if user.Status == "disabled" {
		return AuthResponse{}, clientError("account is disabled")
	}
	identity := intent.Identity
	identity.UserID = strings.TrimSpace(identity.UserID)
	identity.UnionID = strings.TrimSpace(identity.UnionID)
	identity.Name = strings.TrimSpace(identity.Name)
	if identity.UserID == "" {
		return AuthResponse{}, clientError("dingtalk user id is required")
	}
	if _, err := r.db.Exec(ctx, `
UPDATE users
SET dingtalk_user_id = $2,
    dingtalk_union_id = $3,
    dingtalk_name = $4,
    dingtalk_bound_at = now(),
    updated_at = now()
WHERE id = $1
  AND tenant_id = $5::uuid
`, user.ID, identity.UserID, identity.UnionID, firstNonEmpty(identity.Name, user.Name), intent.TenantID); err != nil {
		return AuthResponse{}, err
	}
	user.DingTalkUserID = identity.UserID
	user.DingTalkUnionID = identity.UnionID
	user.DingTalkName = firstNonEmpty(identity.Name, user.Name)
	user.DingTalkBound = true
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: user.TenantID, TenantName: user.TenantName, FinanceEnabled: user.FinanceEnabled})
	r.audit(auditCtx, user.Name, "user.dingtalk_login_bind", "user", user.ID, "", identity.UserID)
	return r.createUserSession(ctx, user, firstNonEmpty(input.Device, "dingtalk-web-bind"), "auth.dingtalk_login_bind")
}

func (r *Repository) dingTalkQuickLoginTenantSettings(ctx context.Context, input DingTalkQuickLoginInput) (Tenant, dingTalkSettingsValue, error) {
	tenants, err := r.Tenants(ctx)
	if err != nil {
		return Tenant{}, dingTalkSettingsValue{}, err
	}
	candidates := tenants
	if input.TenantID != "" || input.TenantCode != "" {
		candidates = make([]Tenant, 0, 1)
		for _, tenant := range tenants {
			if input.TenantID != "" && tenant.ID != input.TenantID {
				continue
			}
			if input.TenantCode != "" && strings.ToLower(tenant.Code) != input.TenantCode {
				continue
			}
			candidates = append(candidates, tenant)
		}
	}
	if len(candidates) == 0 {
		return Tenant{}, dingTalkSettingsValue{}, clientError("tenant not found")
	}

	var matchedTenant Tenant
	var matchedSettings dingTalkSettingsValue
	matches := 0
	for _, tenant := range candidates {
		if tenant.Status != "active" {
			continue
		}
		settings, _, err := r.readDingTalkSettingsByTenantID(ctx, tenant.ID)
		if err != nil {
			return Tenant{}, dingTalkSettingsValue{}, err
		}
		if !settings.Enabled || settings.ClientID == "" || settings.ClientSecret == "" || settings.CorpID == "" {
			continue
		}
		if input.CorpID != "" && settings.CorpID != input.CorpID {
			continue
		}
		matches++
		matchedTenant = tenant
		matchedSettings = settings
	}
	if matches == 0 {
		return Tenant{}, dingTalkSettingsValue{}, clientError("dingtalk application is not configured")
	}
	if matches > 1 {
		return Tenant{}, dingTalkSettingsValue{}, clientError("tenant is required for dingtalk login")
	}
	return matchedTenant, matchedSettings, nil
}

func (r *Repository) userByDingTalkIdentity(ctx context.Context, tenantID string, identity dingTalkIdentity) (User, error) {
	identity.UserID = strings.TrimSpace(identity.UserID)
	identity.UnionID = strings.TrimSpace(identity.UnionID)
	if identity.UserID == "" && identity.UnionID == "" {
		return User{}, clientError("dingtalk user id is required")
	}
	var user User
	err := r.db.QueryRow(ctx, `
SELECT u.id::text, u.tenant_id::text, t.name, t.code, u.name, u.email, u.phone, u.department, u.group_name, u.role, u.status, u.email_verified,
       u.dingtalk_user_id, u.dingtalk_union_id, u.dingtalk_name, u.dingtalk_user_id <> '',
       t.finance_enabled, u.auth_epoch
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE u.tenant_id = $1::uuid
  AND u.dingtalk_user_id <> ''
  AND (u.dingtalk_user_id = $2 OR ($3 <> '' AND u.dingtalk_union_id = $3))
  AND t.status = 'active'
ORDER BY u.updated_at DESC
LIMIT 1
`, tenantID, identity.UserID, identity.UnionID).Scan(&user.ID, &user.TenantID, &user.TenantName, &user.TenantCode, &user.Name, &user.Email, &user.Phone, &user.Department, &user.GroupName, &user.Role, &user.Status, &user.EmailVerified, &user.DingTalkUserID, &user.DingTalkUnionID, &user.DingTalkName, &user.DingTalkBound, &user.FinanceEnabled, &user.AuthEpoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, clientError("dingtalk account is not bound to a LIRS user")
	}
	return user, err
}

func (r *Repository) loginUserForTenant(ctx context.Context, tenantID string, email string) (User, string, error) {
	tenantID = strings.TrimSpace(tenantID)
	email = strings.TrimSpace(strings.ToLower(email))
	if tenantID == "" || email == "" {
		return User{}, "", clientError("invalid login input")
	}
	var user User
	var passwordHash string
	err := r.db.QueryRow(ctx, `
SELECT u.id::text, u.tenant_id::text, t.name, t.code, u.name, u.email, u.phone, u.department, u.group_name, u.role, u.status, u.email_verified,
       u.dingtalk_user_id, u.dingtalk_union_id, u.dingtalk_name, u.dingtalk_user_id <> '',
       t.finance_enabled, u.auth_epoch, u.password_hash
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE u.tenant_id = $1::uuid
  AND lower(u.email) = lower($2)
  AND t.status = 'active'
ORDER BY u.created_at DESC
LIMIT 1
`, tenantID, email).Scan(&user.ID, &user.TenantID, &user.TenantName, &user.TenantCode, &user.Name, &user.Email, &user.Phone, &user.Department, &user.GroupName, &user.Role, &user.Status, &user.EmailVerified, &user.DingTalkUserID, &user.DingTalkUnionID, &user.DingTalkName, &user.DingTalkBound, &user.FinanceEnabled, &user.AuthEpoch, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", clientError("invalid email, password, or institution")
	}
	return user, passwordHash, err
}

func (r *Repository) createUserSession(ctx context.Context, user User, device string, action string) (AuthResponse, error) {
	device = strings.TrimSpace(device)
	if device == "" {
		device = "web"
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = "auth.login"
	}
	token, err := randomToken()
	if err != nil {
		return AuthResponse{}, err
	}
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	_, err = r.db.Exec(ctx, `
INSERT INTO sessions (user_id, token_hash, auth_epoch, device_info, expires_at)
VALUES ($1, $2, $3, $4, $5)
`, user.ID, tokenHash(token), user.AuthEpoch, device, expiresAt)
	if err != nil {
		return AuthResponse{}, err
	}
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: user.TenantID, TenantName: user.TenantName, FinanceEnabled: user.FinanceEnabled})
	r.audit(auditCtx, user.Name, action, "user", user.ID, "", device)
	return AuthResponse{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

func (r *Repository) CurrentUser(ctx context.Context, token string) (User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return User{}, clientError("missing token")
	}
	hash := tokenHash(token)
	var user User
	err := r.db.QueryRow(ctx, `
SELECT u.id::text, u.tenant_id::text, t.name, t.code, u.name, u.email, u.phone, u.department, u.group_name, u.role, u.status, u.email_verified,
       u.dingtalk_user_id, u.dingtalk_union_id, u.dingtalk_name, u.dingtalk_user_id <> '',
       t.finance_enabled, u.auth_epoch
FROM sessions s
JOIN users u ON u.id = s.user_id
JOIN tenants t ON t.id = u.tenant_id
WHERE s.token_hash = $1
  AND s.revoked_at IS NULL
  AND s.expires_at > now()
  AND s.auth_epoch = u.auth_epoch
  AND u.status <> 'disabled'
  AND t.status = 'active'
`, hash).Scan(&user.ID, &user.TenantID, &user.TenantName, &user.TenantCode, &user.Name, &user.Email, &user.Phone, &user.Department, &user.GroupName, &user.Role, &user.Status, &user.EmailVerified, &user.DingTalkUserID, &user.DingTalkUnionID, &user.DingTalkName, &user.DingTalkBound, &user.FinanceEnabled, &user.AuthEpoch)
	if err != nil {
		return User{}, err
	}
	_, _ = r.db.Exec(ctx, `UPDATE sessions SET last_used_at = now() WHERE token_hash = $1`, hash)
	return user, nil
}

func (r *Repository) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return clientError("missing token")
	}
	hash := tokenHash(token)
	_, err := r.db.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`, hash)
	return err
}

func (r *Repository) LogoutAll(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return clientError("missing user")
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE users SET auth_epoch = auth_epoch + 1, updated_at = now() WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	r.audit(ctx, "system", "auth.logout_all", "user", userID, "", "revoked")
	return nil
}

func (r *Repository) CreateReservation(ctx context.Context, input ReservationInput) (Reservation, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Reservation{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	notifications := make([]Notification, 0, 1)
	reservation, err := r.createReservationInTx(ctx, tx, input, &notifications)
	if err != nil {
		return Reservation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	r.invalidateDashboard(ctx)
	r.enqueueEvent(ctx, "reservation.created", map[string]any{
		"reservationId": reservation.ID,
		"instrumentId":  reservation.InstrumentID,
		"userName":      reservation.UserName,
		"status":        reservation.Status,
	})
	r.audit(ctx, reservation.UserName, "reservation.create", "reservation", reservation.ID, "", reservation.Status)
	return reservation, nil
}

func (r *Repository) CreateReservationBatch(ctx context.Context, input ReservationBatchInput) ([]Reservation, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.UserName = strings.TrimSpace(input.UserName)
	input.Purpose = strings.TrimSpace(input.Purpose)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.InstrumentID == "" || input.UserName == "" || input.Purpose == "" || len(input.Periods) == 0 {
		return nil, clientError("invalid reservation batch input")
	}
	if len(input.Periods) > 72 {
		return nil, clientError("reservation batch has too many periods")
	}

	normalizedPeriods := make([]ReservationPeriodInput, 0, len(input.Periods))
	totalDuration := time.Duration(0)
	for _, period := range input.Periods {
		if !period.EndTime.After(period.StartTime) {
			return nil, clientError("invalid reservation period")
		}
		if period.EndTime.Sub(period.StartTime) < time.Hour {
			return nil, clientError("minimum reservation duration is one hour")
		}
		if !isHourAligned(period.StartTime) || !isHourAligned(period.EndTime) {
			return nil, clientError("reservation time must use whole-hour granularity")
		}
		totalDuration += period.EndTime.Sub(period.StartTime)
		normalizedPeriods = append(normalizedPeriods, period)
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var maxBookingHours int
	err = tx.QueryRow(ctx, `
SELECT max_booking_hours
FROM instruments
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.InstrumentID, TenantFromContext(ctx).AllTenants, TenantFromContext(ctx).TenantID).Scan(&maxBookingHours)
	if err != nil {
		return nil, err
	}
	if maxBookingHours > 0 && totalDuration > time.Duration(maxBookingHours)*time.Hour {
		return nil, fmt.Errorf("reservation exceeds maximum duration of %d hours", maxBookingHours)
	}

	reservations := make([]Reservation, 0, len(normalizedPeriods))
	notifications := make([]Notification, 0, len(normalizedPeriods))
	for index, period := range normalizedPeriods {
		key := input.IdempotencyKey
		if key != "" {
			key = fmt.Sprintf("%s:%d:%s:%s", key, index, period.StartTime.UTC().Format(time.RFC3339), period.EndTime.UTC().Format(time.RFC3339))
		}
		reservation, err := r.createReservationInTx(ctx, tx, ReservationInput{
			InstrumentID:   input.InstrumentID,
			UserID:         input.UserID,
			UserName:       input.UserName,
			Purpose:        input.Purpose,
			StartTime:      period.StartTime,
			EndTime:        period.EndTime,
			IdempotencyKey: key,
		}, &notifications)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, reservation)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	r.invalidateDashboard(ctx)
	r.enqueueDingTalkNotifications(notifications...)
	for _, reservation := range reservations {
		r.enqueueEvent(ctx, "reservation.created", map[string]any{
			"reservationId": reservation.ID,
			"instrumentId":  reservation.InstrumentID,
			"userName":      reservation.UserName,
			"status":        reservation.Status,
		})
		r.audit(ctx, reservation.UserName, "reservation.create", "reservation", reservation.ID, "", reservation.Status)
	}
	return reservations, nil
}

func (r *Repository) createReservationInTx(ctx context.Context, tx pgx.Tx, input ReservationInput, notifications *[]Notification) (Reservation, error) {
	tenant := TenantFromContext(ctx)
	input.UserID = strings.TrimSpace(input.UserID)
	input.UserName = strings.TrimSpace(input.UserName)
	input.Purpose = strings.TrimSpace(input.Purpose)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.InstrumentID == "" || input.UserName == "" || input.Purpose == "" || !input.EndTime.After(input.StartTime) {
		return Reservation{}, clientError("invalid reservation input")
	}
	if input.EndTime.Sub(input.StartTime) < time.Hour {
		return Reservation{}, clientError("minimum reservation duration is one hour")
	}
	if !isHourAligned(input.StartTime) || !isHourAligned(input.EndTime) {
		return Reservation{}, clientError("reservation time must use whole-hour granularity")
	}

	if input.IdempotencyKey == "" {
		input.IdempotencyKey = fmt.Sprintf("%s:%s:%s:%s:%s", input.InstrumentID, strings.ToLower(input.UserName), strings.ToLower(input.Purpose), input.StartTime.UTC().Format(time.RFC3339), input.EndTime.UTC().Format(time.RFC3339))
	}

	var hourlyRate float64
	var instrumentName string
	var instrumentStatus string
	var maxBookingHours int
	var minAdvanceHours int
	var bookingWindowDays int
	var bookingIntervalHours int
	var serviceStartHour int
	var serviceEndHour int
	err := tx.QueryRow(ctx, `
SELECT name, status, hourly_rate::float8, max_booking_hours, min_advance_hours, booking_window_days, booking_interval_hours, service_start_hour, service_end_hour
FROM instruments
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.InstrumentID, tenant.AllTenants, tenant.TenantID).Scan(&instrumentName, &instrumentStatus, &hourlyRate, &maxBookingHours, &minAdvanceHours, &bookingWindowDays, &bookingIntervalHours, &serviceStartHour, &serviceEndHour)
	if err != nil {
		return Reservation{}, err
	}
	serviceStartHour, serviceEndHour = normalizeServiceHours(serviceStartHour, serviceEndHour)
	if instrumentStatus == "maintenance" || instrumentStatus == "disabled" {
		return Reservation{}, clientError("instrument is not bookable")
	}
	if maxBookingHours > 0 && input.EndTime.Sub(input.StartTime) > time.Duration(maxBookingHours)*time.Hour {
		return Reservation{}, fmt.Errorf("reservation exceeds maximum duration of %d hours", maxBookingHours)
	}
	if minAdvanceHours > 0 && input.StartTime.Before(time.Now().UTC().Add(time.Duration(minAdvanceHours)*time.Hour)) {
		return Reservation{}, fmt.Errorf("reservation must be submitted at least %d hours in advance", minAdvanceHours)
	}
	if bookingWindowDays > 0 && input.StartTime.After(appToday().AddDate(0, 0, bookingWindowDays+1)) {
		return Reservation{}, fmt.Errorf("reservation must start within %d days", bookingWindowDays)
	}
	if !isWithinServiceHours(input.StartTime, input.EndTime, serviceStartHour, serviceEndHour) {
		return Reservation{}, clientError("reservation must be within service hours")
	}
	if !isAlignedToReservationInterval(input.StartTime, bookingIntervalHours, serviceStartHour) || !isAlignedToReservationInterval(input.EndTime, bookingIntervalHours, serviceStartHour) {
		return Reservation{}, fmt.Errorf("reservation time must use %d hour intervals", maxInt(bookingIntervalHours, 1))
	}

	var userID, userStatus, groupName string
	var emailVerified bool
	var userErr error
	if input.UserID != "" {
		userErr = tx.QueryRow(ctx, `
SELECT id::text, name, status, group_name, email_verified
FROM users
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.UserID, tenant.AllTenants, tenant.TenantID).Scan(&userID, &input.UserName, &userStatus, &groupName, &emailVerified)
	} else {
		userErr = tx.QueryRow(ctx, `
SELECT id::text, status, group_name, email_verified
FROM users
WHERE name = $1 AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY created_at DESC
LIMIT 1
`, input.UserName, tenant.AllTenants, tenant.TenantID).Scan(&userID, &userStatus, &groupName, &emailVerified)
	}
	if userErr != nil && !errors.Is(userErr, pgx.ErrNoRows) {
		return Reservation{}, userErr
	}
	if errors.Is(userErr, pgx.ErrNoRows) {
		return Reservation{}, clientError("user must be registered before booking")
	}
	if userStatus != "" && userStatus != "active" {
		return Reservation{}, clientError("user is not active")
	}
	if !emailVerified {
		return Reservation{}, clientError("email must be verified before booking")
	}

	var maintenanceConflict int
	if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM maintenance_orders
WHERE instrument_id = $1
  AND status IN ('reported', 'assigned', 'in_progress')
  AND period && tstzrange($2, $3, '[)')
  AND ($4::boolean OR tenant_id = $5::uuid)
`, input.InstrumentID, input.StartTime, input.EndTime, tenant.AllTenants, tenant.TenantID).Scan(&maintenanceConflict); err != nil {
		return Reservation{}, err
	}
	if maintenanceConflict > 0 {
		return Reservation{}, clientError("reservation conflicts with maintenance window")
	}

	existing, err := r.findDuplicateReservation(ctx, tx, input)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Reservation{}, err
	}

	hours := input.EndTime.Sub(input.StartTime).Hours()
	fee := hours * hourlyRate
	var reservation Reservation
	err = tx.QueryRow(ctx, `
INSERT INTO reservations (tenant_id, user_id, instrument_id, user_name, group_name, purpose, period, status, fee, idempotency_key)
VALUES ($10, $1, $2, $3, $4, $5, tstzrange($6, $7, '[)'), 'pending', $8, $9)
RETURNING id::text, tenant_id::text, COALESCE(user_id::text, ''), COALESCE(instrument_id::text, ''), user_name, group_name, purpose, lower(period), upper(period), status, fee::float8
`, userID, input.InstrumentID, input.UserName, groupName, input.Purpose, input.StartTime, input.EndTime, fee, input.IdempotencyKey, tenant.TenantID).Scan(
		&reservation.ID,
		&reservation.TenantID,
		&reservation.UserID,
		&reservation.InstrumentID,
		&reservation.UserName,
		&reservation.GroupName,
		&reservation.Purpose,
		&reservation.StartTime,
		&reservation.EndTime,
		&reservation.Status,
		&reservation.Fee,
	)
	if err != nil {
		return Reservation{}, err
	}
	reservation.InstrumentName = instrumentName

	notification, err := r.createNotificationTx(ctx, tx, tenant.TenantID, userID, groupName, "", "group", "预约状态更新", fmt.Sprintf("%s 提交了 %s 的预约申请，当前状态：待审批。", input.UserName, instrumentName), "info")
	if err != nil {
		return Reservation{}, err
	}
	if notifications != nil {
		*notifications = append(*notifications, notification)
	}
	return reservation, nil
}

func (r *Repository) ApproveReservation(ctx context.Context, id string, approved bool, actor string, comment string) (Reservation, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	comment = strings.TrimSpace(comment)
	if actor == "" {
		actor = "system"
	}
	status := "rejected"
	action := "reject"
	if approved {
		status = "approved"
		action = "approve"
	}
	if comment == "" {
		comment = status
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Reservation{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var reservation Reservation
	err = tx.QueryRow(ctx, `
UPDATE reservations r
SET status = $2, decided_at = now()
WHERE r.id = $1 AND r.status = 'pending'
  AND ($3::boolean OR r.tenant_id = $4::uuid)
RETURNING r.id::text, r.tenant_id::text, COALESCE(r.user_id::text, ''), COALESCE(r.instrument_id::text, ''),
    COALESCE((SELECT name FROM instruments WHERE id = r.instrument_id), '已删除仪器'),
    r.user_name, r.group_name, r.purpose, lower(r.period), upper(r.period), r.status, r.fee::float8
`, id, status, tenant.AllTenants, tenant.TenantID).Scan(
		&reservation.ID,
		&reservation.TenantID,
		&reservation.UserID,
		&reservation.InstrumentID,
		&reservation.InstrumentName,
		&reservation.UserName,
		&reservation.GroupName,
		&reservation.Purpose,
		&reservation.StartTime,
		&reservation.EndTime,
		&reservation.Status,
		&reservation.Fee,
	)
	if err != nil {
		return Reservation{}, err
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO approval_actions (tenant_id, reservation_id, actor, action, comment)
VALUES ($1, $2, $3, $4, $5)
`, reservation.TenantID, reservation.ID, actor, action, comment); err != nil {
		return Reservation{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, err
	}
	if reservation.UserID != "" {
		if _, err := r.createNotification(ctx, reservation.TenantID, reservation.UserID, reservation.GroupName, "", "personal", "预约状态更新", fmt.Sprintf("%s 的 %s 预约状态已更新为%s。", reservation.UserName, reservation.InstrumentName, reservationStatusLabel(reservation.Status)), notificationLevelForStatus(reservation.Status)); err != nil {
			return Reservation{}, err
		}
	}
	r.invalidateDashboard(ctx)
	r.enqueueEvent(ctx, "reservation.reviewed", map[string]any{
		"reservationId": reservation.ID,
		"instrumentId":  reservation.InstrumentID,
		"userName":      reservation.UserName,
		"status":        reservation.Status,
	})
	if approved {
		r.emitAccessControlGrant(ctx, reservation)
	}
	r.audit(ctx, actor, "reservation."+action, "reservation", reservation.ID, "pending", reservation.Status)
	return reservation, nil
}

func (r *Repository) CompleteReservation(ctx context.Context, id string) (Reservation, error) {
	tenant := TenantFromContext(ctx)
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Reservation{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var reservation Reservation
	err = tx.QueryRow(ctx, `
UPDATE reservations r
SET status = 'completed', checked_out_at = now()
WHERE r.id = $1 AND r.status = 'in_use'
  AND ($2::boolean OR r.tenant_id = $3::uuid)
RETURNING r.id::text, r.tenant_id::text, COALESCE(r.user_id::text, ''), COALESCE(r.instrument_id::text, ''),
    COALESCE((SELECT name FROM instruments WHERE id = r.instrument_id), '已删除仪器'),
    r.user_name, r.group_name, r.purpose, lower(r.period), upper(r.period), r.status, r.fee::float8
`, id, tenant.AllTenants, tenant.TenantID).Scan(
		&reservation.ID,
		&reservation.TenantID,
		&reservation.UserID,
		&reservation.InstrumentID,
		&reservation.InstrumentName,
		&reservation.UserName,
		&reservation.GroupName,
		&reservation.Purpose,
		&reservation.StartTime,
		&reservation.EndTime,
		&reservation.Status,
		&reservation.Fee,
	)
	if err != nil {
		return Reservation{}, err
	}
	if reservation.UserID == "" {
		return Reservation{}, clientError("reservation user is missing")
	}

	_, err = tx.Exec(ctx, `
INSERT INTO ledger_entries (tenant_id, reservation_id, user_id, user_name, group_name, description, amount)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`, reservation.TenantID, reservation.ID, reservation.UserID, reservation.UserName, reservation.GroupName, fmt.Sprintf("%s 使用完成: %s", reservation.UserName, reservation.InstrumentName), reservation.Fee)
	if err != nil {
		return Reservation{}, err
	}

	_, err = tx.Exec(ctx, `
INSERT INTO financial_accounts (tenant_id, user_id, user_name, group_name, balance)
VALUES ($1, $2, $3, $4, $5 * -1)
ON CONFLICT (tenant_id, user_id) WHERE user_id IS NOT NULL
DO UPDATE SET user_name = EXCLUDED.user_name,
              group_name = EXCLUDED.group_name,
              balance = financial_accounts.balance - $5,
              updated_at = now()
`, reservation.TenantID, reservation.UserID, reservation.UserName, reservation.GroupName, reservation.Fee)
	if err != nil {
		return Reservation{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, err
	}
	if _, err := r.createNotification(ctx, reservation.TenantID, reservation.UserID, reservation.GroupName, "", "personal", "费用已入账", fmt.Sprintf("%s 的预约完成，已生成 %.2f 元费用流水。", reservation.InstrumentName, reservation.Fee), "success"); err != nil {
		return Reservation{}, err
	}
	r.invalidateDashboard(ctx)
	r.enqueueEvent(ctx, "reservation.completed", map[string]any{
		"reservationId": reservation.ID,
		"instrumentId":  reservation.InstrumentID,
		"userName":      reservation.UserName,
		"fee":           reservation.Fee,
	})
	r.emitAccessControlRevoke(ctx, reservation, "completed")
	r.audit(ctx, reservation.UserName, "reservation.checkout", "reservation", reservation.ID, "in_use", reservation.Status)
	return reservation, nil
}

func (r *Repository) CheckInReservation(ctx context.Context, id string) (Reservation, error) {
	tenant := TenantFromContext(ctx)
	var reservation Reservation
	err := r.db.QueryRow(ctx, `
UPDATE reservations r
SET status = 'in_use', checked_in_at = now()
WHERE r.id = $1 AND r.status = 'approved'
  AND ($2::boolean OR r.tenant_id = $3::uuid)
  AND now() >= lower(r.period) - (
      SELECT checkin_window_minutes * interval '1 minute'
      FROM instruments
      WHERE id = r.instrument_id
  )
  AND now() <= upper(r.period)
RETURNING r.id::text, r.tenant_id::text, COALESCE(r.user_id::text, ''), COALESCE(r.instrument_id::text, ''),
    COALESCE((SELECT name FROM instruments WHERE id = r.instrument_id), '已删除仪器'),
    r.user_name, r.group_name, r.purpose, lower(r.period), upper(r.period), r.status, r.fee::float8
`, id, tenant.AllTenants, tenant.TenantID).Scan(
		&reservation.ID,
		&reservation.TenantID,
		&reservation.UserID,
		&reservation.InstrumentID,
		&reservation.InstrumentName,
		&reservation.UserName,
		&reservation.GroupName,
		&reservation.Purpose,
		&reservation.StartTime,
		&reservation.EndTime,
		&reservation.Status,
		&reservation.Fee,
	)
	if err != nil {
		return Reservation{}, err
	}
	_, err = r.createNotification(ctx, reservation.TenantID, reservation.UserID, reservation.GroupName, "", "personal", "预约已签到", fmt.Sprintf("%s 已开始使用 %s。", reservation.UserName, reservation.InstrumentName), "info")
	if err != nil {
		return Reservation{}, err
	}
	r.invalidateDashboard(ctx)
	r.audit(ctx, reservation.UserName, "reservation.checkin", "reservation", reservation.ID, "approved", reservation.Status)
	return reservation, nil
}

func (r *Repository) CancelReservation(ctx context.Context, id string, reason string, bypassCutoff bool) (Reservation, error) {
	tenant := TenantFromContext(ctx)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "用户取消"
	}
	var reservation Reservation
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Reservation{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var notification Notification
	err = tx.QueryRow(ctx, `
UPDATE reservations r
SET status = 'cancelled', cancel_reason = $2, cancelled_at = now()
WHERE r.id = $1 AND r.status IN ('pending', 'approved')
  AND ($4::boolean OR r.tenant_id = $5::uuid)
  AND ($3 OR now() <= lower(r.period) - (
      SELECT cancel_cutoff_hours * interval '1 hour'
      FROM instruments
      WHERE id = r.instrument_id
  ))
RETURNING r.id::text, r.tenant_id::text, COALESCE(r.user_id::text, ''), COALESCE(r.instrument_id::text, ''),
    COALESCE((SELECT name FROM instruments WHERE id = r.instrument_id), '已删除仪器'),
    r.user_name, r.group_name, r.purpose, lower(r.period), upper(r.period), r.status, r.fee::float8
`, id, reason, bypassCutoff, tenant.AllTenants, tenant.TenantID).Scan(
		&reservation.ID,
		&reservation.TenantID,
		&reservation.UserID,
		&reservation.InstrumentID,
		&reservation.InstrumentName,
		&reservation.UserName,
		&reservation.GroupName,
		&reservation.Purpose,
		&reservation.StartTime,
		&reservation.EndTime,
		&reservation.Status,
		&reservation.Fee,
	)
	if err != nil {
		return Reservation{}, err
	}
	if reservation.UserID != "" {
		notification, err = r.createNotificationTx(ctx, tx, reservation.TenantID, reservation.UserID, reservation.GroupName, "", "personal", "预约状态更新", fmt.Sprintf("%s 的 %s 预约状态已更新为%s，原因：%s。", reservation.UserName, reservation.InstrumentName, reservationStatusLabel(reservation.Status), reason), notificationLevelForStatus(reservation.Status))
		if err != nil {
			return Reservation{}, err
		}
	}
	if err := r.auditTx(ctx, tx, reservation.TenantID, reservation.UserName, "reservation.cancel", "reservation", reservation.ID, "", reason); err != nil {
		return Reservation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, err
	}
	r.enqueueDingTalkNotifications(notification)
	r.invalidateDashboard(ctx)
	r.emitAccessControlRevoke(ctx, reservation, reason)
	return reservation, nil
}

func (r *Repository) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := r.db.Exec(ctx, `
DELETE FROM sessions
WHERE expires_at < now()
   OR (revoked_at IS NOT NULL AND revoked_at < now() - interval '30 days')
`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *Repository) ExpireStaleReservations(ctx context.Context, maxAge time.Duration) (int, error) {
	if maxAge <= 0 {
		maxAge = 24 * time.Hour
	}
	const batchSize = 200
	type expiredReservation struct {
		id             string
		userID         string
		userName       string
		groupName      string
		tenantID       string
		instrumentName string
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	rows, err := tx.Query(ctx, `
WITH stale AS (
  SELECT id
  FROM reservations
  WHERE status = 'pending'
    AND created_at < now() - ($1 * interval '1 second')
  ORDER BY created_at
  LIMIT $2
  FOR UPDATE SKIP LOCKED
)
UPDATE reservations r
SET status = 'cancelled', cancel_reason = '审批超时自动取消', cancelled_at = now()
FROM stale
WHERE r.id = stale.id
RETURNING r.id::text, COALESCE(r.user_id::text, ''), r.user_name, r.group_name,
          r.tenant_id::text,
          COALESCE((SELECT name FROM instruments WHERE id = r.instrument_id), '已删除仪器')
`, int64(maxAge.Seconds()), batchSize)
	if err != nil {
		return 0, err
	}

	expired := make([]expiredReservation, 0, batchSize)
	for rows.Next() {
		var item expiredReservation
		if err := rows.Scan(&item.id, &item.userID, &item.userName, &item.groupName, &item.tenantID, &item.instrumentName); err != nil {
			rows.Close()
			return len(expired), err
		}
		expired = append(expired, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return len(expired), err
	}
	rows.Close()
	notifications := make([]Notification, 0, len(expired))
	for _, item := range expired {
		if item.userID != "" {
			notification, err := r.createNotificationTx(ctx, tx, item.tenantID, item.userID, item.groupName, "", "personal", "预约状态更新", fmt.Sprintf("%s 的 %s 预约状态已更新为已取消，原因：审批超时自动取消。", item.userName, item.instrumentName), "warning")
			if err != nil {
				return len(expired), err
			}
			notifications = append(notifications, notification)
		}
		if err := r.auditTx(ctx, tx, item.tenantID, "system", "reservation.auto_cancel", "reservation", item.id, "pending", "cancelled"); err != nil {
			return len(expired), err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return len(expired), err
	}
	count := len(expired)
	if count > 0 {
		r.enqueueDingTalkNotifications(notifications...)
		r.invalidateDashboard(ctx)
		r.enqueueEvent(ctx, "reservation.auto_cancelled", map[string]any{"count": count})
	}
	return count, nil
}

func (r *Repository) InstrumentSlots(ctx context.Context, id string, start time.Time, days int) ([]Slot, error) {
	tenant := TenantFromContext(ctx)
	if days <= 0 || days > 30 {
		days = 30
	}

	instrument, err := r.Instrument(ctx, id)
	if err != nil {
		return nil, err
	}
	if instrument.BookingWindowDays > 0 && days > instrument.BookingWindowDays {
		days = instrument.BookingWindowDays
	}
	serviceStartHour, serviceEndHour := normalizeServiceHours(instrument.ServiceStartHour, instrument.ServiceEndHour)
	localStart := start.In(reservationServiceLocation())
	start = time.Date(localStart.Year(), localStart.Month(), localStart.Day(), serviceStartHour, 0, 0, 0, reservationServiceLocation())
	end := start.AddDate(0, 0, days)
	stepHours := instrument.BookingIntervalHours
	if stepHours <= 0 {
		stepHours = 1
	}
	step := time.Duration(stepHours) * time.Hour
	bookableAfter := time.Now().UTC().Add(time.Duration(maxInt(instrument.MinAdvanceHours, 0)) * time.Hour)

	reservationRows, err := r.db.Query(ctx, `
SELECT lower(period), upper(period), status
FROM reservations
WHERE instrument_id = $1
  AND status IN ('pending', 'approved', 'in_use')
  AND period && tstzrange($2, $3, '[)')
  AND ($4::boolean OR tenant_id = $5::uuid)
`, id, start, end, tenant.AllTenants, instrument.TenantID)
	if err != nil {
		return nil, err
	}
	defer reservationRows.Close()

	occupied := make([]Slot, 0)
	for reservationRows.Next() {
		var slot Slot
		var slotStart, slotEnd time.Time
		if err := reservationRows.Scan(&slotStart, &slotEnd, &slot.Reason); err != nil {
			return nil, err
		}
		slot.StartTime = slotStart.Format(time.RFC3339)
		slot.EndTime = slotEnd.Format(time.RFC3339)
		slot.Status = "occupied"
		occupied = append(occupied, slot)
	}
	if err := reservationRows.Err(); err != nil {
		return nil, err
	}

	maintenanceRows, err := r.db.Query(ctx, `
SELECT lower(period), upper(period), status
FROM maintenance_orders
WHERE instrument_id = $1
  AND status IN ('reported', 'assigned', 'in_progress')
  AND period && tstzrange($2, $3, '[)')
  AND ($4::boolean OR tenant_id = $5::uuid)
`, id, start, end, tenant.AllTenants, instrument.TenantID)
	if err != nil {
		return nil, err
	}
	defer maintenanceRows.Close()
	for maintenanceRows.Next() {
		var slot Slot
		var slotStart, slotEnd time.Time
		if err := maintenanceRows.Scan(&slotStart, &slotEnd, &slot.Reason); err != nil {
			return nil, err
		}
		slot.StartTime = slotStart.Format(time.RFC3339)
		slot.EndTime = slotEnd.Format(time.RFC3339)
		slot.Status = "maintenance"
		occupied = append(occupied, slot)
	}
	if err := maintenanceRows.Err(); err != nil {
		return nil, err
	}

	serviceHours := serviceEndHour - serviceStartHour
	slots := make([]Slot, 0, days*maxInt((serviceHours+stepHours-1)/stepHours, 1))
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		serviceStart := time.Date(day.Year(), day.Month(), day.Day(), serviceStartHour, 0, 0, 0, day.Location())
		serviceEnd := time.Date(day.Year(), day.Month(), day.Day(), serviceEndHour, 0, 0, 0, day.Location())
		for cursor := serviceStart; cursor.Before(serviceEnd); cursor = cursor.Add(step) {
			slotEnd := cursor.Add(step)
			if slotEnd.After(serviceEnd) {
				break
			}
			slot := Slot{
				StartTime: cursor.Format(time.RFC3339),
				EndTime:   slotEnd.Format(time.RFC3339),
				Status:    "available",
				Reason:    "",
			}
			if instrument.Status == "maintenance" || instrument.Status == "disabled" {
				slot.Status = instrument.Status
				slot.Reason = "instrument_" + instrument.Status
			}
			if slot.Status == "available" && cursor.Before(bookableAfter) {
				slot.Status = "unavailable"
				slot.Reason = "min_advance"
			}
			for _, block := range occupied {
				blockStart, _ := time.Parse(time.RFC3339, block.StartTime)
				blockEnd, _ := time.Parse(time.RFC3339, block.EndTime)
				if cursor.Before(blockEnd) && slotEnd.After(blockStart) {
					slot.Status = block.Status
					slot.Reason = block.Reason
					break
				}
			}
			slots = append(slots, slot)
		}
	}
	return slots, nil
}

func materialColumn(alias string, column string) string {
	if alias == "" {
		return column
	}
	return alias + "." + column
}

func materialDamageQuantitySQL(alias string) string {
	return fmt.Sprintf(`(
    SELECT COALESCE(sum(mdr.quantity), 0)::int
    FROM material_damage_reports mdr
    WHERE mdr.material_id = %s
      AND mdr.status = 'processed'
)`, materialColumn(alias, "id"))
}

func materialStockSQL(alias string) string {
	return fmt.Sprintf(`CASE
    WHEN EXISTS (
        SELECT 1
        FROM material_units mu
        WHERE mu.material_id = %[1]s
    ) THEN (
        SELECT count(*)::int
        FROM material_units mu
        WHERE mu.material_id = %[1]s
          AND mu.status IN ('available', 'reserved')
    )
    ELSE %[2]s
END`, materialColumn(alias, "id"), materialColumn(alias, "stock"))
}

func materialStatusSQL(alias string) string {
	status := materialColumn(alias, "status")
	expiresAt := materialExpiresAtSQL(alias)
	openedAt := materialColumn(alias, "opened_at")
	openExpireDays := materialColumn(alias, "open_expire_days")
	freezeThawCount := materialColumn(alias, "freeze_thaw_count")
	freezeThawLimit := materialColumn(alias, "freeze_thaw_limit")
	nearExpiryDays := materialColumn(alias, "near_expiry_days")
	stock := materialStockSQL(alias)
	warningLine := materialColumn(alias, "warning_line")
	damageQuantity := materialDamageQuantitySQL(alias)
	appDate := appDateSQL()
	return fmt.Sprintf(`CASE
    WHEN %[1]s = 'disabled' THEN 'disabled'
    WHEN %[2]s IS NOT NULL AND %[2]s < %[11]s THEN 'expired'
    WHEN %[3]s IS NOT NULL AND %[4]s > 0 AND (%[3]s + %[4]s) < %[11]s THEN 'open_expired'
    WHEN %[5]s > 0 AND %[6]s >= %[5]s THEN 'freeze_thaw_exceeded'
    WHEN %[1]s = 'damaged' OR %[7]s > 0 THEN 'damaged'
    WHEN %[2]s IS NOT NULL AND %[2]s <= %[11]s + %[8]s THEN 'near_expiry'
    WHEN %[9]s <= %[10]s THEN 'low'
    ELSE 'normal'
END`, status, expiresAt, openedAt, openExpireDays, freezeThawLimit, freezeThawCount, damageQuantity, nearExpiryDays, stock, warningLine, appDate)
}

func materialExpiresAtSQL(alias string) string {
	return fmt.Sprintf(`CASE
    WHEN EXISTS (
        SELECT 1
        FROM material_units mu
        WHERE mu.material_id = %[1]s
          AND mu.status IN ('available', 'reserved')
    ) THEN (
        SELECT min(mu.expires_at)
        FROM material_units mu
        WHERE mu.material_id = %[1]s
          AND mu.status IN ('available', 'reserved')
    )
    ELSE %[2]s
END`, materialColumn(alias, "id"), materialColumn(alias, "expires_at"))
}

func materialSelectColumns(alias string) string {
	parentName := "COALESCE(parent_material.name, '')"
	if alias == "materials" {
		parentName = `COALESCE((SELECT parent.name FROM materials parent WHERE parent.id = materials.parent_material_id), '')`
	}
	return fmt.Sprintf(`%s::text, %s, %s, %s, %s, %s, %s, %s::float8, %s, %s, %s, %s, %s,
       %s, %s, %s, %s, COALESCE(%s::text, ''), %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
       COALESCE(%s::text, ''), COALESCE(%s::text, ''), %s,
       CASE WHEN %s IS NOT NULL AND %s > 0 THEN (%s + %s)::text ELSE '' END,
       %s, %s, %s, %s, %s, %s`,
		materialColumn(alias, "id"),
		materialColumn(alias, "name"),
		materialColumn(alias, "product_type"),
		materialColumn(alias, "category"),
		materialColumn(alias, "subcategory"),
		materialColumn(alias, "spec"),
		materialColumn(alias, "unit"),
		materialColumn(alias, "unit_price"),
		materialStockSQL(alias),
		materialColumn(alias, "warning_line"),
		materialColumn(alias, "supplier"),
		materialColumn(alias, "manufacturer"),
		materialColumn(alias, "batch_no"),
		materialColumn(alias, "catalog_no"),
		materialColumn(alias, "cas_no"),
		materialColumn(alias, "grade"),
		materialColumn(alias, "concentration"),
		materialColumn(alias, "parent_material_id"),
		parentName,
		materialColumn(alias, "dilution_factor"),
		materialColumn(alias, "preparation_method"),
		materialColumn(alias, "storage_condition"),
		materialColumn(alias, "storage_room"),
		materialColumn(alias, "storage_cabinet"),
		materialColumn(alias, "storage_layer"),
		materialColumn(alias, "storage_slot"),
		materialColumn(alias, "tender_contract"),
		materialColumn(alias, "contract_no"),
		materialColumn(alias, "remark"),
		materialColumn(alias, "certificate_url"),
		materialColumn(alias, "standard_certificate_url"),
		materialColumn(alias, "attachment_url"),
		materialColumn(alias, "qr_code"),
		materialColumn(alias, "expires_at"),
		materialColumn(alias, "opened_at"),
		materialColumn(alias, "open_expire_days"),
		materialColumn(alias, "opened_at"),
		materialColumn(alias, "open_expire_days"),
		materialColumn(alias, "opened_at"),
		materialColumn(alias, "open_expire_days"),
		materialColumn(alias, "freeze_thaw_count"),
		materialColumn(alias, "freeze_thaw_limit"),
		materialColumn(alias, "approval_required"),
		materialColumn(alias, "near_expiry_days"),
		materialDamageQuantitySQL(alias),
		materialStatusSQL(alias),
	)
}

func scanMaterial(row scanner) (Material, error) {
	var item Material
	err := row.Scan(
		&item.ID,
		&item.Name,
		&item.ProductType,
		&item.Category,
		&item.Subcategory,
		&item.Spec,
		&item.Unit,
		&item.UnitPrice,
		&item.Stock,
		&item.WarningLine,
		&item.Supplier,
		&item.Manufacturer,
		&item.BatchNo,
		&item.CatalogNo,
		&item.CASNo,
		&item.Grade,
		&item.Concentration,
		&item.ParentMaterialID,
		&item.ParentMaterialName,
		&item.DilutionFactor,
		&item.PreparationMethod,
		&item.StorageCondition,
		&item.StorageRoom,
		&item.StorageCabinet,
		&item.StorageLayer,
		&item.StorageSlot,
		&item.TenderContract,
		&item.ContractNo,
		&item.Remark,
		&item.CertificateURL,
		&item.StandardCertificateURL,
		&item.AttachmentURL,
		&item.QRCode,
		&item.ExpiresAt,
		&item.OpenedAt,
		&item.OpenExpireDays,
		&item.OpenExpiresAt,
		&item.FreezeThawCount,
		&item.FreezeThawLimit,
		&item.ApprovalRequired,
		&item.NearExpiryDays,
		&item.DamagedQuantity,
		&item.Status,
	)
	return item, err
}

func scanMaterialBatch(row scanner) (MaterialBatch, error) {
	var item MaterialBatch
	err := row.Scan(
		&item.ID,
		&item.MaterialID,
		&item.BatchNo,
		&item.Quantity,
		&item.ExpiresAt,
		&item.Location,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func scanMaterialUnit(row scanner) (MaterialUnit, error) {
	var item MaterialUnit
	err := row.Scan(
		&item.ID,
		&item.MaterialID,
		&item.BatchID,
		&item.BatchNo,
		&item.UnitCode,
		&item.ExpiresAt,
		&item.Location,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func (r *Repository) attachMaterialBatches(ctx context.Context, items []Material) ([]Material, error) {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item.Stock > 0 || item.ProductType == "standard" || item.ProductType == "reagent" || item.ProductType == "consumable" {
			ids = append(ids, item.ID)
		}
	}
	if len(ids) == 0 {
		return items, nil
	}
	rows, err := r.db.Query(ctx, `
SELECT id::text, material_id::text, batch_no, quantity, COALESCE(expires_at::text, ''), location, status, created_at, updated_at
FROM material_batches
WHERE material_id::text = ANY($1::text[])
ORDER BY status, expires_at NULLS LAST, batch_no
`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	batchesByMaterial := make(map[string][]MaterialBatch)
	for rows.Next() {
		batch, err := scanMaterialBatch(rows)
		if err != nil {
			return nil, err
		}
		batchesByMaterial[batch.MaterialID] = append(batchesByMaterial[batch.MaterialID], batch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	unitRows, err := r.db.Query(ctx, `
SELECT mu.id::text, mu.material_id::text, COALESCE(mu.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       mu.unit_code, COALESCE(mu.expires_at::text, ''), mu.location, mu.status, mu.created_at, mu.updated_at
FROM material_units mu
LEFT JOIN material_batches mb ON mb.id = mu.batch_id
WHERE mu.material_id::text = ANY($1::text[])
ORDER BY mu.status, mu.expires_at NULLS LAST, mu.unit_code
`, ids)
	if err != nil {
		return nil, err
	}
	defer unitRows.Close()
	unitsByMaterial := make(map[string][]MaterialUnit)
	unitsByBatch := make(map[string][]MaterialUnit)
	for unitRows.Next() {
		unit, err := scanMaterialUnit(unitRows)
		if err != nil {
			return nil, err
		}
		unitsByMaterial[unit.MaterialID] = append(unitsByMaterial[unit.MaterialID], unit)
		if unit.BatchID != "" {
			unitsByBatch[unit.BatchID] = append(unitsByBatch[unit.BatchID], unit)
		}
	}
	if err := unitRows.Err(); err != nil {
		return nil, err
	}
	for index := range items {
		items[index].Units = unitsByMaterial[items[index].ID]
		batches := batchesByMaterial[items[index].ID]
		for batchIndex := range batches {
			batches[batchIndex].Units = unitsByBatch[batches[batchIndex].ID]
			batches[batchIndex].Quantity = countAvailableMaterialUnits(batches[batchIndex].Units)
		}
		items[index].Batches = batches
	}
	return items, nil
}

func countAvailableMaterialUnits(units []MaterialUnit) int {
	count := 0
	for _, unit := range units {
		if unit.Status == "available" {
			count++
		}
	}
	return count
}

func (r *Repository) attachMaterialBatch(ctx context.Context, item Material) (Material, error) {
	items, err := r.attachMaterialBatches(ctx, []Material{item})
	if err != nil {
		return Material{}, err
	}
	return items[0], nil
}

func (r *Repository) Materials(ctx context.Context) ([]Material, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
SELECT %s
FROM materials m
LEFT JOIN materials parent_material ON parent_material.id = m.parent_material_id
WHERE ($1::boolean OR m.tenant_id = $2::uuid)
ORDER BY m.category, m.name
`, materialSelectColumns("m")), tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Material, 0)
	for rows.Next() {
		item, err := scanMaterial(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return r.attachMaterialBatches(ctx, items)
}

func (r *Repository) Material(ctx context.Context, id string) (Material, error) {
	tenant := TenantFromContext(ctx)
	item, err := scanMaterial(r.db.QueryRow(ctx, fmt.Sprintf(`
SELECT %s
FROM materials m
LEFT JOIN materials parent_material ON parent_material.id = m.parent_material_id
WHERE m.id = $1
  AND ($2::boolean OR m.tenant_id = $3::uuid)
`, materialSelectColumns("m")), id, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return Material{}, err
	}
	return r.attachMaterialBatch(ctx, item)
}

func (r *Repository) MaterialByQRCode(ctx context.Context, code string) (Material, error) {
	tenant := TenantFromContext(ctx)
	code = strings.TrimSpace(code)
	if code == "" {
		return Material{}, clientError("missing material qr code")
	}
	item, err := scanMaterial(r.db.QueryRow(ctx, fmt.Sprintf(`
SELECT %s
FROM materials m
LEFT JOIN materials parent_material ON parent_material.id = m.parent_material_id
WHERE (m.qr_code = $1 OR m.id::text = $1)
  AND ($2::boolean OR m.tenant_id = $3::uuid)
`, materialSelectColumns("m")), code, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return Material{}, err
	}
	return r.attachMaterialBatch(ctx, item)
}

func (r *Repository) InventoryLedger(ctx context.Context) ([]InventoryLedgerEntry, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT il.id::text, il.material_id::text, m.name, COALESCE(il.request_id::text, ''),
       COALESCE(il.purchase_id::text, ''), COALESCE(il.damage_id::text, ''), il.change_qty, il.reason, il.created_at
FROM inventory_ledger il
JOIN materials m ON m.id = il.material_id
WHERE ($1::boolean OR il.tenant_id = $2::uuid)
ORDER BY il.created_at DESC
LIMIT 200
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]InventoryLedgerEntry, 0)
	for rows.Next() {
		var item InventoryLedgerEntry
		if err := rows.Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.RequestID, &item.PurchaseID, &item.DamageID, &item.ChangeQty, &item.Reason, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) MaterialCategories(ctx context.Context) ([]MaterialCategory, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, name, parent_name, display_order, status, created_at, updated_at
FROM material_categories
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY display_order, parent_name, name
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialCategory, 0)
	for rows.Next() {
		var item MaterialCategory
		if err := rows.Scan(&item.ID, &item.Name, &item.ParentName, &item.DisplayOrder, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveMaterialCategory(ctx context.Context, id string, input MaterialCategoryInput) (MaterialCategory, error) {
	tenant := TenantFromContext(ctx)
	input.Name = strings.TrimSpace(input.Name)
	input.ParentName = strings.TrimSpace(input.ParentName)
	input.Status = strings.TrimSpace(input.Status)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Name == "" || (input.Status != "active" && input.Status != "disabled") {
		return MaterialCategory{}, clientError("invalid material category input")
	}
	if id == "" {
		var item MaterialCategory
		err := r.db.QueryRow(ctx, `
INSERT INTO material_categories (tenant_id, name, parent_name, display_order, status)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, name) DO UPDATE
SET parent_name = EXCLUDED.parent_name,
    display_order = EXCLUDED.display_order,
    status = EXCLUDED.status,
    updated_at = now()
RETURNING id::text, name, parent_name, display_order, status, created_at, updated_at
`, tenant.TenantID, input.Name, input.ParentName, input.DisplayOrder, input.Status).Scan(&item.ID, &item.Name, &item.ParentName, &item.DisplayOrder, &item.Status, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return MaterialCategory{}, err
		}
		r.audit(ctx, input.Actor, "material_category.save", "material_category", item.ID, "", item.Name)
		return item, nil
	}

	var item MaterialCategory
	err := r.db.QueryRow(ctx, `
UPDATE material_categories
SET name = $2, parent_name = $3, display_order = $4, status = $5, updated_at = now()
WHERE id = $1 AND ($6::boolean OR tenant_id = $7::uuid)
RETURNING id::text, name, parent_name, display_order, status, created_at, updated_at
`, id, input.Name, input.ParentName, input.DisplayOrder, input.Status, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &item.Name, &item.ParentName, &item.DisplayOrder, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return MaterialCategory{}, err
	}
	r.audit(ctx, input.Actor, "material_category.update", "material_category", item.ID, "", item.Status)
	return item, nil
}

func (r *Repository) DeleteMaterialCategory(ctx context.Context, id string, actor string) (MaterialCategory, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var item MaterialCategory
	err := r.db.QueryRow(ctx, `
UPDATE material_categories
SET status = 'disabled', updated_at = now()
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
RETURNING id::text, name, parent_name, display_order, status, created_at, updated_at
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &item.Name, &item.ParentName, &item.DisplayOrder, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return MaterialCategory{}, err
	}
	r.audit(ctx, actor, "material_category.disable", "material_category", item.ID, "", item.Status)
	return item, nil
}

func (r *Repository) PurchasableMaterials(ctx context.Context) ([]PurchasableMaterial, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT pm.id::text, pm.id_no, pm.sequence_no, COALESCE(pm.procurement_project_id::text, ''),
       COALESCE(pp.name, pm.procurement_project), COALESCE(pp.expires_at::text, ''),
       COALESCE(pp.status, 'active'),
       pm.project_name, pm.brand, pm.spec, pm.unit, pm.purchase_price::float8,
       pm.remark, pm.technical_requirement, pm.min_spec, pm.status, pm.created_at, pm.updated_at
FROM purchasable_materials pm
LEFT JOIN procurement_projects pp ON pp.id = pm.procurement_project_id
WHERE pm.status = 'active'
  AND ($1::boolean OR pm.tenant_id = $2::uuid)
ORDER BY pm.project_name, pm.sequence_no, pm.id_no
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PurchasableMaterial, 0)
	for rows.Next() {
		item, err := scanPurchasableMaterial(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ProcurementProjects(ctx context.Context) ([]ProcurementProject, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, name, COALESCE(expires_at::text, ''), status, created_at, updated_at
FROM procurement_projects
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY name
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ProcurementProject, 0)
	for rows.Next() {
		item, err := scanProcurementProject(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveProcurementProject(ctx context.Context, id string, input ProcurementProjectInput) (ProcurementProject, error) {
	tenant := TenantFromContext(ctx)
	input = normalizeProcurementProject(input)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Name == "" {
		return ProcurementProject{}, clientError("material procurement project name is required")
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Status != "active" && input.Status != "disabled" {
		return ProcurementProject{}, clientError("material procurement project status is invalid")
	}
	var item ProcurementProject
	var err error
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO procurement_projects (tenant_id, name, expires_at, status)
VALUES ($1, $2, NULLIF($3, '')::date, $4)
ON CONFLICT (tenant_id, name) DO UPDATE
SET expires_at = EXCLUDED.expires_at,
    status = EXCLUDED.status,
    updated_at = now()
RETURNING id::text, name, COALESCE(expires_at::text, ''), status, created_at, updated_at
`, tenant.TenantID, input.Name, input.ExpiresAt, input.Status).Scan(&item.ID, &item.Name, &item.ExpiresAt, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	} else {
		err = r.db.QueryRow(ctx, `
UPDATE procurement_projects
SET name = $2, expires_at = NULLIF($3, '')::date, status = $4, updated_at = now()
WHERE id = $1
  AND ($5::boolean OR tenant_id = $6::uuid)
RETURNING id::text, name, COALESCE(expires_at::text, ''), status, created_at, updated_at
`, id, input.Name, input.ExpiresAt, input.Status, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &item.Name, &item.ExpiresAt, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	}
	if err != nil {
		return ProcurementProject{}, err
	}
	r.audit(ctx, input.Actor, "procurement_project.save", "procurement_project", item.ID, "", item.Name)
	return item, nil
}

func (r *Repository) DeleteProcurementProject(ctx context.Context, id string, actor string) (ProcurementProject, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var item ProcurementProject
	err := r.db.QueryRow(ctx, `
UPDATE procurement_projects
SET status = 'disabled', updated_at = now()
WHERE id = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
RETURNING id::text, name, COALESCE(expires_at::text, ''), status, created_at, updated_at
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &item.Name, &item.ExpiresAt, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return ProcurementProject{}, err
	}
	r.audit(ctx, actor, "procurement_project.delete", "procurement_project", item.ID, item.Name, item.Status)
	return item, nil
}

func (r *Repository) SavePurchasableMaterial(ctx context.Context, id string, input PurchasableMaterialInput) (PurchasableMaterial, error) {
	tenant := TenantFromContext(ctx)
	input = normalizePurchasableMaterial(input)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.IDNo == "" || input.SequenceNo == "" || input.ProjectName == "" || input.Brand == "" || input.Spec == "" || input.Unit == "" || input.PurchasePrice < 0 {
		return PurchasableMaterial{}, clientError("invalid purchasable material input")
	}
	projectID, projectName, err := r.ensureProcurementProject(ctx, nil, input.ProcurementProjectID, input.ProcurementProject)
	if err != nil {
		return PurchasableMaterial{}, err
	}
	input.ProcurementProject = projectName
	if id == "" {
		var item PurchasableMaterial
		err := r.db.QueryRow(ctx, `
INSERT INTO purchasable_materials (
    tenant_id, id_no, sequence_no, procurement_project_id, procurement_project, project_name, brand, spec, unit, purchase_price,
    remark, technical_requirement, min_spec, status
)
VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'active')
ON CONFLICT (tenant_id, id_no) DO UPDATE
SET sequence_no = EXCLUDED.sequence_no,
    procurement_project_id = EXCLUDED.procurement_project_id,
    procurement_project = EXCLUDED.procurement_project,
    project_name = EXCLUDED.project_name,
    brand = EXCLUDED.brand,
    spec = EXCLUDED.spec,
    unit = EXCLUDED.unit,
    purchase_price = EXCLUDED.purchase_price,
    remark = EXCLUDED.remark,
    technical_requirement = EXCLUDED.technical_requirement,
    min_spec = EXCLUDED.min_spec,
    status = 'active',
    updated_at = now()
RETURNING id::text, id_no, sequence_no, COALESCE(procurement_project_id::text, ''), procurement_project,
          COALESCE((SELECT expires_at::text FROM procurement_projects WHERE id = purchasable_materials.procurement_project_id), ''),
          COALESCE((SELECT status FROM procurement_projects WHERE id = purchasable_materials.procurement_project_id), 'active'),
          project_name, brand, spec, unit, purchase_price::float8,
          remark, technical_requirement, min_spec, status, created_at, updated_at
`, tenant.TenantID, input.IDNo, input.SequenceNo, projectID, input.ProcurementProject, input.ProjectName, input.Brand, input.Spec, input.Unit, input.PurchasePrice, input.Remark, input.TechnicalRequirement, input.MinSpec).Scan(
			&item.ID, &item.IDNo, &item.SequenceNo, &item.ProcurementProjectID, &item.ProcurementProject, &item.ProcurementExpiresAt, &item.ProcurementProjectStatus, &item.ProjectName, &item.Brand, &item.Spec, &item.Unit, &item.PurchasePrice, &item.Remark, &item.TechnicalRequirement, &item.MinSpec, &item.Status, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return PurchasableMaterial{}, err
		}
		r.audit(ctx, input.Actor, "purchasable_material.save", "purchasable_material", item.ID, "", item.IDNo)
		return item, nil
	}

	var item PurchasableMaterial
	err = r.db.QueryRow(ctx, `
UPDATE purchasable_materials
SET id_no = $2,
    sequence_no = $3,
    procurement_project_id = NULLIF($4, '')::uuid,
    procurement_project = $5,
    project_name = $6,
    brand = $7,
    spec = $8,
    unit = $9,
    purchase_price = $10,
    remark = $11,
    technical_requirement = $12,
    min_spec = $13,
    status = 'active',
    updated_at = now()
WHERE id = $1
  AND ($14::boolean OR tenant_id = $15::uuid)
RETURNING id::text, id_no, sequence_no, COALESCE(procurement_project_id::text, ''), procurement_project,
          COALESCE((SELECT expires_at::text FROM procurement_projects WHERE id = purchasable_materials.procurement_project_id), ''),
          COALESCE((SELECT status FROM procurement_projects WHERE id = purchasable_materials.procurement_project_id), 'active'),
          project_name, brand, spec, unit, purchase_price::float8,
          remark, technical_requirement, min_spec, status, created_at, updated_at
`, id, input.IDNo, input.SequenceNo, projectID, input.ProcurementProject, input.ProjectName, input.Brand, input.Spec, input.Unit, input.PurchasePrice, input.Remark, input.TechnicalRequirement, input.MinSpec, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.IDNo, &item.SequenceNo, &item.ProcurementProjectID, &item.ProcurementProject, &item.ProcurementExpiresAt, &item.ProcurementProjectStatus, &item.ProjectName, &item.Brand, &item.Spec, &item.Unit, &item.PurchasePrice, &item.Remark, &item.TechnicalRequirement, &item.MinSpec, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return PurchasableMaterial{}, err
	}
	r.audit(ctx, input.Actor, "purchasable_material.update", "purchasable_material", item.ID, "", item.IDNo)
	return item, nil
}

func (r *Repository) DeletePurchasableMaterial(ctx context.Context, id string, actor string) (PurchasableMaterial, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var item PurchasableMaterial
	err := r.db.QueryRow(ctx, `
UPDATE purchasable_materials
SET status = 'deleted', updated_at = now()
WHERE id = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
RETURNING id::text, id_no, sequence_no, COALESCE(procurement_project_id::text, ''), procurement_project,
          COALESCE((SELECT expires_at::text FROM procurement_projects WHERE id = purchasable_materials.procurement_project_id), ''),
          COALESCE((SELECT status FROM procurement_projects WHERE id = purchasable_materials.procurement_project_id), 'active'),
          project_name, brand, spec, unit, purchase_price::float8,
          remark, technical_requirement, min_spec, status, created_at, updated_at
`, id, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.IDNo, &item.SequenceNo, &item.ProcurementProjectID, &item.ProcurementProject, &item.ProcurementExpiresAt, &item.ProcurementProjectStatus, &item.ProjectName, &item.Brand, &item.Spec, &item.Unit, &item.PurchasePrice, &item.Remark, &item.TechnicalRequirement, &item.MinSpec, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return PurchasableMaterial{}, err
	}
	r.audit(ctx, actor, "purchasable_material.delete", "purchasable_material", item.ID, item.IDNo, item.Status)
	return item, nil
}

func (r *Repository) ImportPurchasableMaterials(ctx context.Context, input PurchasableMaterialImportInput) (MaterialImportResult, error) {
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	records, err := purchasableMaterialImportRecords(input.Filename, input.Content)
	if err != nil {
		return MaterialImportResult{}, WrapClientError("material purchasable import failed", err)
	}
	result := MaterialImportResult{}
	if len(records) == 0 {
		return result, clientError("material purchasable import failed: 文件内容为空")
	}
	headerIndex := -1
	for i, row := range records {
		if purchasableMaterialLooksLikeHeader(row) {
			headerIndex = i
			break
		}
	}
	if headerIndex < 0 {
		return result, clientError("material purchasable import failed: 未找到表头，请确认包含 ID号、序号、项目名称、品牌、规格、单位、采购价（元）")
	}
	header := materialImportHeaderIndex(records[headerIndex])
	currentProject := ""
	items := make([]PurchasableMaterialInput, 0, len(records)-headerIndex-1)
	for rowIndex, row := range records[headerIndex+1:] {
		line := headerIndex + rowIndex + 2
		if rowBlank(row) {
			continue
		}
		if project := purchasableMaterialProjectHeader(row); project != "" {
			currentProject = project
			continue
		}
		item := purchasableMaterialInputFromRow(header, row, currentProject)
		item.Actor = input.Actor
		if item.IDNo == "" || item.SequenceNo == "" || item.ProjectName == "" || item.Brand == "" || item.Spec == "" || item.Unit == "" {
			if purchasableMaterialLooksLikeNoteRow(row) {
				result.Skipped++
				continue
			}
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行缺少ID号、序号、项目名称、品牌、规格或单位；行内容：%s", line, purchasableMaterialRowPreview(row)))
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		message := "没有导入任何有效物资行"
		if len(result.Errors) > 0 {
			message = strings.Join(limitStrings(result.Errors, 5), "；")
		}
		return result, clientErrorf("material purchasable import failed: %s", message)
	}
	created, updated, err := r.savePurchasableMaterialsBulk(ctx, items)
	if err != nil {
		return result, WrapClientError("material purchasable import failed: 数据库写入失败", err)
	}
	result.Created = created
	result.Updated = updated
	result.Message = fmt.Sprintf("导入完成：有效 %d 行，新增 %d 行，更新 %d 行，跳过 %d 行。", len(items), created, updated, result.Skipped)
	return result, nil
}

func (r *Repository) savePurchasableMaterialsBulk(ctx context.Context, items []PurchasableMaterialInput) (int, int, error) {
	if len(items) == 0 {
		return 0, 0, nil
	}
	tenant := TenantFromContext(ctx)
	idNos := make([]string, 0, len(items))
	for _, item := range items {
		idNos = append(idNos, item.IDNo)
	}
	existingRows, err := r.db.Query(ctx, `
SELECT id_no
FROM purchasable_materials
WHERE tenant_id = $1::uuid AND id_no = ANY($2)
`, tenant.TenantID, idNos)
	if err != nil {
		return 0, 0, err
	}
	existing := make(map[string]struct{}, len(items))
	for existingRows.Next() {
		var idNo string
		if err := existingRows.Scan(&idNo); err != nil {
			existingRows.Close()
			return 0, 0, err
		}
		existing[idNo] = struct{}{}
	}
	existingRows.Close()
	if err := existingRows.Err(); err != nil {
		return 0, 0, err
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	projectIDs, err := ensureProcurementProjectsTx(ctx, tx, tenant.TenantID, items)
	if err != nil {
		return 0, 0, err
	}
	batch := &pgx.Batch{}
	for _, item := range items {
		projectID := projectIDs[item.ProcurementProject]
		batch.Queue(`
INSERT INTO purchasable_materials (
    tenant_id, id_no, sequence_no, procurement_project_id, procurement_project, project_name, brand, spec, unit, purchase_price,
    remark, technical_requirement, min_spec, status
)
VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'active')
ON CONFLICT (tenant_id, id_no) DO UPDATE
SET sequence_no = EXCLUDED.sequence_no,
    procurement_project_id = EXCLUDED.procurement_project_id,
    procurement_project = EXCLUDED.procurement_project,
    project_name = EXCLUDED.project_name,
    brand = EXCLUDED.brand,
    spec = EXCLUDED.spec,
    unit = EXCLUDED.unit,
    purchase_price = EXCLUDED.purchase_price,
    remark = EXCLUDED.remark,
    technical_requirement = EXCLUDED.technical_requirement,
    min_spec = EXCLUDED.min_spec,
    status = 'active',
    updated_at = now()
`, tenant.TenantID, item.IDNo, item.SequenceNo, projectID, item.ProcurementProject, item.ProjectName, item.Brand, item.Spec, item.Unit, item.PurchasePrice, item.Remark, item.TechnicalRequirement, item.MinSpec)
	}
	results := tx.SendBatch(ctx, batch)
	for range items {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return 0, 0, err
		}
	}
	if err := results.Close(); err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	updated := 0
	for _, item := range items {
		if _, ok := existing[item.IDNo]; ok {
			updated++
		}
	}
	created := len(items) - updated
	actor := items[0].Actor
	if actor == "" {
		actor = "system"
	}
	r.audit(ctx, actor, "purchasable_material.import", "purchasable_material", "", "", fmt.Sprintf("created=%d updated=%d", created, updated))
	return created, updated, nil
}

func scanPurchasableMaterial(row scanner) (PurchasableMaterial, error) {
	var item PurchasableMaterial
	err := row.Scan(
		&item.ID,
		&item.IDNo,
		&item.SequenceNo,
		&item.ProcurementProjectID,
		&item.ProcurementProject,
		&item.ProcurementExpiresAt,
		&item.ProcurementProjectStatus,
		&item.ProjectName,
		&item.Brand,
		&item.Spec,
		&item.Unit,
		&item.PurchasePrice,
		&item.Remark,
		&item.TechnicalRequirement,
		&item.MinSpec,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func scanProcurementProject(row scanner) (ProcurementProject, error) {
	var item ProcurementProject
	err := row.Scan(&item.ID, &item.Name, &item.ExpiresAt, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (r *Repository) SaveMaterial(ctx context.Context, id string, input MaterialInput) (Material, error) {
	tenant := TenantFromContext(ctx)
	input = normalizeMaterial(input)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	var purchase MaterialPurchase
	if id == "" && input.PurchaseSerialNo != "" {
		var err error
		purchase, err = r.materialPurchaseBySerial(ctx, input.PurchaseSerialNo)
		if err != nil {
			return Material{}, err
		}
		input = r.applyMaterialPurchaseToMaterialInput(ctx, input, purchase)
	}
	if input.Name == "" || input.Category == "" || input.Spec == "" || input.Unit == "" || input.Stock < 0 || input.WarningLine < 0 || input.UnitPrice < 0 {
		return Material{}, clientError("invalid material input")
	}
	if input.Status == "" {
		input.Status = "normal"
	}
	if !validMaterialStatus(input.Status) {
		return Material{}, clientError("invalid material status")
	}
	if input.ProductType == "" {
		input.ProductType = "consumable"
	}
	if !validMaterialProductType(input.ProductType) {
		return Material{}, clientError("invalid material product type")
	}
	if input.OpenExpireDays < 0 || input.FreezeThawCount < 0 || input.FreezeThawLimit < 0 || input.NearExpiryDays < 0 {
		return Material{}, clientError("invalid material lifecycle input")
	}
	footerSettings, err := r.FooterSettings(ctx)
	if err != nil {
		return Material{}, err
	}
	if id != "" && input.QRCode == "" {
		input.QRCode = materialDetailURL(footerSettings.BaseURL, id)
	}

	if id == "" {
		tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return Material{}, err
		}
		defer func() {
			_ = tx.Rollback(ctx)
		}()
		notifications := make([]Notification, 0, 4)
		item, err := scanMaterial(tx.QueryRow(ctx, fmt.Sprintf(`
INSERT INTO materials (
    tenant_id, name, product_type, category, subcategory, spec, unit, unit_price, stock, warning_line,
    supplier, manufacturer, batch_no, catalog_no, cas_no, grade, concentration,
    parent_material_id, dilution_factor, preparation_method,
    storage_condition, storage_room, storage_cabinet, storage_layer, storage_slot,
    tender_contract, contract_no, remark, certificate_url, standard_certificate_url, attachment_url,
    qr_code, expires_at, opened_at, open_expire_days, freeze_thaw_count, freeze_thaw_limit,
    approval_required, near_expiry_days, status
)
VALUES (
    $40, $1, $2, $3, $4, $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14, $15, $16,
    NULLIF($17, '')::uuid, $18, $19,
    $20, $21, $22, $23, $24,
    $25, $26, $27, $28, $29,
    $30, $31, NULLIF($32, '')::date, NULLIF($33, '')::date, $34, $35, $36,
    $37, $38, $39
)
RETURNING %s
`, materialSelectColumns("materials")), input.Name, input.ProductType, input.Category, input.Subcategory, input.Spec, input.Unit, input.UnitPrice, input.Stock, input.WarningLine, input.Supplier, input.Manufacturer, input.BatchNo, input.CatalogNo, input.CASNo, input.Grade, input.Concentration, input.ParentMaterialID, input.DilutionFactor, input.PreparationMethod, input.StorageCondition, input.StorageRoom, input.StorageCabinet, input.StorageLayer, input.StorageSlot, input.TenderContract, input.ContractNo, input.Remark, input.CertificateURL, input.StandardCertificateURL, input.AttachmentURL, input.QRCode, input.ExpiresAt, input.OpenedAt, input.OpenExpireDays, input.FreezeThawCount, input.FreezeThawLimit, input.ApprovalRequired, input.NearExpiryDays, input.Status, tenant.TenantID))
		if err != nil {
			return Material{}, err
		}
		if input.QRCode == "" {
			item.QRCode = materialDetailURL(footerSettings.BaseURL, item.ID)
			if _, err := tx.Exec(ctx, `
UPDATE materials
SET qr_code = $1
WHERE id = $2 AND tenant_id = $3::uuid
`, item.QRCode, item.ID, tenant.TenantID); err != nil {
				return Material{}, err
			}
		}
		if input.ProductType == "standard" && input.Stock > 0 {
			batchNo := input.BatchNo
			if batchNo == "" {
				batchNo = "默认批次"
			}
			var batchID string
			if _, err := tx.Exec(ctx, `
INSERT INTO material_batches (tenant_id, material_id, batch_no, quantity, expires_at, location, status)
VALUES ($1, $2, $3, $4, NULLIF($5, '')::date, $6, 'active')
ON CONFLICT (tenant_id, material_id, batch_no) DO UPDATE
SET quantity = material_batches.quantity + EXCLUDED.quantity,
    expires_at = COALESCE(EXCLUDED.expires_at, material_batches.expires_at),
    location = COALESCE(NULLIF(EXCLUDED.location, ''), material_batches.location),
    status = 'active',
    updated_at = now()
`, tenant.TenantID, item.ID, batchNo, input.Stock, input.ExpiresAt, materialInputBatchLocation(input)); err != nil {
				return Material{}, err
			}
			if err := tx.QueryRow(ctx, `
SELECT id::text
FROM material_batches
WHERE tenant_id = $1::uuid AND material_id = $2 AND batch_no = $3
`, tenant.TenantID, item.ID, batchNo).Scan(&batchID); err != nil {
				return Material{}, err
			}
			if err := createMaterialUnits(ctx, tx, MaterialUnitGenerationInput{
				TenantID:     tenant.TenantID,
				MaterialID:   item.ID,
				MaterialName: item.Name,
				BatchID:      batchID,
				Quantity:     input.Stock,
				ExpiresAt:    input.ExpiresAt,
				Location:     materialInputBatchLocation(input),
			}); err != nil {
				return Material{}, err
			}
			if _, err := syncStandardMaterialStock(ctx, tx, item.ID, tenant.TenantID); err != nil {
				return Material{}, err
			}
			item, err = scanMaterial(tx.QueryRow(ctx, fmt.Sprintf(`
SELECT %s
FROM materials
WHERE id = $1 AND tenant_id = $2::uuid
`, materialSelectColumns("materials")), item.ID, tenant.TenantID))
			if err != nil {
				return Material{}, err
			}
		}
		if input.ProductType != "standard" && input.Stock > 0 {
			if err := createMaterialUnits(ctx, tx, MaterialUnitGenerationInput{
				TenantID:     tenant.TenantID,
				MaterialID:   item.ID,
				MaterialName: item.Name,
				Quantity:     input.Stock,
				ExpiresAt:    input.ExpiresAt,
				Location:     materialInputBatchLocation(input),
			}); err != nil {
				return Material{}, err
			}
			item, err = scanMaterial(tx.QueryRow(ctx, fmt.Sprintf(`
SELECT %s
FROM materials
WHERE id = $1 AND tenant_id = $2::uuid
`, materialSelectColumns("materials")), item.ID, tenant.TenantID))
			if err != nil {
				return Material{}, err
			}
		}
		if purchase.ID != "" {
			updateTag, err := tx.Exec(ctx, `
UPDATE material_purchases
SET material_id = $2,
    status = 'received',
    received_at = COALESCE(received_at, now())
WHERE id = $1
  AND tenant_id = $3::uuid
  AND status IN ('registered', 'approved', 'ordered')
  AND NOT EXISTS (
      SELECT 1
      FROM material_purchase_monthly_confirmations mpmc
      WHERE mpmc.tenant_id = material_purchases.tenant_id
        AND mpmc.month = to_char(material_purchases.created_at, 'YYYY-MM')
  )
	`, purchase.ID, item.ID, tenant.TenantID)
			if err != nil {
				return Material{}, err
			}
			if updateTag.RowsAffected() > 0 {
				if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, 'receive', '资源入库关联申购流水号')
`, tenant.TenantID, purchase.ID, input.Actor); err != nil {
					return Material{}, err
				}
				body := fmt.Sprintf("%s x%d 已完成入库，资源名称：%s，储存位置：%s。", firstNonEmpty(purchase.PurchaseItemName, purchase.MaterialName, item.Name), purchase.Quantity, item.Name, firstNonEmpty(materialLocation(item), "未登记"))
				created, err := r.createMaterialEventNotificationsTx(ctx, tx, tenant.TenantID, purchase.RequesterID, purchase.GroupName, "耗材申购完成入库", body, "success")
				if err != nil {
					return Material{}, err
				}
				notifications = append(notifications, created...)
			}
		}
		if item.Stock <= item.WarningLine {
			body := fmt.Sprintf("%s 当前库存 %d%s，低于预警线 %d%s。", item.Name, item.Stock, item.Unit, item.WarningLine, item.Unit)
			created, err := r.createMaterialEventNotificationsTx(ctx, tx, tenant.TenantID, purchase.RequesterID, purchase.GroupName, "耗材库存预警", body, "warning")
			if err != nil {
				return Material{}, err
			}
			notifications = append(notifications, created...)
		}
		if materialNearExpiry(input.ExpiresAt, input.NearExpiryDays) {
			body := fmt.Sprintf("%s 有效期为 %s，已进入临期预警范围。", item.Name, input.ExpiresAt)
			created, err := r.createMaterialEventNotificationsTx(ctx, tx, tenant.TenantID, purchase.RequesterID, purchase.GroupName, "耗材有效期告警", body, "warning")
			if err != nil {
				return Material{}, err
			}
			notifications = append(notifications, created...)
		}
		_, err = tx.Exec(ctx, `
INSERT INTO inventory_ledger (tenant_id, material_id, change_qty, reason)
VALUES ($1, $2, $3, '初始入库')
`, tenant.TenantID, item.ID, item.Stock)
		if err != nil {
			return Material{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Material{}, err
		}
		item, err = r.Material(ctx, item.ID)
		if err != nil {
			return Material{}, err
		}
		r.enqueueDingTalkNotifications(notifications...)
		r.audit(ctx, input.Actor, "material.create", "material", item.ID, "", item.Name)
		return item, nil
	}

	item, err := scanMaterial(r.db.QueryRow(ctx, fmt.Sprintf(`
UPDATE materials
SET name = $2, product_type = $3, category = $4, subcategory = $5, spec = $6, unit = $7, unit_price = $8, warning_line = $9,
    supplier = $10, manufacturer = $11, batch_no = $12, catalog_no = $13, cas_no = $14, grade = $15,
    concentration = $16, parent_material_id = NULLIF($17, '')::uuid, dilution_factor = $18, preparation_method = $19,
    storage_condition = $20, storage_room = $21, storage_cabinet = $22, storage_layer = $23, storage_slot = $24,
    tender_contract = $25, contract_no = $26, remark = $27, certificate_url = $28, standard_certificate_url = $29, attachment_url = $30, qr_code = $31,
    expires_at = NULLIF($32, '')::date, opened_at = NULLIF($33, '')::date,
    open_expire_days = $34, freeze_thaw_count = $35, freeze_thaw_limit = $36,
    approval_required = $37, near_expiry_days = $38, status = $39
WHERE id = $1 AND ($40::boolean OR tenant_id = $41::uuid)
RETURNING %s
`, materialSelectColumns("materials")), id, input.Name, input.ProductType, input.Category, input.Subcategory, input.Spec, input.Unit, input.UnitPrice, input.WarningLine, input.Supplier, input.Manufacturer, input.BatchNo, input.CatalogNo, input.CASNo, input.Grade, input.Concentration, input.ParentMaterialID, input.DilutionFactor, input.PreparationMethod, input.StorageCondition, input.StorageRoom, input.StorageCabinet, input.StorageLayer, input.StorageSlot, input.TenderContract, input.ContractNo, input.Remark, input.CertificateURL, input.StandardCertificateURL, input.AttachmentURL, input.QRCode, input.ExpiresAt, input.OpenedAt, input.OpenExpireDays, input.FreezeThawCount, input.FreezeThawLimit, input.ApprovalRequired, input.NearExpiryDays, input.Status, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return Material{}, err
	}
	r.audit(ctx, input.Actor, "material.update", "material", item.ID, "", item.Status)
	return item, nil
}

func (r *Repository) ImportMaterials(ctx context.Context, input MaterialImportInput) (MaterialImportResult, error) {
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	records, err := purchasableMaterialImportRecords(input.Filename, input.Content)
	if err != nil {
		return MaterialImportResult{}, WrapClientError("material import failed", err)
	}
	result := MaterialImportResult{}
	if len(records) == 0 {
		return result, clientError("material import failed: 文件内容为空")
	}
	headerIndex := -1
	for i, row := range records {
		if materialLooksLikeHeader(row) {
			headerIndex = i
			break
		}
	}
	if headerIndex < 0 {
		return result, clientError("material import failed: 未找到表头，请确认包含资源名称、一级目录、规格、单位")
	}
	header := materialImportHeaderIndex(records[headerIndex])
	for rowIndex, row := range records[headerIndex+1:] {
		line := headerIndex + rowIndex + 2
		if rowBlank(row) {
			continue
		}
		materialInput := materialInputFromCSVRow(header, row)
		materialInput.Actor = input.Actor
		materialInput, err = r.applyPurchasableMaterialToMaterialInput(ctx, materialInput, materialCSVValue(header, row, "可采购物资ID号", "采购目录ID号", "ID号", "idNo"))
		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行匹配采购目录失败：%s", line, err.Error()))
			continue
		}
		if materialInput.Name == "" || materialInput.Category == "" || materialInput.Spec == "" || materialInput.Unit == "" {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行缺少资源名称、一级目录、规格或单位", line))
			continue
		}
		id, err := r.materialIDByImportKey(ctx, materialInput)
		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行查询现有产品失败", line))
			continue
		}
		if _, err := r.SaveMaterial(ctx, id, materialInput); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d行导入失败：%s", line, err.Error()))
			continue
		}
		if id == "" {
			result.Created++
		} else {
			result.Updated++
		}
	}
	result.Message = fmt.Sprintf("导入完成：新增 %d 行，更新 %d 行，跳过 %d 行。", result.Created, result.Updated, result.Skipped)
	return result, nil
}

func (r *Repository) ImportMaterialsCSV(ctx context.Context, content string, actor string) (MaterialImportResult, error) {
	return r.ImportMaterials(ctx, MaterialImportInput{Filename: "materials.csv", Content: []byte(content), Actor: actor})
}

func (r *Repository) applyPurchasableMaterialToMaterialInput(ctx context.Context, input MaterialInput, purchasableIDNo string) (MaterialInput, error) {
	purchasableIDNo = strings.TrimSpace(purchasableIDNo)
	if purchasableIDNo == "" {
		return normalizeMaterial(input), nil
	}
	tenant := TenantFromContext(ctx)
	var purchasable PurchasableMaterial
	err := r.db.QueryRow(ctx, `
SELECT pm.id::text, pm.id_no, pm.sequence_no, COALESCE(pm.procurement_project_id::text, ''),
       COALESCE(pp.name, pm.procurement_project), COALESCE(pp.expires_at::text, ''),
       COALESCE(pp.status, 'active'),
       pm.project_name, pm.brand, pm.spec, pm.unit, pm.purchase_price::float8,
       pm.remark, pm.technical_requirement, pm.min_spec, pm.status, pm.created_at, pm.updated_at
FROM purchasable_materials pm
LEFT JOIN procurement_projects pp ON pp.id = pm.procurement_project_id
WHERE pm.status = 'active'
  AND pm.id_no = $1
  AND ($2::boolean OR pm.tenant_id = $3::uuid)
LIMIT 1
`, purchasableIDNo, tenant.AllTenants, tenant.TenantID).Scan(
		&purchasable.ID, &purchasable.IDNo, &purchasable.SequenceNo, &purchasable.ProcurementProjectID, &purchasable.ProcurementProject, &purchasable.ProcurementExpiresAt, &purchasable.ProcurementProjectStatus, &purchasable.ProjectName, &purchasable.Brand, &purchasable.Spec, &purchasable.Unit, &purchasable.PurchasePrice, &purchasable.Remark, &purchasable.TechnicalRequirement, &purchasable.MinSpec, &purchasable.Status, &purchasable.CreatedAt, &purchasable.UpdatedAt,
	)
	if err != nil {
		return input, err
	}
	if input.Name == "" {
		input.Name = purchasable.ProjectName
	}
	input.Spec = firstNonEmpty(input.Spec, purchasable.Spec)
	input.Unit = firstNonEmpty(input.Unit, purchasable.Unit)
	if input.UnitPrice <= 0 {
		input.UnitPrice = purchasable.PurchasePrice
	}
	input.Supplier = firstNonEmpty(input.Supplier, purchasable.Brand)
	input.Manufacturer = firstNonEmpty(input.Manufacturer, purchasable.Brand)
	input.CatalogNo = firstNonEmpty(input.CatalogNo, purchasable.IDNo)
	input.TenderContract = firstNonEmpty(input.TenderContract, purchasable.ProcurementProject)
	input.ContractNo = firstNonEmpty(input.ContractNo, purchasable.ProcurementProject)
	input.Remark = firstNonEmpty(input.Remark, purchasable.Remark)
	return normalizeMaterial(input), nil
}

func (r *Repository) materialIDByImportKey(ctx context.Context, input MaterialInput) (string, error) {
	tenant := TenantFromContext(ctx)
	var id string
	if input.QRCode != "" {
		err := r.db.QueryRow(ctx, `
SELECT id::text
FROM materials
WHERE qr_code = $1 AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY created_at DESC
LIMIT 1
`, input.QRCode, tenant.AllTenants, tenant.TenantID).Scan(&id)
		if err == nil || !errors.Is(err, pgx.ErrNoRows) {
			return id, err
		}
	}
	if input.BatchNo != "" {
		err := r.db.QueryRow(ctx, `
SELECT id::text
FROM materials
WHERE name = $1 AND batch_no = $2 AND ($3::boolean OR tenant_id = $4::uuid)
ORDER BY created_at DESC
LIMIT 1
`, input.Name, input.BatchNo, tenant.AllTenants, tenant.TenantID).Scan(&id)
		if err == nil || !errors.Is(err, pgx.ErrNoRows) {
			return id, err
		}
	}
	return "", nil
}

func (r *Repository) DeleteMaterial(ctx context.Context, id string, actor string) (Material, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var oldStatus string
	if err := r.db.QueryRow(ctx, `SELECT status FROM materials WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)`, id, tenant.AllTenants, tenant.TenantID).Scan(&oldStatus); err != nil {
		return Material{}, err
	}
	item, err := scanMaterial(r.db.QueryRow(ctx, fmt.Sprintf(`
UPDATE materials
SET status = 'disabled'
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
RETURNING %s
`, materialSelectColumns("materials")), id, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return Material{}, err
	}
	r.audit(ctx, actor, "material.delete", "material", item.ID, oldStatus, item.Status)
	return item, nil
}

func (r *Repository) AdjustMaterialStock(ctx context.Context, id string, input StockAdjustmentInput) (Material, error) {
	tenant := TenantFromContext(ctx)
	input.Reason = strings.TrimSpace(input.Reason)
	input.BatchID = strings.TrimSpace(input.BatchID)
	input.BatchNo = strings.TrimSpace(input.BatchNo)
	input.UnitID = strings.TrimSpace(input.UnitID)
	input.ExpiresAt = strings.TrimSpace(input.ExpiresAt)
	input.Location = strings.TrimSpace(input.Location)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if id == "" || input.ChangeQty == 0 || input.Reason == "" {
		return Material{}, clientError("invalid stock adjustment input")
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Material{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)

	var stock int
	var productType, materialTenantID string
	if err := tx.QueryRow(ctx, `SELECT stock, product_type, tenant_id::text FROM materials WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid) FOR UPDATE`, id, tenant.AllTenants, tenant.TenantID).Scan(&stock, &productType, &materialTenantID); err != nil {
		return Material{}, err
	}
	if err := reconcileMaterialUnits(ctx, tx, id, materialTenantID); err != nil {
		return Material{}, err
	}
	if productType == "standard" {
		item, err := r.adjustStandardMaterialStock(ctx, tx, id, materialTenantID, input)
		if err != nil {
			return Material{}, err
		}
		if item.Stock <= item.WarningLine {
			requesterID, requesterGroup, err := r.materialRequesterForMaterialTx(ctx, tx, materialTenantID, item.ID)
			if err != nil {
				return Material{}, err
			}
			created, err := r.createMaterialEventNotificationsTx(ctx, tx, materialTenantID, requesterID, requesterGroup, "耗材库存预警", fmt.Sprintf("%s 当前库存 %d%s，低于预警线 %d%s。", item.Name, item.Stock, item.Unit, item.WarningLine, item.Unit), "warning")
			if err != nil {
				return Material{}, err
			}
			notifications = append(notifications, created...)
		}
		if err := tx.Commit(ctx); err != nil {
			return Material{}, err
		}
		r.enqueueDingTalkNotifications(notifications...)
		item, err = r.Material(ctx, item.ID)
		if err != nil {
			return Material{}, err
		}
		r.audit(ctx, input.Actor, "material.stock_adjust", "material", item.ID, fmt.Sprint(stock), fmt.Sprint(item.Stock))
		return item, nil
	}
	if stock+input.ChangeQty < 0 {
		return Material{}, clientError("material stock cannot be negative")
	}
	unitID := input.UnitID
	unitCode := ""
	if productType != "standard" && input.ChangeQty < 0 {
		if unitID != "" && input.ChangeQty != -1 {
			return Material{}, clientError("material unit adjustment quantity must be -1")
		}
		if -input.ChangeQty == 1 && unitID != "" {
			if err := tx.QueryRow(ctx, `
SELECT unit_code
FROM material_units
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'available'
FOR UPDATE
`, unitID, id, materialTenantID).Scan(&unitCode); err != nil {
				return Material{}, err
			}
		} else {
			unitIDs, unitCodes, err := availableMaterialUnitsForDeduction(ctx, tx, id, materialTenantID, -input.ChangeQty)
			if err != nil {
				return Material{}, err
			}
			if len(unitIDs) != -input.ChangeQty {
				return Material{}, clientError("insufficient material unit stock")
			}
			unitID = strings.Join(unitIDs, ",")
			unitCode = strings.Join(unitCodes, "，")
		}
		updateTag, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'used', updated_at = now()
WHERE id::text = ANY($1::text[]) AND material_id = $2 AND tenant_id = $3::uuid AND status = 'available'
`, strings.Split(unitID, ","), id, materialTenantID)
		if err != nil {
			return Material{}, err
		}
		if updateTag.RowsAffected() != int64(-input.ChangeQty) {
			return Material{}, clientError("material unit stock changed")
		}
	}
	item, err := scanMaterial(tx.QueryRow(ctx, fmt.Sprintf(`
UPDATE materials
SET stock = stock + $2
WHERE id = $1 AND ($3::boolean OR tenant_id = $4::uuid)
RETURNING %s
`, materialSelectColumns("materials")), id, input.ChangeQty, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return Material{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO inventory_ledger (tenant_id, material_id, change_qty, reason)
VALUES ($1, $2, $3, $4)
`, tenant.TenantID, item.ID, input.ChangeQty, materialUnitReason(input.Reason, "", unitCode)); err != nil {
		return Material{}, err
	}
	if productType != "standard" && input.ChangeQty > 0 {
		if err := createMaterialUnits(ctx, tx, MaterialUnitGenerationInput{
			TenantID:     materialTenantID,
			MaterialID:   item.ID,
			MaterialName: item.Name,
			Quantity:     input.ChangeQty,
			ExpiresAt:    input.ExpiresAt,
			Location:     input.Location,
		}); err != nil {
			return Material{}, err
		}
		item, err = scanMaterial(tx.QueryRow(ctx, fmt.Sprintf(`
SELECT %s
FROM materials
WHERE id = $1 AND tenant_id = $2::uuid
`, materialSelectColumns("materials")), item.ID, materialTenantID))
		if err != nil {
			return Material{}, err
		}
	}
	if item.Stock <= item.WarningLine {
		requesterID, requesterGroup, err := r.materialRequesterForMaterialTx(ctx, tx, materialTenantID, item.ID)
		if err != nil {
			return Material{}, err
		}
		created, err := r.createMaterialEventNotificationsTx(ctx, tx, materialTenantID, requesterID, requesterGroup, "耗材库存预警", fmt.Sprintf("%s 当前库存 %d%s，低于预警线 %d%s。", item.Name, item.Stock, item.Unit, item.WarningLine, item.Unit), "warning")
		if err != nil {
			return Material{}, err
		}
		notifications = append(notifications, created...)
	}
	if err := tx.Commit(ctx); err != nil {
		return Material{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	r.audit(ctx, input.Actor, "material.stock_adjust", "material", item.ID, fmt.Sprint(stock), fmt.Sprint(item.Stock))
	return item, nil
}

func (r *Repository) adjustStandardMaterialStock(ctx context.Context, tx pgx.Tx, materialID string, materialTenantID string, input StockAdjustmentInput) (Material, error) {
	batchNo := input.BatchNo
	batchID := input.BatchID
	materialName := materialNameForUnitGeneration(ctx, tx, materialID)
	unitCode := ""
	if input.ChangeQty < 0 {
		if input.UnitID != "" {
			if input.ChangeQty != -1 {
				return Material{}, clientError("material unit adjustment quantity must be -1")
			}
			var unitBatchID string
			if err := tx.QueryRow(ctx, `
SELECT COALESCE(batch_id::text, ''), unit_code
FROM material_units
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'available'
FOR UPDATE
`, input.UnitID, materialID, materialTenantID).Scan(&unitBatchID, &unitCode); err != nil {
				return Material{}, err
			}
			if unitBatchID != "" {
				batchID = unitBatchID
				_ = tx.QueryRow(ctx, `SELECT batch_no FROM material_batches WHERE id = $1`, batchID).Scan(&batchNo)
			}
			if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'used', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'available'
`, input.UnitID, materialID, materialTenantID); err != nil {
				return Material{}, err
			}
			if err := syncMaterialBatchQuantity(ctx, tx, batchID); err != nil {
				return Material{}, err
			}
		} else if input.BatchID != "" {
			if err := tx.QueryRow(ctx, `
SELECT batch_no
FROM material_batches
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid
FOR UPDATE
`, input.BatchID, materialID, materialTenantID).Scan(&batchNo); err != nil {
				return Material{}, err
			}
			unitIDs, unitCodes, err := availableMaterialUnitsForBatchDeduction(ctx, tx, materialID, materialTenantID, input.BatchID, -input.ChangeQty)
			if err != nil {
				return Material{}, err
			}
			if len(unitIDs) != -input.ChangeQty {
				return Material{}, clientError("insufficient material unit stock")
			}
			unitCode = strings.Join(unitCodes, "，")
			if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'used', updated_at = now()
WHERE id::text = ANY($1::text[]) AND material_id = $2 AND tenant_id = $3::uuid AND status = 'available'
`, unitIDs, materialID, materialTenantID); err != nil {
				return Material{}, err
			}
			if err := syncMaterialBatchQuantity(ctx, tx, input.BatchID); err != nil {
				return Material{}, err
			}
		} else {
			return Material{}, clientError("standard material outbound requires batch or unit")
		}
	} else if input.BatchID != "" {
		if err := tx.QueryRow(ctx, `
SELECT batch_no
FROM material_batches
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid
FOR UPDATE
`, input.BatchID, materialID, materialTenantID).Scan(&batchNo); err != nil {
			return Material{}, err
		}
		if err := createMaterialUnits(ctx, tx, MaterialUnitGenerationInput{
			TenantID:     materialTenantID,
			MaterialID:   materialID,
			MaterialName: materialName,
			BatchID:      batchID,
			Quantity:     input.ChangeQty,
			ExpiresAt:    input.ExpiresAt,
			Location:     input.Location,
		}); err != nil {
			return Material{}, err
		}
		if _, err := tx.Exec(ctx, `
UPDATE material_batches
SET expires_at = COALESCE(NULLIF($2, '')::date, expires_at),
    location = COALESCE(NULLIF($3, ''), location),
    updated_at = now()
WHERE id = $1 AND material_id = $4 AND tenant_id = $5::uuid
`, input.BatchID, input.ExpiresAt, input.Location, materialID, materialTenantID); err != nil {
			return Material{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, input.BatchID); err != nil {
			return Material{}, err
		}
	} else {
		if batchNo == "" {
			return Material{}, clientError("standard material inbound requires batch number")
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO material_batches (tenant_id, material_id, batch_no, quantity, expires_at, location, status)
VALUES ($1, $2, $3, $4, NULLIF($5, '')::date, $6, 'active')
ON CONFLICT (tenant_id, material_id, batch_no) DO UPDATE
SET quantity = material_batches.quantity + EXCLUDED.quantity,
    expires_at = COALESCE(EXCLUDED.expires_at, material_batches.expires_at),
    location = COALESCE(NULLIF(EXCLUDED.location, ''), material_batches.location),
    status = 'active',
    updated_at = now()
`, materialTenantID, materialID, batchNo, input.ChangeQty, input.ExpiresAt, input.Location); err != nil {
			return Material{}, err
		}
		if err := tx.QueryRow(ctx, `
SELECT id::text
FROM material_batches
WHERE tenant_id = $1::uuid AND material_id = $2 AND batch_no = $3
`, materialTenantID, materialID, batchNo).Scan(&batchID); err != nil {
			return Material{}, err
		}
		if err := createMaterialUnits(ctx, tx, MaterialUnitGenerationInput{
			TenantID:     materialTenantID,
			MaterialID:   materialID,
			MaterialName: materialName,
			BatchID:      batchID,
			Quantity:     input.ChangeQty,
			ExpiresAt:    input.ExpiresAt,
			Location:     input.Location,
		}); err != nil {
			return Material{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, batchID); err != nil {
			return Material{}, err
		}
	}
	if _, err := syncMaterialStock(ctx, tx, materialID, materialTenantID); err != nil {
		return Material{}, err
	}
	ledgerReason := materialUnitReason(input.Reason, batchNo, unitCode)
	if _, err := tx.Exec(ctx, `
INSERT INTO inventory_ledger (tenant_id, material_id, change_qty, reason)
VALUES ($1, $2, $3, $4)
`, materialTenantID, materialID, input.ChangeQty, ledgerReason); err != nil {
		return Material{}, err
	}
	item, err := scanMaterial(tx.QueryRow(ctx, fmt.Sprintf(`
SELECT %s
FROM materials
WHERE id = $1 AND tenant_id = $2::uuid
`, materialSelectColumns("materials")), materialID, materialTenantID))
	if err != nil {
		return Material{}, err
	}
	return item, nil
}

func syncMaterialStock(ctx context.Context, tx pgx.Tx, materialID string, materialTenantID string) (int, error) {
	var stock int
	err := tx.QueryRow(ctx, `
UPDATE materials
SET stock = (
    SELECT count(*)::int
    FROM material_units
    WHERE material_id = $1
      AND tenant_id = $2::uuid
      AND status IN ('available', 'reserved')
)
WHERE id = $1 AND tenant_id = $2::uuid
RETURNING stock
`, materialID, materialTenantID).Scan(&stock)
	return stock, err
}

func syncStandardMaterialStock(ctx context.Context, tx pgx.Tx, materialID string, materialTenantID string) (int, error) {
	return syncMaterialStock(ctx, tx, materialID, materialTenantID)
}

func reconcileMaterialUnits(ctx context.Context, tx pgx.Tx, materialID string, materialTenantID string) error {
	var existingUnits int
	if err := tx.QueryRow(ctx, `
SELECT count(*)::int
FROM material_units
WHERE material_id = $1 AND tenant_id = $2::uuid
`, materialID, materialTenantID).Scan(&existingUnits); err != nil {
		return err
	}
	if existingUnits > 0 {
		return nil
	}
	var name, expiresAt, location string
	var stock int
	if err := tx.QueryRow(ctx, `
SELECT name, stock, COALESCE(expires_at::text, ''),
       concat_ws(' / ', NULLIF(storage_room, ''), NULLIF(storage_cabinet, ''), NULLIF(storage_layer, ''), NULLIF(storage_slot, ''))
FROM materials
WHERE id = $1 AND tenant_id = $2::uuid
`, materialID, materialTenantID).Scan(&name, &stock, &expiresAt, &location); err != nil {
		return err
	}
	return createMaterialUnits(ctx, tx, MaterialUnitGenerationInput{
		TenantID:     materialTenantID,
		MaterialID:   materialID,
		MaterialName: name,
		Quantity:     stock,
		ExpiresAt:    expiresAt,
		Location:     location,
	})
}

func backfillMaterialUnits(ctx context.Context, tx pgx.Tx) error {
	inputs := make([]MaterialUnitGenerationInput, 0)
	batchRows, err := tx.Query(ctx, `
SELECT mb.tenant_id::text, mb.material_id::text, m.name, mb.id::text,
       (mb.quantity - (
           SELECT count(*)::int
           FROM material_units mu
           WHERE mu.batch_id = mb.id
             AND mu.status IN ('available', 'reserved')
       ))::int,
       COALESCE(mb.expires_at::text, ''), mb.location
FROM material_batches mb
JOIN materials m ON m.id = mb.material_id
WHERE mb.quantity > (
      SELECT count(*)::int
      FROM material_units mu
      WHERE mu.batch_id = mb.id
        AND mu.status IN ('available', 'reserved')
  )
ORDER BY mb.created_at, mb.id
`)
	if err != nil {
		return err
	}
	for batchRows.Next() {
		var input MaterialUnitGenerationInput
		if err := batchRows.Scan(&input.TenantID, &input.MaterialID, &input.MaterialName, &input.BatchID, &input.Quantity, &input.ExpiresAt, &input.Location); err != nil {
			batchRows.Close()
			return err
		}
		inputs = append(inputs, input)
	}
	if err := batchRows.Err(); err != nil {
		batchRows.Close()
		return err
	}
	batchRows.Close()

	for _, input := range inputs {
		if err := createMaterialUnits(ctx, tx, input); err != nil {
			return err
		}
	}

	inputs = inputs[:0]
	materialRows, err := tx.Query(ctx, `
SELECT m.tenant_id::text, m.id::text, m.name,
       (m.stock - (
           SELECT count(*)::int
           FROM material_units mu
           WHERE mu.material_id = m.id
             AND mu.status IN ('available', 'reserved')
       ))::int,
       COALESCE(m.expires_at::text, ''),
       concat_ws(' / ', NULLIF(m.storage_room, ''), NULLIF(m.storage_cabinet, ''), NULLIF(m.storage_layer, ''), NULLIF(m.storage_slot, ''))
FROM materials m
WHERE m.product_type <> 'standard'
  AND m.stock > (
      SELECT count(*)::int
      FROM material_units mu
      WHERE mu.material_id = m.id
        AND mu.status IN ('available', 'reserved')
  )
ORDER BY m.created_at, m.id
`)
	if err != nil {
		return err
	}
	for materialRows.Next() {
		var input MaterialUnitGenerationInput
		if err := materialRows.Scan(&input.TenantID, &input.MaterialID, &input.MaterialName, &input.Quantity, &input.ExpiresAt, &input.Location); err != nil {
			materialRows.Close()
			return err
		}
		inputs = append(inputs, input)
	}
	if err := materialRows.Err(); err != nil {
		materialRows.Close()
		return err
	}
	materialRows.Close()

	for _, input := range inputs {
		if err := createMaterialUnits(ctx, tx, input); err != nil {
			return err
		}
	}
	return nil
}

func normalizeMaterialUnitCodes(ctx context.Context, tx pgx.Tx) error {
	rows, err := tx.Query(ctx, `
SELECT mu.id::text, mu.tenant_id::text, m.name, mu.unit_code
FROM material_units mu
JOIN materials m ON m.id = mu.material_id
ORDER BY mu.created_at, mu.id
`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type unitCodeCandidate struct {
		id       string
		tenantID string
		prefix   string
		datePart string
		unitCode string
	}
	candidates := make([]unitCodeCandidate, 0)
	for rows.Next() {
		var candidate unitCodeCandidate
		var materialName string
		if err := rows.Scan(&candidate.id, &candidate.tenantID, &materialName, &candidate.unitCode); err != nil {
			return err
		}
		candidate.prefix = materialUnitCodePrefix(materialName)
		candidate.datePart = materialUnitCodeDatePart(candidate.unitCode)
		if !materialUnitCodeMatchesRule(candidate.unitCode, candidate.prefix, candidate.datePart) {
			candidates = append(candidates, candidate)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()
	for _, candidate := range candidates {
		for {
			sequence, err := nextMaterialUnitCodeSequence(ctx, tx, candidate.tenantID, candidate.prefix, candidate.datePart)
			if err != nil {
				return err
			}
			unitCode := fmt.Sprintf("%s-%s-%04d", candidate.prefix, candidate.datePart, sequence)
			updateTag, err := tx.Exec(ctx, `
UPDATE material_units
SET unit_code = $2, updated_at = now()
WHERE id = $1
  AND NOT EXISTS (
      SELECT 1
      FROM material_units existing
      WHERE existing.tenant_id = material_units.tenant_id
        AND existing.unit_code = $2
        AND existing.id <> material_units.id
  )
`, candidate.id, unitCode)
			if err != nil {
				return err
			}
			if updateTag.RowsAffected() == 1 {
				break
			}
		}
	}
	return nil
}

func materialUnitCodeDatePart(unitCode string) string {
	parts := strings.Split(unitCode, "-")
	if len(parts) >= 5 && len(parts[len(parts)-4]) == 4 && len(parts[len(parts)-3]) == 2 && len(parts[len(parts)-2]) == 2 {
		datePart := strings.Join(parts[len(parts)-4:len(parts)-1], "-")
		if _, err := time.Parse("2006-01-02", datePart); err == nil {
			return datePart
		}
	}
	return appDateString()
}

func availableMaterialUnitsForDeduction(ctx context.Context, tx pgx.Tx, materialID string, materialTenantID string, quantity int) ([]string, []string, error) {
	if quantity <= 0 {
		return nil, nil, nil
	}
	rows, err := tx.Query(ctx, `
SELECT id::text, unit_code
FROM material_units
WHERE material_id = $1
  AND tenant_id = $2::uuid
  AND status = 'available'
ORDER BY expires_at NULLS LAST, created_at, unit_code
LIMIT $3
FOR UPDATE
`, materialID, materialTenantID, quantity)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, quantity)
	codes := make([]string, 0, quantity)
	for rows.Next() {
		var id, code string
		if err := rows.Scan(&id, &code); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return ids, codes, nil
}

func availableMaterialUnitsForBatchDeduction(ctx context.Context, tx pgx.Tx, materialID string, materialTenantID string, batchID string, quantity int) ([]string, []string, error) {
	if quantity <= 0 {
		return nil, nil, nil
	}
	rows, err := tx.Query(ctx, `
SELECT id::text, unit_code
FROM material_units
WHERE material_id = $1
  AND tenant_id = $2::uuid
  AND batch_id = $3::uuid
  AND status = 'available'
ORDER BY expires_at NULLS LAST, created_at, unit_code
LIMIT $4
FOR UPDATE
`, materialID, materialTenantID, batchID, quantity)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, quantity)
	codes := make([]string, 0, quantity)
	for rows.Next() {
		var id, code string
		if err := rows.Scan(&id, &code); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return ids, codes, nil
}

func syncMaterialBatchQuantity(ctx context.Context, tx pgx.Tx, batchID string) error {
	if strings.TrimSpace(batchID) == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `
UPDATE material_batches
SET quantity = (
    SELECT count(*)::int
    FROM material_units
    WHERE batch_id = $1
      AND status IN ('available', 'reserved')
),
status = CASE
    WHEN (
        SELECT count(*)::int
        FROM material_units
        WHERE batch_id = $1
          AND status IN ('available', 'reserved')
    ) = 0 THEN 'depleted'
    ELSE 'active'
END,
updated_at = now()
WHERE id = $1
`, batchID)
	return err
}

type MaterialUnitGenerationInput struct {
	TenantID     string
	MaterialID   string
	MaterialName string
	BatchID      string
	Quantity     int
	ExpiresAt    string
	Location     string
}

func createMaterialUnits(ctx context.Context, tx pgx.Tx, input MaterialUnitGenerationInput) error {
	if input.Quantity <= 0 {
		return nil
	}
	prefix := materialUnitCodePrefix(input.MaterialName)
	datePart := appDateString()
	for index := 0; index < input.Quantity; index++ {
		for {
			sequence, err := nextMaterialUnitCodeSequence(ctx, tx, input.TenantID, prefix, datePart)
			if err != nil {
				return err
			}
			unitCode := fmt.Sprintf("%s-%s-%04d", prefix, datePart, sequence)
			insertTag, err := tx.Exec(ctx, `
INSERT INTO material_units (tenant_id, material_id, batch_id, unit_code, expires_at, location, status)
VALUES ($1, $2, NULLIF($3, '')::uuid, $4, NULLIF($5, '')::date, $6, 'available')
ON CONFLICT (tenant_id, unit_code) DO NOTHING
`, input.TenantID, input.MaterialID, input.BatchID, unitCode, input.ExpiresAt, input.Location)
			if err != nil {
				return err
			}
			if insertTag.RowsAffected() == 1 {
				break
			}
		}
	}
	if err := syncMaterialBatchQuantity(ctx, tx, input.BatchID); err != nil {
		return err
	}
	_, err := syncMaterialStock(ctx, tx, input.MaterialID, input.TenantID)
	return err
}

func nextMaterialUnitCodeSequence(ctx context.Context, tx pgx.Tx, tenantID string, prefix string, datePart string) (int, error) {
	rows, err := tx.Query(ctx, `
SELECT unit_code
FROM material_units
WHERE tenant_id = $1::uuid
  AND unit_code LIKE $2
`, tenantID, prefix+"-"+datePart+"-%")
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	maxSequence := 0
	for rows.Next() {
		var unitCode string
		if err := rows.Scan(&unitCode); err != nil {
			return 0, err
		}
		sequence, ok := materialUnitCodeSequence(unitCode, prefix, datePart)
		if ok && sequence > maxSequence {
			maxSequence = sequence
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return maxSequence + 1, nil
}

func materialUnitCodeSequence(unitCode string, prefix string, datePart string) (int, bool) {
	suffix := strings.TrimPrefix(unitCode, prefix+"-"+datePart+"-")
	if suffix == unitCode || len(suffix) != 4 {
		return 0, false
	}
	sequence, err := strconv.Atoi(suffix)
	if err != nil || sequence <= 0 {
		return 0, false
	}
	return sequence, true
}

func materialUnitCodeMatchesRule(unitCode string, prefix string, datePart string) bool {
	sequence, ok := materialUnitCodeSequence(unitCode, prefix, datePart)
	return ok && unitCode == fmt.Sprintf("%s-%s-%04d", prefix, datePart, sequence)
}

func materialNameForUnitGeneration(ctx context.Context, tx pgx.Tx, materialID string) string {
	var name string
	_ = tx.QueryRow(ctx, `SELECT name FROM materials WHERE id = $1`, materialID).Scan(&name)
	return name
}

func materialUnitCodePrefix(name string) string {
	words := strings.FieldsFunc(strings.TrimSpace(name), func(r rune) bool {
		return unicode.IsSpace(r) || r == '-' || r == '_' || r == '/' || r == '\\' || r == '(' || r == ')' || r == '（' || r == '）'
	})
	if len(words) == 0 {
		return "WP"
	}
	var builder strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		for _, r := range word {
			builder.WriteString(materialRuneInitial(r))
		}
	}
	prefix := builder.String()
	if prefix == "" {
		return "WP"
	}
	return prefix
}

func materialRuneInitial(r rune) string {
	if r <= unicode.MaxASCII {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return strings.ToUpper(string(r))
		}
		return ""
	}
	if initial, ok := commonChineseInitials[r]; ok {
		return initial
	}
	switch {
	case r >= '阿' && r <= '凹':
		return "A"
	case r >= '八' && r <= '簿':
		return "B"
	case r >= '嚓' && r <= '错':
		return "C"
	case r >= '咑' && r <= '鵽':
		return "D"
	case r >= '妸' && r <= '樲':
		return "E"
	case r >= '发' && r <= '猤':
		return "F"
	case r >= '旮' && r <= '腂':
		return "G"
	case r >= '妎' && r <= '夻':
		return "H"
	case r >= '丌' && r <= '攈':
		return "J"
	case r >= '咔' && r <= '穒':
		return "K"
	case r >= '垃' && r <= '擽':
		return "L"
	case r >= '嘸' && r <= '椧':
		return "M"
	case r >= '拏' && r <= '瘧':
		return "N"
	case r >= '筽' && r <= '漚':
		return "O"
	case r >= '妑' && r <= '曝':
		return "P"
	case r >= '七' && r <= '裠':
		return "Q"
	case r >= '亽' && r <= '鶸':
		return "R"
	case r >= '仨' && r <= '蜶':
		return "S"
	case r >= '侤' && r <= '籜':
		return "T"
	case r >= '屲' && r <= '鶩':
		return "W"
	case r >= '夕' && r <= '鑂':
		return "X"
	case r >= '丫' && r <= '韻':
		return "Y"
	case r >= '帀' && r <= '咗':
		return "Z"
	default:
		return "X"
	}
}

var commonChineseInitials = map[rune]string{
	'铅': "Q",
	'标': "B",
	'准': "Z",
	'溶': "R",
	'液': "Y",
	'无': "W",
	'菌': "J",
	'移': "Y",
	'吸': "X",
	'头': "T",
	'琼': "Q",
	'脂': "Z",
	'糖': "T",
	'缓': "H",
	'冲': "C",
	'离': "L",
	'心': "X",
	'管': "G",
	'瓶': "P",
	'盒': "H",
	'包': "B",
	'试': "S",
	'剂': "J",
	'耗': "H",
	'材': "C",
}

func materialBatchReason(reason string, batchNo string) string {
	reason = strings.TrimSpace(reason)
	batchNo = strings.TrimSpace(batchNo)
	if batchNo == "" {
		return reason
	}
	return fmt.Sprintf("%s（批次：%s）", reason, batchNo)
}

func materialUnitReason(reason string, batchNo string, unitCode string) string {
	reason = strings.TrimSpace(reason)
	parts := make([]string, 0, 2)
	if strings.TrimSpace(batchNo) != "" {
		parts = append(parts, "批次："+strings.TrimSpace(batchNo))
	}
	if strings.TrimSpace(unitCode) != "" {
		parts = append(parts, "编号："+strings.TrimSpace(unitCode))
	}
	if len(parts) == 0 {
		return reason
	}
	return fmt.Sprintf("%s（%s）", reason, strings.Join(parts, "，"))
}

func materialInputBatchLocation(input MaterialInput) string {
	return strings.Join(nonEmptyStrings(input.StorageRoom, input.StorageCabinet, input.StorageLayer, input.StorageSlot), " / ")
}

func materialLocation(item Material) string {
	return strings.Join(nonEmptyStrings(item.StorageRoom, item.StorageCabinet, item.StorageLayer, item.StorageSlot), " / ")
}

func materialNearExpiry(expiresAt string, nearExpiryDays int) bool {
	expiresAt = strings.TrimSpace(expiresAt)
	if expiresAt == "" {
		return false
	}
	if nearExpiryDays < 0 {
		nearExpiryDays = 0
	}
	expiry, err := time.Parse("2006-01-02", expiresAt)
	if err != nil {
		return false
	}
	today := appToday()
	return !expiry.Before(today) && !expiry.After(today.AddDate(0, 0, nearExpiryDays))
}

func nonEmptyStrings(values ...string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			items = append(items, value)
		}
	}
	return items
}

func (r *Repository) MaterialRequests(ctx context.Context) ([]MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT mr.id::text, mr.material_id::text, m.name, COALESCE(mr.requester_id::text, ''),
       mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       COALESCE(mr.unit_id::text, ''), COALESCE(mu.unit_code, ''), COALESCE(mu.location, mb.location, ''),
       mr.quantity, mr.purpose, mr.status, mr.created_at
FROM material_requests mr
JOIN materials m ON m.id = mr.material_id
LEFT JOIN material_batches mb ON mb.id = mr.batch_id
LEFT JOIN material_units mu ON mu.id = mr.unit_id
WHERE ($1::boolean OR mr.tenant_id = $2::uuid)
ORDER BY mr.created_at DESC
LIMIT 100
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialRequest, 0)
	for rows.Next() {
		item, err := scanMaterialRequest(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) MaterialRequestsForMonth(ctx context.Context, month string) ([]MaterialRequestExportRow, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT mr.id::text, mr.material_id::text, m.name, COALESCE(mr.requester_id::text, ''),
       mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       COALESCE(mr.unit_id::text, ''), COALESCE(mu.unit_code, ''),
       COALESCE(NULLIF(mu.location, ''), NULLIF(mb.location, ''), NULLIF(concat_ws(' / ', NULLIF(m.storage_room, ''), NULLIF(m.storage_cabinet, ''), NULLIF(m.storage_layer, ''), NULLIF(m.storage_slot, '')), ''), ''),
       mr.quantity, mr.purpose, mr.status, mr.created_at,
       COALESCE(NULLIF(m.catalog_no, ''), NULLIF(m.cas_no, ''), NULLIF(m.grade, ''), ''),
       COALESCE(NULLIF(m.manufacturer, ''), NULLIF(m.supplier, ''), ''),
       m.spec,
       m.unit,
       COALESCE(mu.expires_at::text, mb.expires_at::text, m.expires_at::text, ''),
       COALESCE((
           SELECT string_agg(maa.actor, '，' ORDER BY maa.created_at)
           FROM material_approval_actions maa
           WHERE maa.material_request_id = mr.id
             AND maa.action IN ('approve', 'outbound')
       ), '')
FROM material_requests mr
JOIN materials m ON m.id = mr.material_id
LEFT JOIN material_batches mb ON mb.id = mr.batch_id
LEFT JOIN material_units mu ON mu.id = mr.unit_id
WHERE ($1::boolean OR mr.tenant_id = $2::uuid)
  AND to_char(mr.created_at, 'YYYY-MM') = $3
ORDER BY mr.created_at, mr.id
`, tenant.AllTenants, tenant.TenantID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialRequestExportRow, 0)
	for rows.Next() {
		var item MaterialRequestExportRow
		err := rows.Scan(
			&item.ID,
			&item.MaterialID,
			&item.MaterialName,
			&item.RequesterID,
			&item.Requester,
			&item.GroupName,
			&item.BatchID,
			&item.BatchNo,
			&item.UnitID,
			&item.UnitCode,
			&item.Location,
			&item.Quantity,
			&item.Purpose,
			&item.Status,
			&item.CreatedAt,
			&item.StandardNo,
			&item.Brand,
			&item.Spec,
			&item.Unit,
			&item.ExpiresAt,
			&item.ApprovalInfo,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) MaterialRequest(ctx context.Context, id string) (MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	return scanMaterialRequest(r.db.QueryRow(ctx, `
SELECT mr.id::text, mr.material_id::text, m.name, COALESCE(mr.requester_id::text, ''),
       mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       COALESCE(mr.unit_id::text, ''), COALESCE(mu.unit_code, ''), COALESCE(mu.location, mb.location, ''),
       mr.quantity, mr.purpose, mr.status, mr.created_at
FROM material_requests mr
JOIN materials m ON m.id = mr.material_id
LEFT JOIN material_batches mb ON mb.id = mr.batch_id
LEFT JOIN material_units mu ON mu.id = mr.unit_id
WHERE mr.id = $1
  AND ($2::boolean OR mr.tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID))
}

func scanMaterialRequest(row scanner) (MaterialRequest, error) {
	var item MaterialRequest
	err := row.Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt)
	return item, err
}

func (r *Repository) CreateMaterialRequest(ctx context.Context, input MaterialRequestInput) (MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	input.RequesterID = strings.TrimSpace(input.RequesterID)
	input.Requester = strings.TrimSpace(input.Requester)
	input.BatchID = strings.TrimSpace(input.BatchID)
	input.UnitID = strings.TrimSpace(input.UnitID)
	input.Purpose = strings.TrimSpace(input.Purpose)
	if input.RequesterID == "" {
		input.RequesterID = strings.TrimSpace(tenant.Actor.UserID)
	}
	if input.Requester == "" {
		input.Requester = strings.TrimSpace(tenant.Actor.Name)
	}
	if input.MaterialID == "" || input.Quantity <= 0 || input.Purpose == "" || (input.RequesterID == "" && input.Requester == "") {
		return MaterialRequest{}, clientError("invalid material request input")
	}

	var requesterID, requesterTenantID, requesterName, requesterStatus, groupName string
	var emailVerified bool
	var err error
	if input.RequesterID != "" {
		err = r.db.QueryRow(ctx, `
SELECT id::text, tenant_id::text, name, status, group_name, email_verified
FROM users
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.RequesterID, tenant.AllTenants, tenant.TenantID).Scan(&requesterID, &requesterTenantID, &requesterName, &requesterStatus, &groupName, &emailVerified)
	} else {
		err = r.db.QueryRow(ctx, `
SELECT id::text, tenant_id::text, name, status, group_name, email_verified
FROM users
WHERE name = $1 AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY created_at DESC
LIMIT 1
`, input.Requester, tenant.AllTenants, tenant.TenantID).Scan(&requesterID, &requesterTenantID, &requesterName, &requesterStatus, &groupName, &emailVerified)
	}
	if err != nil {
		return MaterialRequest{}, err
	}
	if requesterStatus != "active" {
		return MaterialRequest{}, clientError("user is not active")
	}
	if !emailVerified {
		return MaterialRequest{}, clientError("email must be verified before requesting materials")
	}
	var availableStock int
	var materialTenantID, materialStatus string
	var expiresAt string
	if err := r.db.QueryRow(ctx, `
SELECT tenant_id::text, stock, status, COALESCE(expires_at::text, '')
FROM materials
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.MaterialID, tenant.AllTenants, tenant.TenantID).Scan(&materialTenantID, &availableStock, &materialStatus, &expiresAt); err != nil {
		return MaterialRequest{}, err
	}
	if requesterTenantID != materialTenantID {
		return MaterialRequest{}, clientError("requester and material must belong to the same tenant")
	}
	if materialStatus == "disabled" {
		return MaterialRequest{}, clientError("material is disabled")
	}
	if expiresAt != "" {
		expireDate, err := time.Parse("2006-01-02", expiresAt)
		if err != nil {
			return MaterialRequest{}, err
		}
		today := appToday()
		if expireDate.Before(today) {
			return MaterialRequest{}, clientError("material is expired")
		}
	}
	batchID := ""
	var batchNo string
	unitID := ""
	var unitCode string
	var unitLocation string
	if input.UnitID == "" {
		return MaterialRequest{}, clientError("material request requires unit")
	}
	if input.Quantity != 1 {
		return MaterialRequest{}, clientError("material unit request quantity must be 1")
	}
	if err := r.db.QueryRow(ctx, `
SELECT mu.id::text, COALESCE(mu.batch_id::text, ''), COALESCE(mb.batch_no, ''), mu.unit_code, COALESCE(mu.location, mb.location, ''), COALESCE(mu.expires_at::text, '')
FROM material_units mu
LEFT JOIN material_batches mb ON mb.id = mu.batch_id
WHERE mu.id = $1
  AND mu.material_id = $2
  AND mu.tenant_id = $3::uuid
  AND mu.status = 'available'
`, input.UnitID, input.MaterialID, materialTenantID).Scan(&unitID, &batchID, &batchNo, &unitCode, &unitLocation, &expiresAt); err != nil {
		return MaterialRequest{}, err
	}
	availableStock = 1
	if expiresAt != "" {
		expireDate, err := time.Parse("2006-01-02", expiresAt)
		if err != nil {
			return MaterialRequest{}, err
		}
		today := appToday()
		if expireDate.Before(today) {
			return MaterialRequest{}, clientError("material unit is expired")
		}
	}
	if availableStock < input.Quantity {
		return MaterialRequest{}, clientError("insufficient material stock")
	}
	requestStatus := "approved"
	if materialApprovalRequired(ctx, r.db, input.MaterialID, materialTenantID) {
		requestStatus = "pending"
	}

	var item MaterialRequest
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialRequest{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	reserveTag, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'reserved', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'available'
`, unitID, input.MaterialID, materialTenantID)
	if err != nil {
		return MaterialRequest{}, err
	}
	if reserveTag.RowsAffected() != 1 {
		return MaterialRequest{}, clientError("material unit is not available")
	}
	if batchID != "" {
		if err := syncMaterialBatchQuantity(ctx, tx, batchID); err != nil {
			return MaterialRequest{}, err
		}
	}
	if _, err := syncMaterialStock(ctx, tx, input.MaterialID, materialTenantID); err != nil {
		return MaterialRequest{}, err
	}
	err = tx.QueryRow(ctx, `
INSERT INTO material_requests (tenant_id, material_id, requester_id, requester, group_name, quantity, purpose, status, decided_at, batch_id, unit_id)
VALUES ($7, $1, $2, $3, $4, $5, $6, $8, CASE WHEN $8 = 'approved' THEN now() ELSE NULL END, NULLIF($9, '')::uuid, NULLIF($11, '')::uuid)
RETURNING id::text, material_id::text, (SELECT name FROM materials WHERE id = material_id),
          COALESCE(requester_id::text, ''), requester, group_name, COALESCE(batch_id::text, ''), $10,
          COALESCE(unit_id::text, ''), $12, $13,
          quantity, purpose, status, created_at
`, input.MaterialID, requesterID, requesterName, groupName, input.Quantity, input.Purpose, materialTenantID, requestStatus, batchID, batchNo, unitID, unitCode, unitLocation).Scan(
		&item.ID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt,
	)
	if err != nil {
		return MaterialRequest{}, err
	}
	notification, err := r.createNotificationTx(ctx, tx, materialTenantID, item.RequesterID, item.GroupName, "", "group", "耗材申领状态更新", fmt.Sprintf("%s 提交了 %s x%d 的申领，当前状态：%s。", item.Requester, item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), "info")
	if err != nil {
		return MaterialRequest{}, err
	}
	notifications = append(notifications, notification)
	if err := tx.Commit(ctx); err != nil {
		return MaterialRequest{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: materialTenantID})
	r.audit(auditCtx, item.Requester, "material.request", "material_request", item.ID, "", item.Status)
	return item, nil
}

func (r *Repository) ApproveMaterialRequest(ctx context.Context, id string, approved bool, actor string, comment string) (MaterialRequest, error) {
	status := "rejected"
	if approved {
		status = "approved"
	}
	return r.updateMaterialRequestStatus(ctx, id, status, actor, comment)
}

func materialApprovalRequired(ctx context.Context, db queryRower, materialID string, tenantID string) bool {
	var approvalRequired bool
	err := db.QueryRow(ctx, `SELECT approval_required FROM materials WHERE id = $1 AND tenant_id = $2::uuid`, materialID, tenantID).Scan(&approvalRequired)
	return err == nil && approvalRequired
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *Repository) OutboundMaterialRequest(ctx context.Context, id string, actor string) (MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialRequest{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 2)

	var item MaterialRequest
	var itemTenantID string
	err = tx.QueryRow(ctx, `
SELECT mr.id::text, mr.tenant_id::text, mr.material_id::text, m.name, COALESCE(mr.requester_id::text, ''),
       mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''),
       COALESCE((SELECT batch_no FROM material_batches WHERE id = mr.batch_id), ''),
       COALESCE(mr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mr.unit_id), ''),
       COALESCE((SELECT location FROM material_units WHERE id = mr.unit_id), (SELECT location FROM material_batches WHERE id = mr.batch_id), ''),
       mr.quantity, mr.purpose, mr.status, mr.created_at
FROM material_requests mr
JOIN materials m ON m.id = mr.material_id
WHERE mr.id = $1 AND mr.status = 'approved'
  AND ($2::boolean OR mr.tenant_id = $3::uuid)
FOR UPDATE OF mr, m
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt)
	if err != nil {
		return MaterialRequest{}, err
	}
	var remainingStock, warningLine int
	var materialUnit string
	if item.UnitID == "" || item.Quantity != 1 {
		return MaterialRequest{}, clientError("material request missing unit")
	}
	if err := tx.QueryRow(ctx, `
SELECT COALESCE(batch_id::text, ''), unit_code, COALESCE(location, '')
FROM material_units
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
FOR UPDATE
`, item.UnitID, item.MaterialID, itemTenantID).Scan(&item.BatchID, &item.UnitCode, &item.Location); err != nil {
		return MaterialRequest{}, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'used', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
		return MaterialRequest{}, err
	}
	if item.BatchID != "" {
		if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
			return MaterialRequest{}, err
		}
		_ = tx.QueryRow(ctx, `SELECT batch_no FROM material_batches WHERE id = $1`, item.BatchID).Scan(&item.BatchNo)
	}
	remainingStock, err = syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID)
	if err != nil {
		return MaterialRequest{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT warning_line, unit FROM materials WHERE id = $1 AND tenant_id = $2::uuid`, item.MaterialID, itemTenantID).Scan(&warningLine, &materialUnit); err != nil {
		return MaterialRequest{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO inventory_ledger (tenant_id, material_id, request_id, change_qty, reason)
VALUES ($1, $2, $3, $4, $5)
`, itemTenantID, item.MaterialID, item.ID, -item.Quantity, materialUnitReason("申领出库", item.BatchNo, item.UnitCode)); err != nil {
		return MaterialRequest{}, err
	}
	if remainingStock <= warningLine {
		created, err := r.createMaterialEventNotificationsTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "耗材库存预警", fmt.Sprintf("%s 出库后库存 %d%s，低于预警线 %d%s。", item.MaterialName, remainingStock, materialUnit, warningLine, materialUnit), "warning")
		if err != nil {
			return MaterialRequest{}, err
		}
		notifications = append(notifications, created...)
	}
	err = tx.QueryRow(ctx, `
UPDATE material_requests mr
SET status = 'outbound'
WHERE mr.id = $1 AND mr.tenant_id = $2::uuid
RETURNING mr.id::text, mr.material_id::text, (SELECT name FROM materials WHERE id = mr.material_id),
          COALESCE(mr.requester_id::text, ''), mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''), $3,
          COALESCE(mr.unit_id::text, ''), $4, $5,
          mr.quantity, mr.purpose, mr.status, mr.created_at
`, id, itemTenantID, item.BatchNo, item.UnitCode, item.Location).Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt)
	if err != nil {
		return MaterialRequest{}, err
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申领状态更新", fmt.Sprintf("%s x%d 的申领状态已更新为%s，储存位置：%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status), firstNonEmpty(item.Location, "未登记")), "success")
		if err != nil {
			return MaterialRequest{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialRequest{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: itemTenantID})
	r.audit(auditCtx, actor, "material.outbound", "material_request", item.ID, "approved", item.Status)
	return item, nil
}

func (r *Repository) CancelMaterialRequest(ctx context.Context, id string, actor string) (MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var item MaterialRequest
	var itemTenantID string
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialRequest{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	err = tx.QueryRow(ctx, `
UPDATE material_requests mr
SET status = 'cancelled'
WHERE mr.id = $1 AND mr.status IN ('pending', 'approved')
  AND ($2::boolean OR mr.tenant_id = $3::uuid)
RETURNING mr.id::text, mr.tenant_id::text, mr.material_id::text, (SELECT name FROM materials WHERE id = mr.material_id),
          COALESCE(mr.requester_id::text, ''), mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''),
          COALESCE((SELECT batch_no FROM material_batches WHERE id = mr.batch_id), ''),
          COALESCE(mr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mr.unit_id), ''),
          COALESCE((SELECT location FROM material_units WHERE id = mr.unit_id), (SELECT location FROM material_batches WHERE id = mr.batch_id), ''),
          mr.quantity, mr.purpose, mr.status, mr.created_at
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt)
	if err != nil {
		return MaterialRequest{}, err
	}
	if item.UnitID != "" {
		if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'available', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
			return MaterialRequest{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
			return MaterialRequest{}, err
		}
		if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
			return MaterialRequest{}, err
		}
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申领状态更新", fmt.Sprintf("%s x%d 的申领状态已更新为%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialRequest{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialRequest{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: itemTenantID})
	r.audit(auditCtx, actor, "material.cancel", "material_request", item.ID, "", item.Status)
	return item, nil
}

func (r *Repository) MaterialPurchases(ctx context.Context) ([]MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
SELECT %s
FROM material_purchases mp
LEFT JOIN materials m ON m.id = mp.material_id
WHERE ($1::boolean OR mp.tenant_id = $2::uuid)
ORDER BY mp.created_at DESC
LIMIT 100
`, materialPurchaseSelectColumns()), tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialPurchase, 0)
	for rows.Next() {
		item, err := scanMaterialPurchase(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) MaterialPurchaseMonthlyConfirmations(ctx context.Context) ([]MaterialPurchaseMonthlyConfirmation, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, month, confirmed_by, confirmed_at
FROM material_purchase_monthly_confirmations
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY month DESC
LIMIT 120
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]MaterialPurchaseMonthlyConfirmation, 0)
	for rows.Next() {
		var item MaterialPurchaseMonthlyConfirmation
		if err := rows.Scan(&item.ID, &item.Month, &item.ConfirmedBy, &item.ConfirmedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ConfirmMaterialPurchaseMonth(ctx context.Context, month string, actor string) (MaterialPurchaseMonthlyConfirmation, error) {
	tenant := TenantFromContext(ctx)
	month = strings.TrimSpace(month)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	if !validMaterialPurchaseMonth(month) {
		return MaterialPurchaseMonthlyConfirmation{}, clientError("invalid material purchase month")
	}
	var item MaterialPurchaseMonthlyConfirmation
	err := r.db.QueryRow(ctx, `
INSERT INTO material_purchase_monthly_confirmations (tenant_id, month, confirmed_by)
VALUES ($1, $2, $3)
ON CONFLICT (tenant_id, month) DO UPDATE
SET confirmed_by = EXCLUDED.confirmed_by,
    confirmed_at = now()
RETURNING id::text, month, confirmed_by, confirmed_at
`, tenant.TenantID, month, actor).Scan(&item.ID, &item.Month, &item.ConfirmedBy, &item.ConfirmedAt)
	if err != nil {
		return MaterialPurchaseMonthlyConfirmation{}, err
	}
	r.audit(ctx, actor, "material_purchase.month_confirm", "material_purchase_month", item.Month, "", item.ConfirmedBy)
	return item, nil
}

func validMaterialPurchaseMonth(month string) bool {
	if len(month) != len("2006-01") {
		return false
	}
	_, err := time.Parse("2006-01", month)
	return err == nil
}

func (r *Repository) MaterialPurchase(ctx context.Context, id string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	return scanMaterialPurchase(r.db.QueryRow(ctx, fmt.Sprintf(`
SELECT %s
FROM material_purchases mp
LEFT JOIN materials m ON m.id = mp.material_id
WHERE mp.id = $1
  AND ($2::boolean OR mp.tenant_id = $3::uuid)
`, materialPurchaseSelectColumns()), id, tenant.AllTenants, tenant.TenantID))
}

func scanMaterialPurchase(row scanner) (MaterialPurchase, error) {
	var item MaterialPurchase
	err := row.Scan(
		&item.ID,
		&item.PurchaseSerialNo,
		&item.MonthlyConfirmed,
		&item.MaterialID,
		&item.MaterialName,
		&item.PurchasableMaterialID,
		&item.PurchaseIDNo,
		&item.PurchaseSequenceNo,
		&item.PurchaseProjectName,
		&item.PurchaseItemName,
		&item.PurchaseBrand,
		&item.PurchaseSpec,
		&item.PurchaseUnit,
		&item.PurchaseRemark,
		&item.PurchaseTechnicalRequirement,
		&item.PurchaseMinSpec,
		&item.RequesterID,
		&item.Requester,
		&item.RequesterPhone,
		&item.RequesterEmail,
		&item.GroupName,
		&item.Quantity,
		&item.EstimatedUnitPrice,
		&item.Supplier,
		&item.Reason,
		&item.Status,
		&item.CreatedAt,
	)
	return item, err
}

func materialPurchaseSelectColumns() string {
	return `mp.id::text,
	       COALESCE(mp.purchase_serial_no, ''),
	       EXISTS (
	           SELECT 1
	           FROM material_purchase_monthly_confirmations mpmc
	           WHERE mpmc.tenant_id = mp.tenant_id
	             AND mpmc.month = to_char(mp.created_at, 'YYYY-MM')
	       ) AS monthly_confirmed,
	       COALESCE(mp.material_id::text, ''),
       COALESCE(NULLIF(mp.purchase_item_name, ''), NULLIF(mp.purchase_project_name, ''), m.name, ''),
       COALESCE(mp.purchasable_material_id::text, ''),
       mp.purchase_id_no,
       mp.purchase_sequence_no,
       COALESCE(NULLIF(mp.purchase_project_name, ''), m.name, ''),
       COALESCE(NULLIF(mp.purchase_item_name, ''), NULLIF(mp.purchase_project_name, ''), m.name, ''),
       COALESCE(NULLIF(mp.purchase_brand, ''), m.manufacturer, ''),
       COALESCE(NULLIF(mp.purchase_spec, ''), m.spec, ''),
       COALESCE(NULLIF(mp.purchase_unit, ''), m.unit, ''),
       mp.purchase_remark,
       mp.purchase_technical_requirement,
       mp.purchase_min_spec,
       COALESCE(mp.requester_id::text, ''),
       mp.requester,
       COALESCE(mp.requester_phone, ''),
       COALESCE(mp.requester_email, ''),
       mp.group_name,
       mp.quantity,
       mp.estimated_unit_price::float8,
       mp.supplier,
       mp.reason,
       mp.status,
       mp.created_at`
}

func (r *Repository) materialPurchaseBySerial(ctx context.Context, serial string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return MaterialPurchase{}, clientError("material purchase serial no is required")
	}
	item, err := scanMaterialPurchase(r.db.QueryRow(ctx, fmt.Sprintf(`
SELECT %s
FROM material_purchases mp
LEFT JOIN materials m ON m.id = mp.material_id
WHERE mp.tenant_id = $2::uuid
  AND (
      mp.id::text = $1
      OR lower(mp.purchase_serial_no) = lower($1)
      OR lower(mp.purchase_id_no) = lower($1)
      OR lower(mp.purchase_sequence_no) = lower($1)
  )
ORDER BY mp.created_at DESC
LIMIT 1
`, materialPurchaseSelectColumns()), serial, tenant.TenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MaterialPurchase{}, clientError("material purchase serial no not found")
		}
		return MaterialPurchase{}, err
	}
	return item, nil
}

func canManageMaterialsRole(role string) bool {
	switch role {
	case "material_admin", "tenant_admin", "lab_admin", "super_admin":
		return true
	default:
		return false
	}
}

func (r *Repository) applyMaterialPurchaseToMaterialInput(ctx context.Context, input MaterialInput, purchase MaterialPurchase) MaterialInput {
	if purchase.ID == "" {
		return normalizeMaterial(input)
	}
	if input.Name == "" {
		input.Name = firstNonEmpty(purchase.PurchaseItemName, purchase.MaterialName, purchase.PurchaseProjectName)
	}
	input.Spec = firstNonEmpty(input.Spec, purchase.PurchaseSpec)
	input.Unit = firstNonEmpty(input.Unit, purchase.PurchaseUnit)
	if input.UnitPrice <= 0 {
		input.UnitPrice = purchase.EstimatedUnitPrice
	}
	if input.Stock <= 0 && purchase.Quantity > 0 {
		input.Stock = purchase.Quantity
	}
	input.Supplier = firstNonEmpty(input.Supplier, purchase.Supplier, purchase.PurchaseBrand)
	input.Manufacturer = firstNonEmpty(input.Manufacturer, purchase.PurchaseBrand)
	input.CatalogNo = firstNonEmpty(input.CatalogNo, purchase.PurchaseIDNo)
	input.TenderContract = firstNonEmpty(input.TenderContract, purchase.PurchaseProjectName)
	input.ContractNo = firstNonEmpty(input.ContractNo, purchase.PurchaseProjectName)
	input.Remark = firstNonEmpty(input.Remark, purchase.PurchaseRemark)
	if input.BatchNo == "" {
		input.BatchNo = firstNonEmpty(purchase.PurchaseSequenceNo, purchase.PurchaseIDNo)
	}
	return normalizeMaterial(input)
}

func (r *Repository) CreateMaterialPurchase(ctx context.Context, input MaterialPurchaseInput) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	input.MaterialID = strings.TrimSpace(input.MaterialID)
	input.PurchasableMaterialID = strings.TrimSpace(input.PurchasableMaterialID)
	input.PurchaseSerialNo = strings.TrimSpace(input.PurchaseSerialNo)
	input.RequesterID = strings.TrimSpace(input.RequesterID)
	input.Requester = strings.TrimSpace(input.Requester)
	input.Supplier = strings.TrimSpace(input.Supplier)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.PurchaseSerialNo != "" {
		purchase, err := r.materialPurchaseBySerial(ctx, input.PurchaseSerialNo)
		if err != nil {
			return MaterialPurchase{}, err
		}
		if input.PurchasableMaterialID == "" {
			input.PurchasableMaterialID = purchase.PurchasableMaterialID
		}
		if input.MaterialID == "" {
			input.MaterialID = purchase.MaterialID
		}
		if input.RequesterID == "" {
			input.RequesterID = purchase.RequesterID
		}
		if input.Requester == "" {
			input.Requester = purchase.Requester
		}
		if input.EstimatedUnitPrice == 0 {
			input.EstimatedUnitPrice = purchase.EstimatedUnitPrice
		}
		if input.Supplier == "" {
			input.Supplier = purchase.Supplier
		}
	}
	if input.RequesterID == "" {
		input.RequesterID = strings.TrimSpace(tenant.Actor.UserID)
	}
	if input.Requester == "" {
		input.Requester = strings.TrimSpace(tenant.Actor.Name)
	}
	if (input.MaterialID == "" && input.PurchasableMaterialID == "") || input.Quantity <= 0 || input.EstimatedUnitPrice < 0 || input.Reason == "" || (input.RequesterID == "" && input.Requester == "") {
		return MaterialPurchase{}, clientError("invalid material purchase input")
	}

	var requesterID, requesterTenantID, requesterName, requesterStatus, requesterPhone, requesterEmail, groupName string
	var emailVerified bool
	var err error
	if input.RequesterID != "" {
		err = r.db.QueryRow(ctx, `
SELECT id::text, tenant_id::text, name, status, phone, email, group_name, email_verified
FROM users
WHERE id = $1 AND status <> 'deleted' AND ($2::boolean OR tenant_id = $3::uuid)
`, input.RequesterID, tenant.AllTenants, tenant.TenantID).Scan(&requesterID, &requesterTenantID, &requesterName, &requesterStatus, &requesterPhone, &requesterEmail, &groupName, &emailVerified)
	} else {
		err = r.db.QueryRow(ctx, `
SELECT id::text, tenant_id::text, name, status, phone, email, group_name, email_verified
FROM users
WHERE name = $1 AND status <> 'deleted' AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY created_at DESC
LIMIT 1
`, input.Requester, tenant.AllTenants, tenant.TenantID).Scan(&requesterID, &requesterTenantID, &requesterName, &requesterStatus, &requesterPhone, &requesterEmail, &groupName, &emailVerified)
	}
	if err != nil {
		return MaterialPurchase{}, err
	}
	if requesterStatus != "active" {
		return MaterialPurchase{}, clientError("user is not active")
	}
	if !emailVerified {
		return MaterialPurchase{}, clientError("email must be verified before purchasing materials")
	}

	var item MaterialPurchase
	var materialTenantID, materialStatus, defaultSupplier string
	if input.PurchasableMaterialID != "" {
		var purchasable PurchasableMaterial
		err := r.db.QueryRow(ctx, `
SELECT pm.id::text, pm.id_no, pm.sequence_no, COALESCE(pm.procurement_project_id::text, ''),
       COALESCE(pp.name, pm.procurement_project), COALESCE(pp.expires_at::text, ''),
       COALESCE(pp.status, 'active'),
       pm.project_name, pm.brand, pm.spec, pm.unit, pm.purchase_price::float8,
       pm.remark, pm.technical_requirement, pm.min_spec, pm.status, pm.created_at, pm.updated_at
FROM purchasable_materials pm
LEFT JOIN procurement_projects pp ON pp.id = pm.procurement_project_id
WHERE pm.id = $1
  AND pm.status = 'active'
  AND (pp.id IS NULL OR pp.status = 'active')
  AND (pp.expires_at IS NULL OR pp.expires_at >= `+appDateSQL()+`)
  AND ($2::boolean OR pm.tenant_id = $3::uuid)
`, input.PurchasableMaterialID, tenant.AllTenants, tenant.TenantID).Scan(
			&purchasable.ID, &purchasable.IDNo, &purchasable.SequenceNo, &purchasable.ProcurementProjectID, &purchasable.ProcurementProject, &purchasable.ProcurementExpiresAt, &purchasable.ProcurementProjectStatus, &purchasable.ProjectName, &purchasable.Brand, &purchasable.Spec, &purchasable.Unit, &purchasable.PurchasePrice, &purchasable.Remark, &purchasable.TechnicalRequirement, &purchasable.MinSpec, &purchasable.Status, &purchasable.CreatedAt, &purchasable.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return MaterialPurchase{}, clientError("material procurement project expired or unavailable")
			}
			return MaterialPurchase{}, err
		}
		materialTenantID = tenant.TenantID
		if requesterTenantID != materialTenantID {
			return MaterialPurchase{}, clientError("requester and purchasable material must belong to the same tenant")
		}
		item.PurchasableMaterialID = purchasable.ID
		item.PurchaseIDNo = purchasable.IDNo
		item.PurchaseSequenceNo = purchasable.SequenceNo
		item.PurchaseProjectName = firstNonEmpty(purchasable.ProcurementProject, purchasable.ProjectName)
		item.PurchaseItemName = purchasable.ProjectName
		item.PurchaseBrand = purchasable.Brand
		item.PurchaseSpec = purchasable.Spec
		item.PurchaseUnit = purchasable.Unit
		item.PurchaseRemark = purchasable.Remark
		item.PurchaseTechnicalRequirement = purchasable.TechnicalRequirement
		item.PurchaseMinSpec = purchasable.MinSpec
		if input.EstimatedUnitPrice == 0 {
			input.EstimatedUnitPrice = purchasable.PurchasePrice
		}
	} else {
		if err := r.db.QueryRow(ctx, `
SELECT tenant_id::text, status, supplier, name, name, manufacturer, spec, unit
FROM materials
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.MaterialID, tenant.AllTenants, tenant.TenantID).Scan(&materialTenantID, &materialStatus, &defaultSupplier, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit); err != nil {
			return MaterialPurchase{}, err
		}
		if requesterTenantID != materialTenantID {
			return MaterialPurchase{}, clientError("requester and material must belong to the same tenant")
		}
		if materialStatus == "disabled" {
			return MaterialPurchase{}, clientError("material is disabled")
		}
		if input.Supplier == "" {
			input.Supplier = defaultSupplier
		}
	}

	notifications := make([]Notification, 0, 1)
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	serialNo, err := r.nextMaterialPurchaseSerialNo(ctx, tx, materialTenantID)
	if err != nil {
		return MaterialPurchase{}, err
	}
	err = tx.QueryRow(ctx, fmt.Sprintf(`
WITH inserted AS (
  INSERT INTO material_purchases (
    tenant_id, purchase_serial_no, material_id, purchasable_material_id,
    purchase_id_no, purchase_sequence_no, purchase_project_name, purchase_item_name, purchase_brand, purchase_spec, purchase_unit,
    purchase_remark, purchase_technical_requirement, purchase_min_spec,
    requester_id, requester, requester_phone, requester_email, group_name, quantity, estimated_unit_price, supplier, reason, status, decided_at
  )
  VALUES ($1, $23, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, 'registered', now())
  RETURNING *
)
SELECT %s
FROM inserted mp
LEFT JOIN materials m ON m.id = mp.material_id
`, materialPurchaseSelectColumns()), materialTenantID, input.MaterialID, item.PurchasableMaterialID, item.PurchaseIDNo, item.PurchaseSequenceNo, item.PurchaseProjectName, item.PurchaseItemName, item.PurchaseBrand, item.PurchaseSpec, item.PurchaseUnit, item.PurchaseRemark, item.PurchaseTechnicalRequirement, item.PurchaseMinSpec, requesterID, requesterName, requesterPhone, requesterEmail, groupName, input.Quantity, input.EstimatedUnitPrice, input.Supplier, input.Reason, serialNo).Scan(
		&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt,
	)
	if err != nil {
		return MaterialPurchase{}, err
	}
	notification, err := r.createNotificationTx(ctx, tx, materialTenantID, item.RequesterID, item.GroupName, "", "group", "耗材申购登记", fmt.Sprintf("%s 登记了 %s x%d 的申购，申购流水号：%s，当前状态：%s。", item.Requester, item.MaterialName, item.Quantity, item.PurchaseSerialNo, materialWorkflowStatusLabel(item.Status)), "info")
	if err != nil {
		return MaterialPurchase{}, err
	}
	notifications = append(notifications, notification)
	if err := r.auditTx(ctx, tx, materialTenantID, item.Requester, "material_purchase.create", "material_purchase", item.ID, "", item.Status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) nextMaterialPurchaseSerialNo(ctx context.Context, tx pgx.Tx, tenantID string) (string, error) {
	month := appNow().Format("200601")
	prefix := "SG" + month + "-"
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, tenantID+":"+prefix); err != nil {
		return "", err
	}
	var maxIndex int
	if err := tx.QueryRow(ctx, `
SELECT COALESCE(MAX(NULLIF(regexp_replace(purchase_serial_no, '^SG[0-9]{6}-', ''), '')::int), 0)
FROM material_purchases
WHERE tenant_id = $1::uuid
  AND purchase_serial_no LIKE $2
  AND purchase_serial_no ~ '^SG[0-9]{6}-[0-9]{4}$'
`, tenantID, prefix+"%").Scan(&maxIndex); err != nil {
		return "", err
	}
	nextIndex := maxIndex + 1
	if nextIndex > 9999 {
		return "", clientError("material purchase serial no exhausted")
	}
	return fmt.Sprintf("%s%04d", prefix, nextIndex), nil
}

func (r *Repository) ApproveMaterialPurchase(ctx context.Context, id string, approved bool, actor string, comment string) (MaterialPurchase, error) {
	status := "rejected"
	if approved {
		status = "approved"
	}
	return r.updateMaterialPurchaseStatus(ctx, id, status, actor, comment)
}

func (r *Repository) ReturnMaterialPurchase(ctx context.Context, id string, actor string, comment string) (MaterialPurchase, error) {
	return r.updateMaterialPurchaseStatus(ctx, id, "returned", actor, comment)
}

func (r *Repository) UpdateMaterialPurchase(ctx context.Context, id string, input MaterialPurchaseUpdateInput) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	id = strings.TrimSpace(id)
	input.PurchasableMaterialID = strings.TrimSpace(input.PurchasableMaterialID)
	input.Supplier = strings.TrimSpace(input.Supplier)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = firstNonEmpty(tenant.Actor.Name, "system")
	}
	if id == "" || input.PurchasableMaterialID == "" || input.Quantity <= 0 || input.EstimatedUnitPrice < 0 || input.Reason == "" {
		return MaterialPurchase{}, clientError("invalid material purchase update input")
	}
	var purchasable PurchasableMaterial
	if err := r.db.QueryRow(ctx, `
SELECT pm.id::text, pm.id_no, pm.sequence_no, COALESCE(pm.procurement_project_id::text, ''),
       COALESCE(pp.name, pm.procurement_project), COALESCE(pp.expires_at::text, ''),
       COALESCE(pp.status, 'active'),
       pm.project_name, pm.brand, pm.spec, pm.unit, pm.purchase_price::float8,
       pm.remark, pm.technical_requirement, pm.min_spec, pm.status, pm.created_at, pm.updated_at
FROM purchasable_materials pm
LEFT JOIN procurement_projects pp ON pp.id = pm.procurement_project_id
WHERE pm.id = $1
  AND pm.status = 'active'
  AND (pp.id IS NULL OR pp.status = 'active')
  AND (pp.expires_at IS NULL OR pp.expires_at >= `+appDateSQL()+`)
  AND ($2::boolean OR pm.tenant_id = $3::uuid)
`, input.PurchasableMaterialID, tenant.AllTenants, tenant.TenantID).Scan(
		&purchasable.ID, &purchasable.IDNo, &purchasable.SequenceNo, &purchasable.ProcurementProjectID, &purchasable.ProcurementProject, &purchasable.ProcurementExpiresAt, &purchasable.ProcurementProjectStatus, &purchasable.ProjectName, &purchasable.Brand, &purchasable.Spec, &purchasable.Unit, &purchasable.PurchasePrice, &purchasable.Remark, &purchasable.TechnicalRequirement, &purchasable.MinSpec, &purchasable.Status, &purchasable.CreatedAt, &purchasable.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MaterialPurchase{}, clientError("material procurement project expired or unavailable")
		}
		return MaterialPurchase{}, err
	}
	if input.EstimatedUnitPrice == 0 {
		input.EstimatedUnitPrice = purchasable.PurchasePrice
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var item MaterialPurchase
	var itemTenantID string
	err = tx.QueryRow(ctx, fmt.Sprintf(`
WITH updated AS (
  UPDATE material_purchases
  SET purchasable_material_id = $2,
      purchase_id_no = $3,
      purchase_sequence_no = $4,
      purchase_project_name = $5,
      purchase_item_name = $6,
      purchase_brand = $7,
      purchase_spec = $8,
      purchase_unit = $9,
      purchase_remark = $10,
      purchase_technical_requirement = $11,
      purchase_min_spec = $12,
      quantity = $13,
      estimated_unit_price = $14,
      supplier = $15,
      reason = $16,
      status = 'registered',
      decided_at = now()
  WHERE id = $1 AND status = 'returned'
    AND ($17::boolean OR tenant_id = $18::uuid)
    AND (requester_id::text = $19 OR $20::boolean)
    AND NOT EXISTS (
        SELECT 1
        FROM material_purchase_monthly_confirmations mpmc
        WHERE mpmc.tenant_id = material_purchases.tenant_id
          AND mpmc.month = to_char(material_purchases.created_at, 'YYYY-MM')
    )
  RETURNING *
)
SELECT %s, mp.tenant_id::text
FROM updated mp
LEFT JOIN materials m ON m.id = mp.material_id
`, materialPurchaseSelectColumns()), id, purchasable.ID, purchasable.IDNo, purchasable.SequenceNo, firstNonEmpty(purchasable.ProcurementProject, purchasable.ProjectName), purchasable.ProjectName, purchasable.Brand, purchasable.Spec, purchasable.Unit, purchasable.Remark, purchasable.TechnicalRequirement, purchasable.MinSpec, input.Quantity, input.EstimatedUnitPrice, input.Supplier, input.Reason, tenant.AllTenants, tenant.TenantID, tenant.Actor.UserID, canManageMaterialsRole(tenant.Actor.Role)).Scan(
		&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt, &itemTenantID,
	)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, 'resubmit', '申请人修改后重新提交')
`, itemTenantID, item.ID, input.Actor); err != nil {
		return MaterialPurchase{}, err
	}
	if err := r.auditTx(ctx, tx, itemTenantID, input.Actor, "material_purchase.resubmit", "material_purchase", item.ID, "returned", item.Status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	return item, nil
}

func (r *Repository) MarkMaterialPurchaseOrdered(ctx context.Context, id string, actor string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	var item MaterialPurchase
	var itemTenantID string
	err = tx.QueryRow(ctx, fmt.Sprintf(`
WITH updated AS (
  UPDATE material_purchases
  SET status = 'ordered', ordered_at = now()
  WHERE id = $1 AND status IN ('registered', 'approved')
    AND ($2::boolean OR tenant_id = $3::uuid)
  RETURNING *
)
SELECT %s, mp.tenant_id::text
FROM updated mp
LEFT JOIN materials m ON m.id = mp.material_id
	`, materialPurchaseSelectColumns()), id, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt, &itemTenantID,
	)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, 'order', '已下单')
`, itemTenantID, item.ID, actor); err != nil {
		return MaterialPurchase{}, err
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申购状态更新", fmt.Sprintf("%s x%d 的申购状态已更新为%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialPurchase{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_purchase.order", "material_purchase", item.ID, "", item.Status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) ReceiveMaterialPurchase(ctx context.Context, id string, actor string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)

	var item MaterialPurchase
	var itemTenantID string
	var productType, defaultBatchNo, defaultExpiresAt, defaultLocation string
	err = tx.QueryRow(ctx, `
SELECT mp.id::text, mp.tenant_id::text, mp.material_id::text, m.name, COALESCE(mp.requester_id::text, ''),
       mp.requester, COALESCE(mp.requester_phone, ''), COALESCE(mp.requester_email, ''), mp.group_name, mp.quantity, mp.estimated_unit_price::float8,
       mp.supplier, mp.reason, mp.status, mp.created_at, m.product_type, m.batch_no, COALESCE(m.expires_at::text, ''),
       concat_ws(' / ', NULLIF(m.storage_room, ''), NULLIF(m.storage_cabinet, ''), NULLIF(m.storage_layer, ''), NULLIF(m.storage_slot, ''))
FROM material_purchases mp
JOIN materials m ON m.id = mp.material_id
WHERE mp.id = $1 AND mp.status IN ('registered', 'approved', 'ordered')
  AND ($2::boolean OR mp.tenant_id = $3::uuid)
FOR UPDATE
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt, &productType, &defaultBatchNo, &defaultExpiresAt, &defaultLocation)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if item.MaterialID == "" {
		return MaterialPurchase{}, clientError("material purchase has no inventory material to receive")
	}
	oldStatus := item.Status
	if productType == "standard" {
		if defaultBatchNo == "" {
			defaultBatchNo = "默认批次"
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO material_batches (tenant_id, material_id, batch_no, quantity, expires_at, location, status)
VALUES ($1, $2, $3, $4, NULLIF($5, '')::date, $6, 'active')
ON CONFLICT (tenant_id, material_id, batch_no) DO UPDATE
SET quantity = material_batches.quantity + EXCLUDED.quantity,
    expires_at = COALESCE(EXCLUDED.expires_at, material_batches.expires_at),
    location = COALESCE(NULLIF(EXCLUDED.location, ''), material_batches.location),
    status = 'active',
    updated_at = now()
`, itemTenantID, item.MaterialID, defaultBatchNo, item.Quantity, defaultExpiresAt, defaultLocation); err != nil {
			return MaterialPurchase{}, err
		}
		var batchID string
		if err := tx.QueryRow(ctx, `
SELECT id::text
FROM material_batches
WHERE tenant_id = $1::uuid AND material_id = $2 AND batch_no = $3
`, itemTenantID, item.MaterialID, defaultBatchNo).Scan(&batchID); err != nil {
			return MaterialPurchase{}, err
		}
		if err := createMaterialUnits(ctx, tx, MaterialUnitGenerationInput{
			TenantID:     itemTenantID,
			MaterialID:   item.MaterialID,
			MaterialName: item.MaterialName,
			BatchID:      batchID,
			Quantity:     item.Quantity,
			ExpiresAt:    defaultExpiresAt,
			Location:     defaultLocation,
		}); err != nil {
			return MaterialPurchase{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, batchID); err != nil {
			return MaterialPurchase{}, err
		}
		if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
			return MaterialPurchase{}, err
		}
	} else {
		if _, err := tx.Exec(ctx, `UPDATE materials SET stock = stock + $2 WHERE id = $1 AND tenant_id = $3::uuid`, item.MaterialID, item.Quantity, itemTenantID); err != nil {
			return MaterialPurchase{}, err
		}
		if err := createMaterialUnits(ctx, tx, MaterialUnitGenerationInput{
			TenantID:     itemTenantID,
			MaterialID:   item.MaterialID,
			MaterialName: item.MaterialName,
			Quantity:     item.Quantity,
			ExpiresAt:    defaultExpiresAt,
			Location:     defaultLocation,
		}); err != nil {
			return MaterialPurchase{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO inventory_ledger (tenant_id, material_id, purchase_id, change_qty, reason)
VALUES ($1, $2, $3, $4, $5)
`, itemTenantID, item.MaterialID, item.ID, item.Quantity, materialBatchReason("申购到货入库", defaultBatchNo)); err != nil {
		return MaterialPurchase{}, err
	}
	err = tx.QueryRow(ctx, fmt.Sprintf(`
WITH updated AS (
  UPDATE material_purchases
  SET status = 'received', received_at = now()
  WHERE id = $1 AND tenant_id = $2::uuid
  RETURNING *
)
SELECT %s
FROM updated mp
LEFT JOIN materials m ON m.id = mp.material_id
	`, materialPurchaseSelectColumns()), id, itemTenantID).Scan(&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, 'receive', '到货入库')
`, itemTenantID, item.ID, actor); err != nil {
		return MaterialPurchase{}, err
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申购状态更新", fmt.Sprintf("%s x%d 的申购状态已更新为%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialPurchase{}, err
		}
		notifications = append(notifications, notification)
	}
	created, err := r.createMaterialEventNotificationsTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "耗材申购完成入库", fmt.Sprintf("%s x%d 已完成入库，储存位置：%s。", item.MaterialName, item.Quantity, firstNonEmpty(defaultLocation, "未登记")), "success")
	if err != nil {
		return MaterialPurchase{}, err
	}
	notifications = append(notifications, created...)
	if materialNearExpiry(defaultExpiresAt, 30) {
		created, err := r.createMaterialEventNotificationsTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "耗材有效期告警", fmt.Sprintf("%s 有效期为 %s，已进入临期预警范围。", item.MaterialName, defaultExpiresAt), "warning")
		if err != nil {
			return MaterialPurchase{}, err
		}
		notifications = append(notifications, created...)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_purchase.receive", "material_purchase", item.ID, oldStatus, item.Status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) CancelMaterialPurchase(ctx context.Context, id string, actor string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	var item MaterialPurchase
	var itemTenantID string
	err = tx.QueryRow(ctx, fmt.Sprintf(`
WITH updated AS (
  UPDATE material_purchases
  SET status = 'cancelled'
  WHERE id = $1 AND status IN ('registered', 'approved', 'returned', 'ordered')
    AND ($2::boolean OR tenant_id = $3::uuid)
    AND NOT EXISTS (
        SELECT 1
        FROM material_purchase_monthly_confirmations mpmc
        WHERE mpmc.tenant_id = material_purchases.tenant_id
          AND mpmc.month = to_char(material_purchases.created_at, 'YYYY-MM')
    )
  RETURNING *
)
SELECT %s, mp.tenant_id::text
FROM updated mp
LEFT JOIN materials m ON m.id = mp.material_id
	`, materialPurchaseSelectColumns()), id, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt, &itemTenantID,
	)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, 'cancel', '已取消')
`, itemTenantID, item.ID, actor); err != nil {
		return MaterialPurchase{}, err
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申购状态更新", fmt.Sprintf("%s x%d 的申购状态已更新为%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialPurchase{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_purchase.cancel", "material_purchase", item.ID, "", item.Status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) MaterialDamages(ctx context.Context) ([]MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT mdr.id::text, mdr.material_id::text, m.name, COALESCE(mdr.reporter_id::text, ''),
       mdr.reporter, mdr.group_name, COALESCE(mdr.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       COALESCE(mdr.unit_id::text, ''), COALESCE(mu.unit_code, ''),
       mdr.quantity, mdr.reason, mdr.photo_url, mdr.attachment_url,
       mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
       COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
       COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
FROM material_damage_reports mdr
JOIN materials m ON m.id = mdr.material_id
LEFT JOIN material_batches mb ON mb.id = mdr.batch_id
LEFT JOIN material_units mu ON mu.id = mdr.unit_id
WHERE ($1::boolean OR mdr.tenant_id = $2::uuid)
ORDER BY mdr.created_at DESC
LIMIT 200
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialDamage, 0)
	for rows.Next() {
		item, err := scanMaterialDamage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) MaterialDamage(ctx context.Context, id string) (MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	return scanMaterialDamage(r.db.QueryRow(ctx, `
SELECT mdr.id::text, mdr.material_id::text, m.name, COALESCE(mdr.reporter_id::text, ''),
       mdr.reporter, mdr.group_name, COALESCE(mdr.batch_id::text, ''), COALESCE(mb.batch_no, ''),
       COALESCE(mdr.unit_id::text, ''), COALESCE(mu.unit_code, ''),
       mdr.quantity, mdr.reason, mdr.photo_url, mdr.attachment_url,
       mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
       COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
       COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
FROM material_damage_reports mdr
JOIN materials m ON m.id = mdr.material_id
LEFT JOIN material_batches mb ON mb.id = mdr.batch_id
LEFT JOIN material_units mu ON mu.id = mdr.unit_id
WHERE mdr.id = $1
  AND ($2::boolean OR mdr.tenant_id = $3::uuid)
`, id, tenant.AllTenants, tenant.TenantID))
}

func scanMaterialDamage(row scanner) (MaterialDamage, error) {
	var item MaterialDamage
	err := row.Scan(
		&item.ID,
		&item.MaterialID,
		&item.MaterialName,
		&item.ReporterID,
		&item.Reporter,
		&item.GroupName,
		&item.BatchID,
		&item.BatchNo,
		&item.UnitID,
		&item.UnitCode,
		&item.Quantity,
		&item.Reason,
		&item.PhotoURL,
		&item.AttachmentURL,
		&item.Status,
		&item.Reviewer,
		&item.ReviewComment,
		&item.CreatedAt,
		&item.ReviewedAt,
		&item.ProcessedAt,
	)
	return item, err
}

func (r *Repository) CreateMaterialDamage(ctx context.Context, input MaterialDamageInput) (MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	input.MaterialID = strings.TrimSpace(input.MaterialID)
	input.ReporterID = strings.TrimSpace(input.ReporterID)
	input.Reporter = strings.TrimSpace(input.Reporter)
	input.UnitID = strings.TrimSpace(input.UnitID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.PhotoURL = strings.TrimSpace(input.PhotoURL)
	input.AttachmentURL = strings.TrimSpace(input.AttachmentURL)
	if input.ReporterID == "" {
		input.ReporterID = strings.TrimSpace(tenant.Actor.UserID)
	}
	if input.Reporter == "" {
		input.Reporter = strings.TrimSpace(tenant.Actor.Name)
	}
	if input.MaterialID == "" || input.UnitID == "" || input.Quantity != 1 || input.Reason == "" || (input.ReporterID == "" && input.Reporter == "") {
		return MaterialDamage{}, clientError("invalid material damage input")
	}

	reporterID, reporterTenantID, reporterName, reporterStatus, groupName, _, err := r.resolveMaterialActor(ctx, input.ReporterID, input.Reporter, tenant)
	if err != nil {
		return MaterialDamage{}, err
	}
	if reporterStatus != "active" {
		return MaterialDamage{}, clientError("user is not active")
	}

	var materialTenantID, materialStatus string
	if err := r.db.QueryRow(ctx, `
SELECT tenant_id::text, status
FROM materials
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.MaterialID, tenant.AllTenants, tenant.TenantID).Scan(&materialTenantID, &materialStatus); err != nil {
		return MaterialDamage{}, err
	}
	if reporterTenantID != materialTenantID {
		return MaterialDamage{}, clientError("reporter and material must belong to the same tenant")
	}
	if materialStatus == "disabled" {
		return MaterialDamage{}, clientError("material is disabled")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialDamage{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	var batchID, batchNo, unitID, unitCode string
	if err := tx.QueryRow(ctx, `
SELECT mu.id::text, COALESCE(mu.batch_id::text, ''), COALESCE(mb.batch_no, ''), mu.unit_code
FROM material_units mu
LEFT JOIN material_batches mb ON mb.id = mu.batch_id
WHERE mu.id = $1
  AND mu.material_id = $2
  AND mu.tenant_id = $3::uuid
  AND mu.status = 'available'
FOR UPDATE
`, input.UnitID, input.MaterialID, materialTenantID).Scan(&unitID, &batchID, &batchNo, &unitCode); err != nil {
		return MaterialDamage{}, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'reserved', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'available'
`, unitID, input.MaterialID, materialTenantID); err != nil {
		return MaterialDamage{}, err
	}
	if batchID != "" {
		if err := syncMaterialBatchQuantity(ctx, tx, batchID); err != nil {
			return MaterialDamage{}, err
		}
	}
	if _, err := syncMaterialStock(ctx, tx, input.MaterialID, materialTenantID); err != nil {
		return MaterialDamage{}, err
	}
	item, err := scanMaterialDamage(tx.QueryRow(ctx, `
INSERT INTO material_damage_reports (tenant_id, material_id, reporter_id, reporter, group_name, batch_id, unit_id, quantity, reason, photo_url, attachment_url)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::uuid, $7, $8, $9, $10, $11)
RETURNING id::text, material_id::text, (SELECT name FROM materials WHERE id = material_id),
          COALESCE(reporter_id::text, ''), reporter, group_name, COALESCE(batch_id::text, ''), $12,
          COALESCE(unit_id::text, ''), $13,
          quantity, reason, photo_url, attachment_url,
          status, reviewer, review_comment, created_at,
          COALESCE(reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
          COALESCE(processed_at, '0001-01-01 00:00:00+00'::timestamptz)
`, materialTenantID, input.MaterialID, optionalID(reporterID), reporterName, groupName, batchID, unitID, input.Quantity, input.Reason, input.PhotoURL, input.AttachmentURL, batchNo, unitCode))
	if err != nil {
		return MaterialDamage{}, err
	}
	notification, err := r.createNotificationTx(ctx, tx, materialTenantID, item.ReporterID, item.GroupName, "", "group", "损毁登记状态更新", fmt.Sprintf("%s 登记了 %s 编号 %s 的损毁，当前状态：%s。", item.Reporter, item.MaterialName, item.UnitCode, materialWorkflowStatusLabel(item.Status)), "warning")
	if err != nil {
		return MaterialDamage{}, err
	}
	notifications = append(notifications, notification)
	if err := tx.Commit(ctx); err != nil {
		return MaterialDamage{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: materialTenantID})
	r.audit(auditCtx, item.Reporter, "material_damage.create", "material_damage", item.ID, "", item.Status)
	return item, nil
}

func (r *Repository) ApproveMaterialDamage(ctx context.Context, id string, approved bool, actor string, comment string) (MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	comment = strings.TrimSpace(comment)
	if actor == "" {
		actor = "system"
	}
	status := "rejected"
	if approved {
		status = "approved"
	}
	if comment == "" {
		comment = status
	}
	var itemTenantID string
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialDamage{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	item, err := scanMaterialDamage(tx.QueryRow(ctx, `
UPDATE material_damage_reports mdr
SET status = $2, reviewer = $3, review_comment = $4, reviewed_at = now()
WHERE mdr.id = $1 AND mdr.status = 'pending'
  AND ($5::boolean OR mdr.tenant_id = $6::uuid)
RETURNING mdr.id::text, mdr.material_id::text, (SELECT name FROM materials WHERE id = mdr.material_id),
          COALESCE(mdr.reporter_id::text, ''), mdr.reporter, mdr.group_name,
          COALESCE(mdr.batch_id::text, ''), COALESCE((SELECT batch_no FROM material_batches WHERE id = mdr.batch_id), ''),
          COALESCE(mdr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mdr.unit_id), ''),
          mdr.quantity, mdr.reason,
          mdr.photo_url, mdr.attachment_url, mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
          COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
          COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
`, id, status, actor, comment, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return MaterialDamage{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT tenant_id::text FROM material_damage_reports WHERE id = $1`, item.ID).Scan(&itemTenantID); err != nil {
		return MaterialDamage{}, err
	}
	if status == "rejected" && item.UnitID != "" {
		if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'available', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
			return MaterialDamage{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
			return MaterialDamage{}, err
		}
		if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
			return MaterialDamage{}, err
		}
	}
	if item.ReporterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.ReporterID, item.GroupName, "", "personal", "损毁登记状态更新", fmt.Sprintf("%s 编号 %s 的损毁登记状态已更新为%s。", item.MaterialName, item.UnitCode, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialDamage{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialDamage{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: itemTenantID})
	r.audit(auditCtx, actor, "material_damage."+status, "material_damage", item.ID, "pending", status)
	return item, nil
}

func (r *Repository) ProcessMaterialDamage(ctx context.Context, id string, actor string) (MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialDamage{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)

	var item MaterialDamage
	var itemTenantID string
	err = tx.QueryRow(ctx, `
SELECT mdr.id::text, mdr.tenant_id::text, mdr.material_id::text, m.name, COALESCE(mdr.reporter_id::text, ''),
       mdr.reporter, mdr.group_name,
       COALESCE(mdr.batch_id::text, ''), COALESCE((SELECT batch_no FROM material_batches WHERE id = mdr.batch_id), ''),
       COALESCE(mdr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mdr.unit_id), ''),
       mdr.quantity, mdr.reason, mdr.photo_url, mdr.attachment_url,
       mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
       COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
       COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
FROM material_damage_reports mdr
JOIN materials m ON m.id = mdr.material_id
WHERE mdr.id = $1 AND mdr.status = 'approved'
  AND ($2::boolean OR mdr.tenant_id = $3::uuid)
FOR UPDATE
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.MaterialID, &item.MaterialName, &item.ReporterID, &item.Reporter, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Quantity, &item.Reason, &item.PhotoURL, &item.AttachmentURL, &item.Status, &item.Reviewer, &item.ReviewComment, &item.CreatedAt, &item.ReviewedAt, &item.ProcessedAt)
	if err != nil {
		return MaterialDamage{}, err
	}
	if item.UnitID == "" || item.Quantity != 1 {
		return MaterialDamage{}, clientError("material damage missing unit")
	}
	if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'damaged', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
		return MaterialDamage{}, err
	}
	if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
		return MaterialDamage{}, err
	}
	if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
		return MaterialDamage{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO inventory_ledger (tenant_id, material_id, damage_id, change_qty, reason)
VALUES ($1, $2, $3, $4, $5)
`, itemTenantID, item.MaterialID, item.ID, -item.Quantity, materialUnitReason("损毁处理："+item.Reason, item.BatchNo, item.UnitCode)); err != nil {
		return MaterialDamage{}, err
	}
	if err := tx.QueryRow(ctx, `
UPDATE material_damage_reports mdr
SET status = 'processed', processed_at = now()
WHERE mdr.id = $1 AND mdr.tenant_id = $2::uuid
RETURNING mdr.id::text, mdr.material_id::text, (SELECT name FROM materials WHERE id = mdr.material_id),
          COALESCE(mdr.reporter_id::text, ''), mdr.reporter, mdr.group_name,
          COALESCE(mdr.batch_id::text, ''), $3,
          COALESCE(mdr.unit_id::text, ''), $4,
          mdr.quantity, mdr.reason,
          mdr.photo_url, mdr.attachment_url, mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
          COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
          COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
`, id, itemTenantID, item.BatchNo, item.UnitCode).Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.ReporterID, &item.Reporter, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Quantity, &item.Reason, &item.PhotoURL, &item.AttachmentURL, &item.Status, &item.Reviewer, &item.ReviewComment, &item.CreatedAt, &item.ReviewedAt, &item.ProcessedAt); err != nil {
		return MaterialDamage{}, err
	}
	if item.ReporterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.ReporterID, item.GroupName, "", "personal", "损毁登记状态更新", fmt.Sprintf("%s 编号 %s 的损毁登记状态已更新为%s。", item.MaterialName, item.UnitCode, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialDamage{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialDamage{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: itemTenantID})
	r.audit(auditCtx, actor, "material_damage.process", "material_damage", item.ID, "approved", item.Status)
	return item, nil
}

func (r *Repository) CancelMaterialDamage(ctx context.Context, id string, actor string) (MaterialDamage, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var itemTenantID string
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialDamage{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	item, err := scanMaterialDamage(tx.QueryRow(ctx, `
UPDATE material_damage_reports mdr
SET status = 'cancelled'
WHERE mdr.id = $1 AND mdr.status = 'pending'
  AND ($2::boolean OR mdr.tenant_id = $3::uuid)
RETURNING mdr.id::text, mdr.material_id::text, (SELECT name FROM materials WHERE id = mdr.material_id),
          COALESCE(mdr.reporter_id::text, ''), mdr.reporter, mdr.group_name,
          COALESCE(mdr.batch_id::text, ''), COALESCE((SELECT batch_no FROM material_batches WHERE id = mdr.batch_id), ''),
          COALESCE(mdr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mdr.unit_id), ''),
          mdr.quantity, mdr.reason,
          mdr.photo_url, mdr.attachment_url, mdr.status, mdr.reviewer, mdr.review_comment, mdr.created_at,
          COALESCE(mdr.reviewed_at, '0001-01-01 00:00:00+00'::timestamptz),
          COALESCE(mdr.processed_at, '0001-01-01 00:00:00+00'::timestamptz)
`, id, tenant.AllTenants, tenant.TenantID))
	if err != nil {
		return MaterialDamage{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT tenant_id::text FROM material_damage_reports WHERE id = $1`, item.ID).Scan(&itemTenantID); err != nil {
		return MaterialDamage{}, err
	}
	if item.UnitID != "" {
		if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'available', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
			return MaterialDamage{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
			return MaterialDamage{}, err
		}
		if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
			return MaterialDamage{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialDamage{}, err
	}
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: itemTenantID})
	r.audit(auditCtx, actor, "material_damage.cancel", "material_damage", item.ID, "", item.Status)
	return item, nil
}

func (r *Repository) MaterialAlertActions(ctx context.Context) ([]MaterialAlertAction, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT maa.id::text, maa.material_id::text, m.name, maa.alert_type, maa.action, maa.comment, maa.actor, maa.created_at
FROM material_alert_actions maa
JOIN materials m ON m.id = maa.material_id
WHERE ($1::boolean OR maa.tenant_id = $2::uuid)
ORDER BY maa.created_at DESC
LIMIT 100
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaterialAlertAction, 0)
	for rows.Next() {
		var item MaterialAlertAction
		if err := rows.Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.AlertType, &item.Action, &item.Comment, &item.Actor, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateMaterialAlertAction(ctx context.Context, materialID string, input MaterialAlertActionInput) (MaterialAlertAction, error) {
	tenant := TenantFromContext(ctx)
	materialID = strings.TrimSpace(materialID)
	input.AlertType = strings.TrimSpace(input.AlertType)
	input.Action = strings.TrimSpace(input.Action)
	input.Comment = strings.TrimSpace(input.Comment)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.AlertType == "" || (input.Action != "handled" && input.Action != "ignored") {
		return MaterialAlertAction{}, clientError("invalid material alert action input")
	}
	var item MaterialAlertAction
	err := r.db.QueryRow(ctx, `
INSERT INTO material_alert_actions (tenant_id, material_id, alert_type, action, comment, actor)
SELECT m.tenant_id, m.id, $2, $3, $4, $5
FROM materials m
WHERE m.id = $1 AND ($6::boolean OR m.tenant_id = $7::uuid)
RETURNING id::text, material_id::text, (SELECT name FROM materials WHERE id = material_id),
          alert_type, action, comment, actor, created_at
`, materialID, input.AlertType, input.Action, input.Comment, input.Actor, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &item.MaterialID, &item.MaterialName, &item.AlertType, &item.Action, &item.Comment, &item.Actor, &item.CreatedAt)
	if err != nil {
		return MaterialAlertAction{}, err
	}
	r.audit(ctx, input.Actor, "material_alert."+input.Action, "material", item.MaterialID, item.AlertType, item.Comment)
	return item, nil
}

func (r *Repository) MaterialAnalytics(ctx context.Context) (MaterialAnalytics, error) {
	materials, err := r.Materials(ctx)
	if err != nil {
		return MaterialAnalytics{}, err
	}
	requests, err := r.MaterialRequests(ctx)
	if err != nil {
		return MaterialAnalytics{}, err
	}
	damages, err := r.MaterialDamages(ctx)
	if err != nil {
		return MaterialAnalytics{}, err
	}
	alertActions, err := r.MaterialAlertActions(ctx)
	if err != nil {
		return MaterialAnalytics{}, err
	}
	now := appNow()
	today := appDateStringAt(now)
	result := MaterialAnalytics{
		ProductTotal:         len(materials),
		MonthlyConsumption:   make([]MaterialConsumptionPoint, 0, 12),
		TopConsumedMaterials: make([]MaterialConsumptionRanking, 0),
		DamageByReason:       make([]MaterialDamageReasonStat, 0),
		ProductTypeBreakdown: make([]MaterialBreakdown, 0),
		CategoryBreakdown:    make([]MaterialBreakdown, 0),
		LatestAlertActions:   alertActions,
	}
	productBreakdown := make(map[string]MaterialBreakdown)
	categoryBreakdown := make(map[string]MaterialBreakdown)
	for _, item := range materials {
		result.StockTotal += item.Stock
		if item.ProductType == "standard" {
			result.StandardTotal++
		}
		switch item.Status {
		case "near_expiry":
			result.NearExpiryTotal++
		case "expired":
			result.ExpiredTotal++
		case "low":
			result.LowStockTotal++
		case "damaged":
			result.DamagedTotal += item.DamagedQuantity
		}
		product := productBreakdown[item.ProductType]
		product.Label = item.ProductType
		product.Count++
		product.Stock += item.Stock
		productBreakdown[item.ProductType] = product
		category := categoryBreakdown[item.Category]
		category.Label = item.Category
		category.Count++
		category.Stock += item.Stock
		categoryBreakdown[item.Category] = category
	}
	monthly := make(map[string]int)
	consumption := make(map[string]MaterialConsumptionRanking)
	for i := 11; i >= 0; i-- {
		month := now.AddDate(0, -i, 0).Format("2006-01")
		monthly[month] = 0
	}
	for _, item := range requests {
		if item.Status != "outbound" {
			continue
		}
		createdDate := appDateStringAt(item.CreatedAt)
		if createdDate == today {
			result.TodayUsageTotal += item.Quantity
		}
		month := item.CreatedAt.In(appLocation).Format("2006-01")
		if _, ok := monthly[month]; ok {
			monthly[month] += item.Quantity
		}
		ranking := consumption[item.MaterialID]
		ranking.MaterialID = item.MaterialID
		ranking.MaterialName = item.MaterialName
		ranking.Quantity += item.Quantity
		consumption[item.MaterialID] = ranking
	}
	for i := 11; i >= 0; i-- {
		month := now.AddDate(0, -i, 0).Format("2006-01")
		result.MonthlyConsumption = append(result.MonthlyConsumption, MaterialConsumptionPoint{Month: month, Quantity: monthly[month]})
	}
	for _, item := range topMaterialConsumption(consumption, 8) {
		result.TopConsumedMaterials = append(result.TopConsumedMaterials, item)
	}
	damageReasons := make(map[string]int)
	for _, item := range damages {
		if item.Status != "processed" {
			continue
		}
		result.DamagedTotal += item.Quantity
		reason := item.Reason
		if reason == "" {
			reason = "未填写原因"
		}
		damageReasons[reason] += item.Quantity
	}
	for reason, quantity := range damageReasons {
		result.DamageByReason = append(result.DamageByReason, MaterialDamageReasonStat{Reason: reason, Quantity: quantity})
	}
	for _, item := range productBreakdown {
		result.ProductTypeBreakdown = append(result.ProductTypeBreakdown, item)
	}
	for _, item := range categoryBreakdown {
		result.CategoryBreakdown = append(result.CategoryBreakdown, item)
	}
	return result, nil
}

func (r *Repository) resolveMaterialActor(ctx context.Context, userID string, userName string, tenant TenantContext) (string, string, string, string, string, bool, error) {
	var resolvedID, resolvedTenantID, resolvedName, status, groupName string
	var emailVerified bool
	var err error
	if userID != "" {
		err = r.db.QueryRow(ctx, `
SELECT id::text, tenant_id::text, name, status, group_name, email_verified
FROM users
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, userID, tenant.AllTenants, tenant.TenantID).Scan(&resolvedID, &resolvedTenantID, &resolvedName, &status, &groupName, &emailVerified)
	} else {
		err = r.db.QueryRow(ctx, `
SELECT id::text, tenant_id::text, name, status, group_name, email_verified
FROM users
WHERE name = $1 AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY created_at DESC
LIMIT 1
`, userName, tenant.AllTenants, tenant.TenantID).Scan(&resolvedID, &resolvedTenantID, &resolvedName, &status, &groupName, &emailVerified)
	}
	return resolvedID, resolvedTenantID, resolvedName, status, groupName, emailVerified, err
}

func (r *Repository) MaintenanceOrders(ctx context.Context) ([]MaintenanceOrder, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT mo.id::text, COALESCE(mo.instrument_id::text, ''), COALESCE(i.name, '已删除仪器'), mo.type, mo.priority, mo.status, mo.handler,
       mo.description, mo.result, lower(mo.period), upper(mo.period), mo.created_at
FROM maintenance_orders mo
LEFT JOIN instruments i ON i.id = mo.instrument_id
WHERE ($1::boolean OR mo.tenant_id = $2::uuid)
ORDER BY mo.created_at DESC
LIMIT 100
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MaintenanceOrder, 0)
	for rows.Next() {
		var item MaintenanceOrder
		if err := rows.Scan(&item.ID, &item.InstrumentID, &item.InstrumentName, &item.Type, &item.Priority, &item.Status, &item.Handler, &item.Description, &item.Result, &item.StartTime, &item.EndTime, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateMaintenanceOrder(ctx context.Context, input MaintenanceInput) (MaintenanceOrder, error) {
	tenant := TenantFromContext(ctx)
	input.Type = strings.TrimSpace(input.Type)
	input.Priority = strings.TrimSpace(input.Priority)
	input.Handler = strings.TrimSpace(input.Handler)
	input.Description = strings.TrimSpace(input.Description)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Type == "" {
		input.Type = "routine"
	}
	if input.Priority == "" {
		input.Priority = "normal"
	}
	if input.InstrumentID == "" || input.Description == "" || !input.EndTime.After(input.StartTime) {
		return MaintenanceOrder{}, clientError("invalid maintenance input")
	}
	status := "assigned"
	if input.Handler == "" {
		status = "reported"
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaintenanceOrder{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var targetTenantID string
	var instrumentName string
	if err := tx.QueryRow(ctx, `
SELECT tenant_id::text, name
FROM instruments
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.InstrumentID, tenant.AllTenants, tenant.TenantID).Scan(&targetTenantID, &instrumentName); err != nil {
		return MaintenanceOrder{}, err
	}

	var inUseCount int
	if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM reservations
WHERE instrument_id = $1 AND status = 'in_use' AND period && tstzrange($2, $3, '[)')
  AND tenant_id = $4::uuid
`, input.InstrumentID, input.StartTime, input.EndTime, targetTenantID).Scan(&inUseCount); err != nil {
		return MaintenanceOrder{}, err
	}
	if inUseCount > 0 && input.Type != "emergency" {
		return MaintenanceOrder{}, clientError("maintenance conflicts with an in-use reservation")
	}

	var item MaintenanceOrder
	err = tx.QueryRow(ctx, `
INSERT INTO maintenance_orders (tenant_id, instrument_id, type, priority, status, handler, description, period)
VALUES ($9, $1, $2, $3, $4, $5, $6, tstzrange($7, $8, '[)'))
RETURNING id::text, instrument_id::text, $10::text, type, priority, status, handler, description, result, lower(period), upper(period), created_at
`, input.InstrumentID, input.Type, input.Priority, status, input.Handler, input.Description, input.StartTime, input.EndTime, targetTenantID, instrumentName).Scan(
		&item.ID, &item.InstrumentID, &item.InstrumentName, &item.Type, &item.Priority, &item.Status, &item.Handler, &item.Description, &item.Result, &item.StartTime, &item.EndTime, &item.CreatedAt,
	)
	if err != nil {
		return MaintenanceOrder{}, err
	}
	notifications := make([]Notification, 0)

	rows, err := tx.Query(ctx, `
UPDATE reservations
SET status = 'cancelled', cancel_reason = '设备维护窗口冲突', cancelled_at = now()
WHERE instrument_id = $1
  AND status IN ('pending', 'approved')
  AND period && tstzrange($2, $3, '[)')
  AND tenant_id = $4::uuid
RETURNING id::text, COALESCE(user_id::text, ''), user_name, group_name
`, input.InstrumentID, input.StartTime, input.EndTime, targetTenantID)
	if err != nil {
		return MaintenanceOrder{}, err
	}
	cancelled := make([]string, 0)
	for rows.Next() {
		var reservationID string
		var userID string
		var userName string
		var groupName string
		if err := rows.Scan(&reservationID, &userID, &userName, &groupName); err != nil {
			rows.Close()
			return MaintenanceOrder{}, err
		}
		cancelled = append(cancelled, fmt.Sprintf("%s/%s", reservationID, userName))
		if userID != "" {
			notification, err := r.createNotificationTx(ctx, tx, targetTenantID, userID, groupName, "", "personal", "预约受维护影响", fmt.Sprintf("%s 的预约因 %s 维护窗口被取消，请重新安排。", userName, item.InstrumentName), "warning")
			if err != nil {
				rows.Close()
				return MaintenanceOrder{}, err
			}
			notifications = append(notifications, notification)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return MaintenanceOrder{}, err
	}

	if _, err := tx.Exec(ctx, `UPDATE instruments SET status = 'maintenance', maintenance_summary = $2 WHERE id = $1 AND tenant_id = $3::uuid`, input.InstrumentID, input.Description, targetTenantID); err != nil {
		return MaintenanceOrder{}, err
	}
	notification, err := r.createNotificationTx(ctx, tx, targetTenantID, "", "", "", "global", "设备维护安排", fmt.Sprintf("%s 已进入维护，影响预约 %d 条。", item.InstrumentName, len(cancelled)), "warning")
	if err != nil {
		return MaintenanceOrder{}, err
	}
	notifications = append(notifications, notification)
	if err := tx.Commit(ctx); err != nil {
		return MaintenanceOrder{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	r.invalidateDashboard(ctx)
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: targetTenantID})
	r.audit(auditCtx, input.Actor, "maintenance.create", "maintenance_order", item.ID, "", strings.Join(cancelled, ","))
	return item, nil
}

func (r *Repository) StartMaintenanceOrder(ctx context.Context, id string, actor string) (MaintenanceOrder, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var item MaintenanceOrder
	var itemTenantID string
	err := r.db.QueryRow(ctx, `
UPDATE maintenance_orders mo
SET status = 'in_progress'
WHERE mo.id = $1 AND mo.status IN ('reported', 'assigned')
  AND ($2::boolean OR mo.tenant_id = $3::uuid)
RETURNING mo.id::text, mo.tenant_id::text, COALESCE(mo.instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = mo.instrument_id), '已删除仪器'), mo.type, mo.priority, mo.status, mo.handler, mo.description, mo.result, lower(mo.period), upper(mo.period), mo.created_at
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.InstrumentID, &item.InstrumentName, &item.Type, &item.Priority, &item.Status, &item.Handler, &item.Description, &item.Result, &item.StartTime, &item.EndTime, &item.CreatedAt)
	if err != nil {
		return MaintenanceOrder{}, err
	}
	if item.InstrumentID == "" {
		return MaintenanceOrder{}, clientError("instrument has been deleted")
	}
	if _, err := r.db.Exec(ctx, `UPDATE instruments SET status = 'maintenance', maintenance_summary = $2 WHERE id = $1 AND tenant_id = $3::uuid`, item.InstrumentID, item.Description, itemTenantID); err != nil {
		return MaintenanceOrder{}, err
	}
	r.invalidateDashboard(ctx)
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: itemTenantID})
	r.audit(auditCtx, actor, "maintenance.start", "maintenance_order", item.ID, "", item.Status)
	return item, nil
}

func (r *Repository) CancelMaintenanceOrder(ctx context.Context, id string, reason string, actor string) (MaintenanceOrder, error) {
	tenant := TenantFromContext(ctx)
	reason = strings.TrimSpace(reason)
	actor = strings.TrimSpace(actor)
	if reason == "" {
		reason = "维护取消"
	}
	if actor == "" {
		actor = "system"
	}
	var item MaintenanceOrder
	var itemTenantID string
	err := r.db.QueryRow(ctx, `
UPDATE maintenance_orders mo
SET status = 'cancelled', result = $2
WHERE mo.id = $1 AND mo.status IN ('reported', 'assigned', 'in_progress')
  AND ($3::boolean OR mo.tenant_id = $4::uuid)
RETURNING mo.id::text, mo.tenant_id::text, COALESCE(mo.instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = mo.instrument_id), '已删除仪器'), mo.type, mo.priority, mo.status, mo.handler, mo.description, mo.result, lower(mo.period), upper(mo.period), mo.created_at
`, id, reason, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.InstrumentID, &item.InstrumentName, &item.Type, &item.Priority, &item.Status, &item.Handler, &item.Description, &item.Result, &item.StartTime, &item.EndTime, &item.CreatedAt)
	if err != nil {
		return MaintenanceOrder{}, err
	}
	if err := r.refreshInstrumentAfterMaintenance(ctx, item.InstrumentID, itemTenantID, reason); err != nil {
		return MaintenanceOrder{}, err
	}
	r.invalidateDashboard(ctx)
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: itemTenantID})
	r.audit(auditCtx, actor, "maintenance.cancel", "maintenance_order", item.ID, "", reason)
	return item, nil
}

func (r *Repository) CompleteMaintenanceOrder(ctx context.Context, id string, result string, actor string) (MaintenanceOrder, error) {
	tenant := TenantFromContext(ctx)
	result = strings.TrimSpace(result)
	actor = strings.TrimSpace(actor)
	if result == "" {
		result = "维护完成，仪器恢复可用。"
	}
	if actor == "" {
		actor = "system"
	}
	var item MaintenanceOrder
	var itemTenantID string
	err := r.db.QueryRow(ctx, `
UPDATE maintenance_orders mo
SET status = 'completed', result = $2
WHERE mo.id = $1 AND mo.status IN ('reported', 'assigned', 'in_progress')
  AND ($3::boolean OR mo.tenant_id = $4::uuid)
RETURNING mo.id::text, mo.tenant_id::text, COALESCE(mo.instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = mo.instrument_id), '已删除仪器'), mo.type, mo.priority, mo.status, mo.handler, mo.description, mo.result, lower(mo.period), upper(mo.period), mo.created_at
`, id, result, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.InstrumentID, &item.InstrumentName, &item.Type, &item.Priority, &item.Status, &item.Handler, &item.Description, &item.Result, &item.StartTime, &item.EndTime, &item.CreatedAt)
	if err != nil {
		return MaintenanceOrder{}, err
	}
	if err := r.refreshInstrumentAfterMaintenance(ctx, item.InstrumentID, itemTenantID, result); err != nil {
		return MaintenanceOrder{}, err
	}
	r.invalidateDashboard(ctx)
	auditCtx := WithTenantContext(ctx, TenantContext{TenantID: itemTenantID})
	r.audit(auditCtx, actor, "maintenance.complete", "maintenance_order", item.ID, "", result)
	return item, nil
}

func (r *Repository) refreshInstrumentAfterMaintenance(ctx context.Context, instrumentID string, tenantID string, summary string) error {
	if strings.TrimSpace(instrumentID) == "" {
		return nil
	}
	var activeCount int
	if err := r.db.QueryRow(ctx, `
SELECT count(*)
FROM maintenance_orders
WHERE instrument_id = $1 AND tenant_id = $2::uuid AND status IN ('reported', 'assigned', 'in_progress')
`, instrumentID, tenantID).Scan(&activeCount); err != nil {
		return err
	}
	if activeCount > 0 {
		_, err := r.db.Exec(ctx, `UPDATE instruments SET status = 'maintenance', maintenance_summary = $2 WHERE id = $1 AND tenant_id = $3::uuid`, instrumentID, summary, tenantID)
		return err
	}
	_, err := r.db.Exec(ctx, `UPDATE instruments SET status = 'available', maintenance_summary = $2 WHERE id = $1 AND tenant_id = $3::uuid`, instrumentID, summary, tenantID)
	return err
}

func (r *Repository) AuditEvents(ctx context.Context) ([]AuditEvent, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, actor, action, target_type, target_id, old_value, new_value, created_at
FROM audit_events
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY created_at DESC
LIMIT 100
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AuditEvent, 0)
	for rows.Next() {
		var item AuditEvent
		if err := rows.Scan(&item.ID, &item.Actor, &item.Action, &item.TargetType, &item.TargetID, &item.OldValue, &item.NewValue, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Operations(ctx context.Context) (Operations, error) {
	tenant := TenantFromContext(ctx)
	dashboard, err := r.Dashboard(ctx)
	if err != nil {
		return Operations{}, err
	}
	ops := Operations{
		Dashboard: dashboard,
		UpdatedAt: time.Now().UTC(),
		Alerts:    make([]OperationAlert, 0),
	}
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM reservations WHERE status = 'in_use' AND ($1::boolean OR tenant_id = $2::uuid)`, tenant.AllTenants, tenant.TenantID).Scan(&ops.InUseInstruments); err != nil {
		return Operations{}, err
	}

	trendRows, err := r.db.Query(ctx, `
SELECT to_char(hour_bucket, 'HH24:MI'), count(r.id)::int
FROM generate_series(date_trunc('hour', now()) - interval '23 hours', date_trunc('hour', now()), interval '1 hour') AS hour_bucket
LEFT JOIN reservations r ON lower(r.period) >= hour_bucket AND lower(r.period) < hour_bucket + interval '1 hour'
  AND ($1::boolean OR r.tenant_id = $2::uuid)
GROUP BY hour_bucket
ORDER BY hour_bucket
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return Operations{}, err
	}
	for trendRows.Next() {
		var point TrendPoint
		if err := trendRows.Scan(&point.Hour, &point.Count); err != nil {
			trendRows.Close()
			return Operations{}, err
		}
		ops.ReservationTrend = append(ops.ReservationTrend, point)
	}
	trendRows.Close()
	if err := trendRows.Err(); err != nil {
		return Operations{}, err
	}

	loadRows, err := r.db.Query(ctx, `
SELECT i.name, COALESCE(sum(EXTRACT(EPOCH FROM (upper(r.period) - lower(r.period))) / 3600), 0)::float8 AS hours
FROM instruments i
LEFT JOIN reservations r ON r.instrument_id = i.id AND r.status IN ('approved', 'in_use', 'completed')
WHERE ($1::boolean OR i.tenant_id = $2::uuid)
GROUP BY i.name
ORDER BY hours DESC, i.name
LIMIT 8
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return Operations{}, err
	}
	for loadRows.Next() {
		var item InstrumentLoad
		if err := loadRows.Scan(&item.InstrumentName, &item.Hours); err != nil {
			loadRows.Close()
			return Operations{}, err
		}
		ops.InstrumentLoads = append(ops.InstrumentLoads, item)
	}
	loadRows.Close()
	if err := loadRows.Err(); err != nil {
		return Operations{}, err
	}

	var reservationApprovalHours float64
	if err := r.db.QueryRow(ctx, `
SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (decided_at - created_at))) / 3600, 0)::float8
FROM reservations
WHERE decided_at IS NOT NULL
  AND ($1::boolean OR tenant_id = $2::uuid)
`, tenant.AllTenants, tenant.TenantID).Scan(&reservationApprovalHours); err != nil {
		return Operations{}, err
	}
	var materialApprovalHours float64
	if err := r.db.QueryRow(ctx, `
SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (decided_at - created_at))) / 3600, 0)::float8
FROM material_requests
WHERE decided_at IS NOT NULL
  AND ($1::boolean OR tenant_id = $2::uuid)
`, tenant.AllTenants, tenant.TenantID).Scan(&materialApprovalHours); err != nil {
		return Operations{}, err
	}
	var maintenanceResponseHours float64
	if err := r.db.QueryRow(ctx, `
SELECT COALESCE(AVG(GREATEST(EXTRACT(EPOCH FROM (lower(period) - created_at)), 0)) / 3600, 0)::float8
FROM maintenance_orders
WHERE ($1::boolean OR tenant_id = $2::uuid)
`, tenant.AllTenants, tenant.TenantID).Scan(&maintenanceResponseHours); err != nil {
		return Operations{}, err
	}
	ops.ApprovalEfficiency = []ApprovalMetric{
		{Label: "预约审批平均处理", Hours: reservationApprovalHours},
		{Label: "耗材审批平均处理", Hours: materialApprovalHours},
		{Label: "维护响应平均处理", Hours: maintenanceResponseHours},
	}
	alertRows, err := r.db.Query(ctx, `
SELECT '仪器' AS source, 'warning' AS level, name || ' 当前维护中' AS body
FROM instruments
WHERE status = 'maintenance'
  AND ($1::boolean OR tenant_id = $2::uuid)
UNION ALL
SELECT '耗材', 'warning', name || ' 库存低于预警线'
FROM materials
WHERE stock <= warning_line
  AND ($1::boolean OR tenant_id = $2::uuid)
LIMIT 10
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return Operations{}, err
	}
	for alertRows.Next() {
		var alert OperationAlert
		if err := alertRows.Scan(&alert.Source, &alert.Level, &alert.Body); err != nil {
			alertRows.Close()
			return Operations{}, err
		}
		ops.Alerts = append(ops.Alerts, alert)
	}
	alertRows.Close()
	if err := alertRows.Err(); err != nil {
		return Operations{}, err
	}
	ops.AlertCount = len(ops.Alerts)
	return ops, nil
}

func (r *Repository) findDuplicateReservation(ctx context.Context, tx pgx.Tx, input ReservationInput) (Reservation, error) {
	tenant := TenantFromContext(ctx)
	var reservation Reservation
	err := tx.QueryRow(ctx, `
SELECT r.id::text, r.tenant_id::text, COALESCE(r.user_id::text, ''), COALESCE(r.instrument_id::text, ''),
       COALESCE((SELECT name FROM instruments WHERE id = r.instrument_id), '已删除仪器'),
       r.user_name, r.group_name, r.purpose, lower(r.period), upper(r.period), r.status, r.fee::float8
FROM reservations r
WHERE (r.idempotency_key = $1 OR (
        r.instrument_id = $2
    AND r.user_name = $3
    AND r.purpose = $4
    AND lower(r.period) = $5
    AND upper(r.period) = $6
))
  AND r.status IN ('pending', 'approved', 'in_use')
  AND ($7::boolean OR r.tenant_id = $8::uuid)
ORDER BY r.created_at DESC
LIMIT 1
`, input.IdempotencyKey, input.InstrumentID, input.UserName, input.Purpose, input.StartTime, input.EndTime, tenant.AllTenants, tenant.TenantID).Scan(
		&reservation.ID,
		&reservation.TenantID,
		&reservation.UserID,
		&reservation.InstrumentID,
		&reservation.InstrumentName,
		&reservation.UserName,
		&reservation.GroupName,
		&reservation.Purpose,
		&reservation.StartTime,
		&reservation.EndTime,
		&reservation.Status,
		&reservation.Fee,
	)
	return reservation, err
}

func (r *Repository) updateMaterialRequestStatus(ctx context.Context, id string, status string, actor string, comment string) (MaterialRequest, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	comment = strings.TrimSpace(comment)
	if actor == "" {
		actor = "system"
	}
	if comment == "" {
		comment = status
	}
	var item MaterialRequest
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialRequest{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	var itemTenantID string
	err = tx.QueryRow(ctx, `
UPDATE material_requests mr
SET status = $2, decided_at = now()
WHERE mr.id = $1 AND mr.status = 'pending'
  AND ($3::boolean OR mr.tenant_id = $4::uuid)
RETURNING mr.id::text, mr.tenant_id::text, mr.material_id::text, (SELECT name FROM materials WHERE id = mr.material_id),
          COALESCE(mr.requester_id::text, ''), mr.requester, mr.group_name, COALESCE(mr.batch_id::text, ''),
          COALESCE((SELECT batch_no FROM material_batches WHERE id = mr.batch_id), ''),
          COALESCE(mr.unit_id::text, ''), COALESCE((SELECT unit_code FROM material_units WHERE id = mr.unit_id), ''),
          COALESCE((SELECT location FROM material_units WHERE id = mr.unit_id), (SELECT location FROM material_batches WHERE id = mr.batch_id), ''),
          mr.quantity, mr.purpose, mr.status, mr.created_at
`, id, status, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &itemTenantID, &item.MaterialID, &item.MaterialName, &item.RequesterID, &item.Requester, &item.GroupName, &item.BatchID, &item.BatchNo, &item.UnitID, &item.UnitCode, &item.Location, &item.Quantity, &item.Purpose, &item.Status, &item.CreatedAt)
	if err != nil {
		return MaterialRequest{}, err
	}
	action := "reject"
	if status == "approved" {
		action = "approve"
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_approval_actions (tenant_id, material_request_id, actor, action, comment)
VALUES ($1, $2, $3, $4, $5)
`, itemTenantID, item.ID, actor, action, comment); err != nil {
		return MaterialRequest{}, err
	}
	if status == "rejected" && item.UnitID != "" {
		if _, err := tx.Exec(ctx, `
UPDATE material_units
SET status = 'available', updated_at = now()
WHERE id = $1 AND material_id = $2 AND tenant_id = $3::uuid AND status = 'reserved'
`, item.UnitID, item.MaterialID, itemTenantID); err != nil {
			return MaterialRequest{}, err
		}
		if err := syncMaterialBatchQuantity(ctx, tx, item.BatchID); err != nil {
			return MaterialRequest{}, err
		}
		if _, err := syncMaterialStock(ctx, tx, item.MaterialID, itemTenantID); err != nil {
			return MaterialRequest{}, err
		}
	}
	if item.RequesterID != "" {
		locationText := ""
		if status == "approved" {
			locationText = fmt.Sprintf("，储存位置：%s", firstNonEmpty(item.Location, "未登记"))
		}
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申领状态更新", fmt.Sprintf("%s x%d 的申领状态已更新为%s%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status), locationText), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialRequest{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material."+status, "material_request", item.ID, "pending", status); err != nil {
		return MaterialRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialRequest{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func (r *Repository) updateMaterialPurchaseStatus(ctx context.Context, id string, status string, actor string, comment string) (MaterialPurchase, error) {
	tenant := TenantFromContext(ctx)
	action, ok := materialPurchaseStatusAction(status)
	if !ok {
		return MaterialPurchase{}, clientError("invalid material purchase status")
	}
	actor = strings.TrimSpace(actor)
	comment = strings.TrimSpace(comment)
	if actor == "" {
		actor = "system"
	}
	if comment == "" {
		comment = status
	}
	var item MaterialPurchase
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MaterialPurchase{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	notifications := make([]Notification, 0, 1)
	var itemTenantID string
	err = tx.QueryRow(ctx, fmt.Sprintf(`
WITH updated AS (
  UPDATE material_purchases
  SET status = $2, decided_at = now()
  WHERE id = $1 AND status = 'registered'
    AND ($3::boolean OR tenant_id = $4::uuid)
    AND NOT EXISTS (
        SELECT 1
        FROM material_purchase_monthly_confirmations mpmc
        WHERE mpmc.tenant_id = material_purchases.tenant_id
          AND mpmc.month = to_char(material_purchases.created_at, 'YYYY-MM')
    )
  RETURNING *
)
SELECT %s, mp.tenant_id::text
FROM updated mp
LEFT JOIN materials m ON m.id = mp.material_id
	`, materialPurchaseSelectColumns()), id, status, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &item.PurchaseSerialNo, &item.MonthlyConfirmed, &item.MaterialID, &item.MaterialName, &item.PurchasableMaterialID, &item.PurchaseIDNo, &item.PurchaseSequenceNo, &item.PurchaseProjectName, &item.PurchaseItemName, &item.PurchaseBrand, &item.PurchaseSpec, &item.PurchaseUnit, &item.PurchaseRemark, &item.PurchaseTechnicalRequirement, &item.PurchaseMinSpec, &item.RequesterID, &item.Requester, &item.RequesterPhone, &item.RequesterEmail, &item.GroupName, &item.Quantity, &item.EstimatedUnitPrice, &item.Supplier, &item.Reason, &item.Status, &item.CreatedAt, &itemTenantID)
	if err != nil {
		return MaterialPurchase{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO material_purchase_actions (tenant_id, material_purchase_id, actor, action, comment)
VALUES ($1, $2, $3, $4, $5)
`, itemTenantID, item.ID, actor, action, comment); err != nil {
		return MaterialPurchase{}, err
	}
	if item.RequesterID != "" {
		notification, err := r.createNotificationTx(ctx, tx, itemTenantID, item.RequesterID, item.GroupName, "", "personal", "耗材申购状态更新", fmt.Sprintf("%s x%d 的申购状态已更新为%s。", item.MaterialName, item.Quantity, materialWorkflowStatusLabel(item.Status)), notificationLevelForStatus(item.Status))
		if err != nil {
			return MaterialPurchase{}, err
		}
		notifications = append(notifications, notification)
	}
	if err := r.auditTx(ctx, tx, itemTenantID, actor, "material_purchase."+status, "material_purchase", item.ID, "registered", status); err != nil {
		return MaterialPurchase{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MaterialPurchase{}, err
	}
	r.enqueueDingTalkNotifications(notifications...)
	return item, nil
}

func materialPurchaseStatusAction(status string) (string, bool) {
	switch status {
	case "approved":
		return "approve", true
	case "rejected":
		return "reject", true
	case "returned":
		return "return", true
	default:
		return "", false
	}
}

func (r *Repository) audit(ctx context.Context, actor string, action string, targetType string, targetID string, oldValue string, newValue string) {
	if r.db == nil {
		return
	}
	tenant := TenantFromContext(ctx)
	if err := r.auditTx(ctx, r.db, tenant.TenantID, actor, action, targetType, targetID, oldValue, newValue); err != nil {
		slog.Warn("write audit event", "action", action, "error", err)
	}
}

type auditWriter interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *Repository) auditTx(ctx context.Context, writer auditWriter, tenantID string, actor string, action string, targetType string, targetID string, oldValue string, newValue string) error {
	if writer == nil {
		return nil
	}
	if strings.TrimSpace(tenantID) == "" {
		tenantID = defaultTenantID
	}
	_, err := writer.Exec(ctx, `
INSERT INTO audit_events (tenant_id, actor, action, target_type, target_id, old_value, new_value)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`, tenantID, actor, action, targetType, targetID, oldValue, newValue)
	return err
}

type settingMeta struct {
	updatedBy string
	updatedAt time.Time
}

func (r *Repository) saveJSONSetting(ctx context.Context, key string, payload []byte, actor string) (settingMeta, error) {
	var meta settingMeta
	err := r.db.QueryRow(ctx, `
INSERT INTO site_settings (setting_key, value, updated_by)
VALUES ($1, $2::jsonb, $3)
ON CONFLICT (setting_key) DO UPDATE
SET value = EXCLUDED.value,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
RETURNING updated_by, updated_at
`, key, string(payload), actor).Scan(&meta.updatedBy, &meta.updatedAt)
	return meta, err
}

func (r *Repository) readGraphMailSettings(ctx context.Context) (graphMailSettingsValue, settingMeta, error) {
	var raw []byte
	var meta settingMeta
	err := r.db.QueryRow(ctx, `SELECT value, updated_by, updated_at FROM site_settings WHERE setting_key = $1`, graphMailSettingsKey).Scan(&raw, &meta.updatedBy, &meta.updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return graphMailSettingsValue{}, settingMeta{updatedBy: "system"}, nil
	}
	if err != nil {
		return graphMailSettingsValue{}, settingMeta{}, err
	}
	var value graphMailSettingsValue
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &value); err != nil {
			return graphMailSettingsValue{}, settingMeta{}, err
		}
	}
	value.TenantID = strings.TrimSpace(value.TenantID)
	value.ClientID = strings.TrimSpace(value.ClientID)
	value.ClientSecret = strings.TrimSpace(value.ClientSecret)
	value.SenderUserPrincipalName = strings.TrimSpace(value.SenderUserPrincipalName)
	return value, meta, nil
}

func (r *Repository) graphMailSettingsValue(ctx context.Context) (graphMailSettingsValue, error) {
	value, _, err := r.readGraphMailSettings(ctx)
	return value, err
}

func (r *Repository) readWeChatSettings(ctx context.Context) (wechatSettingsValue, settingMeta, error) {
	var raw []byte
	var meta settingMeta
	err := r.db.QueryRow(ctx, `SELECT value, updated_by, updated_at FROM site_settings WHERE setting_key = $1`, wechatSettingsKey).Scan(&raw, &meta.updatedBy, &meta.updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return wechatSettingsValue{AccountType: "service_account"}, settingMeta{updatedBy: "system"}, nil
	}
	if err != nil {
		return wechatSettingsValue{}, settingMeta{}, err
	}
	var value wechatSettingsValue
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &value); err != nil {
			return wechatSettingsValue{}, settingMeta{}, err
		}
	}
	if value.AccountType == "" {
		value.AccountType = "service_account"
	}
	return value, meta, nil
}

func (r *Repository) readDingTalkSettings(ctx context.Context) (dingTalkSettingsValue, settingMeta, error) {
	tenantID := TenantFromContext(ctx).TenantID
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	return r.readDingTalkSettingsByTenantID(ctx, tenantID)
}

func (r *Repository) readDingTalkSettingsByTenantID(ctx context.Context, tenantID string) (dingTalkSettingsValue, settingMeta, error) {
	var raw []byte
	var meta settingMeta
	settingKey := tenantScopedDingTalkSettingsKey(tenantID)
	err := r.db.QueryRow(ctx, `SELECT value, updated_by, updated_at FROM site_settings WHERE setting_key = $1`, settingKey).Scan(&raw, &meta.updatedBy, &meta.updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.db.QueryRow(ctx, `SELECT value, updated_by, updated_at FROM site_settings WHERE setting_key = $1`, dingTalkSettingsKey).Scan(&raw, &meta.updatedBy, &meta.updatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return dingTalkSettingsValue{}, settingMeta{updatedBy: "system"}, nil
		}
	}
	if err != nil {
		return dingTalkSettingsValue{}, settingMeta{}, err
	}
	var value dingTalkSettingsValue
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &value); err != nil {
			return dingTalkSettingsValue{}, settingMeta{}, err
		}
	}
	value = normalizeDingTalkSettingsValue(raw, value)
	value.ClientID = strings.TrimSpace(value.ClientID)
	value.ClientSecret = strings.TrimSpace(value.ClientSecret)
	value.CorpID = strings.TrimSpace(value.CorpID)
	value.RobotCode = strings.TrimSpace(value.RobotCode)
	value.OAuthRedirectURI = strings.TrimSpace(value.OAuthRedirectURI)
	value.EventCallbackURL = strings.TrimSpace(value.EventCallbackURL)
	value.EventAesKey = strings.TrimSpace(value.EventAesKey)
	value.EventToken = strings.TrimSpace(value.EventToken)
	return value, meta, nil
}

func normalizeDingTalkSettingsValue(raw []byte, value dingTalkSettingsValue) dingTalkSettingsValue {
	if value.SchemaVersion <= 0 {
		value.SchemaVersion = 2
	}
	return value
}

func (r *Repository) dingTalkSettingsValue(ctx context.Context) (dingTalkSettingsValue, error) {
	value, _, err := r.readDingTalkSettings(ctx)
	return value, err
}

func graphMailSettingsFromValue(value graphMailSettingsValue, updatedBy string, updatedAt time.Time) GraphMailSettings {
	return GraphMailSettings{
		Enabled:                 value.Enabled,
		TenantID:                value.TenantID,
		ClientID:                value.ClientID,
		SenderUserPrincipalName: value.SenderUserPrincipalName,
		SaveToSentItems:         value.SaveToSentItems,
		ClientSecretConfigured:  value.ClientSecret != "",
		UpdatedBy:               updatedBy,
		UpdatedAt:               updatedAt,
	}
}

func wechatSettingsFromValue(value wechatSettingsValue, updatedBy string, updatedAt time.Time) WeChatSettings {
	return WeChatSettings{
		Enabled:             value.Enabled,
		AccountType:         value.AccountType,
		AppID:               value.AppID,
		ServiceAccountName:  value.ServiceAccountName,
		TemplateID:          value.TemplateID,
		Token:               value.Token,
		EncodingAESKey:      value.EncodingAESKey,
		AppSecretConfigured: value.AppSecret != "",
		UpdatedBy:           updatedBy,
		UpdatedAt:           updatedAt,
	}
}

func dingTalkSettingsFromValue(value dingTalkSettingsValue, updatedBy string, updatedAt time.Time) DingTalkSettings {
	return DingTalkSettings{
		SchemaVersion:          2,
		Enabled:                value.Enabled,
		ClientID:               value.ClientID,
		CorpID:                 value.CorpID,
		RobotCode:              value.RobotCode,
		OAuthRedirectURI:       value.OAuthRedirectURI,
		EventCallbackURL:       value.EventCallbackURL,
		ClientSecretConfigured: value.ClientSecret != "",
		EventAesKeyConfigured:  value.EventAesKey != "",
		EventTokenConfigured:   value.EventToken != "",
		UpdatedBy:              updatedBy,
		UpdatedAt:              updatedAt,
	}
}

func (r *Repository) readAccessControlSettings(ctx context.Context) (accessControlSettingsValue, settingMeta, error) {
	var raw []byte
	var meta settingMeta
	err := r.db.QueryRow(ctx, `SELECT value, updated_by, updated_at FROM site_settings WHERE setting_key = $1`, accessControlSettingsKey).Scan(&raw, &meta.updatedBy, &meta.updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultAccessControlSettingsValue(), settingMeta{updatedBy: "system"}, nil
	}
	if err != nil {
		return accessControlSettingsValue{}, settingMeta{}, err
	}
	var value accessControlSettingsValue
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &value); err != nil {
			return accessControlSettingsValue{}, settingMeta{}, err
		}
	}
	if value.Vendor == "" {
		value.Vendor = "hikvision"
	}
	return normalizeAccessControlSettingsValue(value), meta, nil
}

func (r *Repository) accessControlSettingsValue(ctx context.Context) (accessControlSettingsValue, error) {
	value, _, err := r.readAccessControlSettings(ctx)
	return value, err
}

func defaultCopySettingsValue() copySettingsValue {
	return copySettingsValue{
		Entries: []CopyEntry{
			copyEntryValue("首页", "主导航", "首页", "nav", "顶部首页入口"),
			copyEntryValue("仪器预约", "主导航", "仪器预约", "nav", "顶部仪器预约入口"),
			copyEntryValue("资源", "主导航", "资源", "nav", "顶部资源分组"),
			copyEntryValue("业务", "主导航", "业务", "nav", "顶部业务分组"),
			copyEntryValue("培训", "主导航", "培训", "nav", "顶部培训分组"),
			copyEntryValue("更多", "主导航", "更多", "nav", "顶部更多分组"),
			copyEntryValue("管理中心", "主导航", "管理中心", "nav", "顶部管理中心入口"),
			copyEntryValue("管理后台", "主导航", "管理后台", "nav", "移动端管理入口"),
			copyEntryValue("普通用户入口", "首页分组", "普通用户入口", "page", "游客首页普通用户入口"),
			copyEntryValue("个人工作台", "首页分组", "个人工作台", "page", "普通用户首页工作台"),
			copyEntryValue("管理员工作台", "首页分组", "管理员工作台", "page", "管理员首页入口"),
			copyEntryValue("登录", "按钮", "登录", "button", "登录入口按钮"),
			copyEntryValue("注册", "按钮", "注册", "button", "注册入口按钮"),
			copyEntryValue("退出登录", "按钮", "退出登录", "button", "退出当前会话"),
			copyEntryValue("退出所有设备", "按钮", "退出所有设备", "button", "退出全部会话"),
			copyEntryValue("搜索", "按钮", "搜索", "button", "搜索按钮"),
			copyEntryValue("筛选", "按钮", "筛选", "button", "筛选按钮"),
			copyEntryValue("保存", "按钮", "保存", "button", "保存按钮"),
			copyEntryValue("修改", "按钮", "修改", "button", "修改按钮"),
			copyEntryValue("删除", "按钮", "删除", "button", "删除按钮"),
			copyEntryValue("取消", "按钮", "取消", "button", "取消按钮"),
			copyEntryValue("通过", "按钮", "通过", "button", "通过按钮"),
			copyEntryValue("拒绝", "按钮", "拒绝", "button", "拒绝按钮"),
			copyEntryValue("提交", "按钮", "提交", "button", "提交按钮"),
			copyEntryValue("详情", "按钮", "详情", "button", "详情按钮"),
			copyEntryValue("查看详情", "按钮", "查看详情", "button", "查看详情按钮"),
			copyEntryValue("新建申购", "按钮", "新建申购", "button", "新建申购按钮"),
			copyEntryValue("新建预约", "按钮", "新建预约", "button", "新建预约按钮"),
			copyEntryValue("提交预约", "按钮", "提交预约", "button", "提交预约按钮"),
			copyEntryValue("提交申领", "按钮", "提交申领", "button", "提交申领按钮"),
			copyEntryValue("提交申购", "按钮", "提交申购", "button", "提交申购按钮"),
			copyEntryValue("确认通过", "按钮", "确认通过", "button", "通过确认按钮"),
			copyEntryValue("确认拒绝", "按钮", "确认拒绝", "button", "拒绝确认按钮"),
			copyEntryValue("确认取消", "按钮", "确认取消", "button", "取消确认按钮"),
			copyEntryValue("导出流水", "按钮", "导出流水", "button", "导出流水按钮"),
			copyEntryValue("导出 CSV", "按钮", "导出 CSV", "button", "导出 CSV 按钮"),
			copyEntryValue("标记下单", "按钮", "标记下单", "button", "申购下单按钮"),
			copyEntryValue("到货入库", "按钮", "到货入库", "button", "申购入库按钮"),
			copyEntryValue("出库", "按钮", "出库", "button", "申领出库按钮"),
			copyEntryValue("签到", "按钮", "签到", "button", "预约签到按钮"),
			copyEntryValue("签退并入账", "按钮", "签退并入账", "button", "预约签退按钮"),
			copyEntryValue("返回系统首页", "按钮", "返回系统首页", "button", "返回首页按钮"),
			copyEntryValue("返回上一页", "按钮", "返回上一页", "button", "返回上一页按钮"),
			copyEntryValue("仪器预约大厅", "页面标题", "仪器预约大厅", "page", "首页主标题"),
			copyEntryValue("通知中心", "页面标题", "通知中心", "page", "通知中心标题"),
			copyEntryValue("财务管理", "页面标题", "财务管理", "page", "财务页标题"),
			copyEntryValue("平台配置中心", "页面标题", "平台配置中心", "page", "后台配置页标题"),
			copyEntryValue("系统基础配置", "页面标题", "系统基础配置", "page", "Footer 页面配置标题"),
			copyEntryValue("实验室运营系统", "品牌", "实验室运营系统", "brand", "系统名称"),
			copyEntryValue("实验室运营管理", "品牌", "实验室运营管理", "brand", "后台管理标题"),
			copyEntryValue("主导航", "辅助", "主导航", "meta", "主导航分组标题"),
			copyEntryValue("移动端主导航", "辅助", "移动端主导航", "meta", "移动端导航无障碍标题"),
			copyEntryValue("当前账号", "辅助", "当前账号", "meta", "个人菜单账号区标题"),
			copyEntryValue("通知", "辅助", "通知", "meta", "顶部通知入口标题"),
			copyEntryValue("主题", "辅助", "主题", "meta", "暗色模式切换按钮标题"),
			copyEntryValue("仪器分类", "筛选", "仪器分类", "filter", "顶部搜索分类选择标题"),
			copyEntryValue("全部分类", "筛选", "全部分类", "filter", "顶部搜索全部分类选项"),
			copyEntryValue("快速查找仪器...", "占位符", "快速查找仪器...", "placeholder", "顶部搜索框"),
			copyEntryValue("搜索仪器名称、型号、部门...", "占位符", "搜索仪器名称、型号、部门...", "placeholder", "首页筛选"),
			copyEntryValue("搜索设备、厂商、编码、仪器...", "占位符", "搜索设备、厂商、编码、仪器...", "placeholder", "IoT 搜索"),
			copyEntryValue("搜索问题、回答或背景...", "占位符", "搜索问题、回答或背景...", "placeholder", "AI 助手搜索"),
			copyEntryValue("搜索申请人、仪器、团队...", "占位符", "搜索申请人、仪器、团队...", "placeholder", "审批搜索"),
			copyEntryValue("搜索产品、申请人、用途", "占位符", "搜索产品、申请人、用途", "placeholder", "产品申领搜索"),
			copyEntryValue("搜索课程、讲师、仪器...", "占位符", "搜索课程、讲师、仪器...", "placeholder", "培训课程搜索"),
			copyEntryValue("搜索标题、作者、项目、任务...", "占位符", "搜索标题、作者、项目、任务...", "placeholder", "ELN 搜索"),
			copyEntryValue("搜索编号、名称、负责人...", "占位符", "搜索编号、名称、负责人...", "placeholder", "样本搜索"),
			copyEntryValue("仪器资源管理", "首页模块", "仪器资源管理", "module", "首页仪器资源卡片标题"),
			copyEntryValue("耗材资源管理", "首页模块", "耗材资源管理", "module", "首页耗材资源卡片标题"),
			copyEntryValue("申领管理", "首页模块", "申领管理", "module", "首页申领管理卡片标题"),
			copyEntryValue("耗材申购", "首页模块", "耗材申购", "module", "首页耗材申购卡片标题"),
			copyEntryValue("消息中心", "首页模块", "消息中心", "module", "首页消息中心卡片标题"),
			copyEntryValue("个人信息", "首页模块", "个人信息", "module", "首页个人信息卡片标题"),
			copyEntryValue("账户设置", "首页模块", "账户设置", "module", "首页账户设置卡片标题"),
			copyEntryValue("培训与准入总览", "首页模块", "培训与准入总览", "module", "首页培训总览卡片标题"),
			copyEntryValue("课程管理", "首页模块", "课程管理", "module", "首页课程管理卡片标题"),
			copyEntryValue("授权记录", "首页模块", "授权记录", "module", "首页授权记录卡片标题"),
			copyEntryValue("在线考试", "首页模块", "在线考试", "module", "首页在线考试卡片标题"),
			copyEntryValue("空间资源", "首页模块", "空间资源", "module", "首页空间资源卡片标题"),
			copyEntryValue("LIMS 检测任务", "首页模块", "LIMS 检测任务", "module", "首页 LIMS 卡片标题"),
			copyEntryValue("ELN 实验记录", "首页模块", "ELN 实验记录", "module", "首页 ELN 卡片标题"),
			copyEntryValue("样本管理", "首页模块", "样本管理", "module", "首页样本卡片标题"),
			copyEntryValue("IoT 设备中心", "首页模块", "IoT 设备中心", "module", "首页 IoT 卡片标题"),
			copyEntryValue("AI 助手", "首页模块", "AI 助手", "module", "首页 AI 卡片标题"),
			copyEntryValue("数据中台", "首页模块", "数据中台", "module", "首页数据中台卡片标题"),
			copyEntryValue("我的申请", "首页模块", "我的申请", "module", "首页工作台卡片标题"),
			copyEntryValue("预约记录", "首页模块", "预约记录", "module", "首页预约卡片标题"),
			copyEntryValue("审批中心", "首页模块", "审批中心", "module", "首页审批卡片标题"),
			copyEntryValue("财务管理", "首页模块", "财务管理", "module", "首页财务卡片标题"),
			copyEntryValue("工作概览", "首页模块", "工作概览", "module", "首页管理工作台卡片标题"),
			copyEntryValue("运营看板", "首页模块", "运营看板", "module", "首页运营看板卡片标题"),
			copyEntryValue("运营分析中心", "首页模块", "运营分析中心", "module", "首页运营分析卡片标题"),
			copyEntryValue("通知管理", "首页模块", "通知管理", "module", "首页通知管理卡片标题"),
			copyEntryValue("安全审计与合规", "首页模块", "安全审计与合规", "module", "首页安全审计卡片标题"),
			copyEntryValue("仪器资源后台", "首页模块", "仪器资源后台", "module", "首页仪器后台卡片标题"),
			copyEntryValue("资源管理后台", "首页模块", "资源管理后台", "module", "首页资源后台卡片标题"),
			copyEntryValue("工单与设备维护", "首页模块", "工单与设备维护", "module", "首页维护卡片标题"),
			copyEntryValue("用户管理", "首页模块", "用户管理", "module", "首页用户管理卡片标题"),
			copyEntryValue("平台配置中心", "首页模块", "平台配置中心", "module", "首页平台配置卡片标题"),
			copyEntryValue("组织架构管理", "首页模块", "组织架构管理", "module", "首页组织架构卡片标题"),
			copyEntryValue("租户配置", "首页模块", "租户配置", "module", "首页租户卡片标题"),
			copyEntryValue("财务模块开关", "首页模块", "财务模块开关", "module", "首页财务开关卡片标题"),
			copyEntryValue("通知通道配置", "首页模块", "通知通道配置", "module", "首页通知通道卡片标题"),
			copyEntryValue("第三方集成", "首页模块", "第三方集成", "module", "首页第三方集成卡片标题"),
			copyEntryValue("系统基础配置", "首页模块", "系统基础配置", "module", "首页 footer 配置卡片标题"),
			copyEntryValue("文案中心", "首页模块", "文案中心", "module", "首页文案中心卡片标题"),
		},
	}
}

func defaultCopySettings() CopySettings {
	value := defaultCopySettingsValue()
	return copySettingsFromValue(copySettingsKey, value, "system", time.Time{})
}

func normalizeCopySettingsValue(value copySettingsValue) copySettingsValue {
	defaults := defaultCopySettingsValue()
	if len(value.Entries) == 0 {
		return defaults
	}

	incoming := make(map[string]CopyEntry, len(value.Entries))
	for _, entry := range value.Entries {
		entry = normalizeCopyEntry(entry, CopyEntry{})
		if entry.Key == "" {
			continue
		}
		incoming[entry.Key] = entry
	}

	normalized := make([]CopyEntry, 0, len(defaults.Entries)+len(incoming))
	seen := make(map[string]struct{}, len(defaults.Entries))
	for _, entry := range defaults.Entries {
		if _, ok := seen[entry.Key]; ok {
			continue
		}
		if override, ok := incoming[entry.Key]; ok {
			normalized = append(normalized, normalizeCopyEntry(override, entry))
		} else {
			normalized = append(normalized, entry)
		}
		seen[entry.Key] = struct{}{}
	}
	for _, entry := range value.Entries {
		entry = normalizeCopyEntry(entry, CopyEntry{})
		if entry.Key == "" {
			continue
		}
		if _, ok := seen[entry.Key]; ok {
			continue
		}
		normalized = append(normalized, entry)
		seen[entry.Key] = struct{}{}
	}
	value.Entries = normalized
	return value
}

func normalizeCopyEntry(entry CopyEntry, fallback CopyEntry) CopyEntry {
	entry.Key = strings.TrimSpace(entry.Key)
	if entry.Key == "" {
		return CopyEntry{}
	}
	entry.Label = strings.TrimSpace(entry.Label)
	entry.Value = strings.TrimSpace(entry.Value)
	entry.Scope = strings.TrimSpace(entry.Scope)
	entry.Description = strings.TrimSpace(entry.Description)
	if entry.Label == "" {
		entry.Label = strings.TrimSpace(fallback.Label)
	}
	if entry.Label == "" {
		entry.Label = entry.Key
	}
	if entry.Value == "" {
		entry.Value = strings.TrimSpace(fallback.Value)
	}
	if entry.Value == "" {
		entry.Value = entry.Key
	}
	if entry.Scope == "" {
		entry.Scope = strings.TrimSpace(fallback.Scope)
	}
	if entry.Scope == "" {
		entry.Scope = "custom"
	}
	if entry.Description == "" {
		entry.Description = strings.TrimSpace(fallback.Description)
	}
	return entry
}

func copySettingsFromValue(key string, value copySettingsValue, updatedBy string, updatedAt time.Time) CopySettings {
	entries := make([]CopyEntry, 0, len(value.Entries))
	for _, entry := range value.Entries {
		if normalized := normalizeCopyEntry(entry, CopyEntry{}); normalized.Key != "" {
			entries = append(entries, normalized)
		}
	}
	return CopySettings{
		Key:       key,
		Entries:   entries,
		UpdatedBy: updatedBy,
		UpdatedAt: updatedAt,
	}
}

func copySettingsValueFromSettings(settings CopySettings) copySettingsValue {
	entries := make([]CopyEntry, 0, len(settings.Entries))
	for _, entry := range settings.Entries {
		if normalized := normalizeCopyEntry(entry, CopyEntry{}); normalized.Key != "" {
			entries = append(entries, normalized)
		}
	}
	return copySettingsValue{Entries: entries}
}

func defaultAccessControlSettingsValue() accessControlSettingsValue {
	return accessControlSettingsValue{
		Vendor:                 "hikvision",
		AutoGrantOnApproval:    true,
		AutoRevokeOnCompletion: true,
	}
}

func normalizeAccessControlSettingsValue(value accessControlSettingsValue) accessControlSettingsValue {
	defaults := defaultAccessControlSettingsValue()
	if value == (accessControlSettingsValue{}) {
		return defaults
	}
	value.Vendor = strings.TrimSpace(strings.ToLower(value.Vendor))
	if value.Vendor == "" {
		value.Vendor = defaults.Vendor
	}
	value.Endpoint = strings.TrimSpace(value.Endpoint)
	value.ClientID = strings.TrimSpace(value.ClientID)
	value.ClientSecret = strings.TrimSpace(value.ClientSecret)
	value.AccessGroup = strings.TrimSpace(value.AccessGroup)
	return value
}

func accessControlSettingsFromValue(value accessControlSettingsValue, updatedBy string, updatedAt time.Time) AccessControlSettings {
	return AccessControlSettings{
		Enabled:                value.Enabled,
		Vendor:                 value.Vendor,
		Endpoint:               value.Endpoint,
		ClientID:               value.ClientID,
		AccessGroup:            value.AccessGroup,
		AutoGrantOnApproval:    value.AutoGrantOnApproval,
		AutoRevokeOnCompletion: value.AutoRevokeOnCompletion,
		ClientSecretConfigured: value.ClientSecret != "",
		UpdatedBy:              updatedBy,
		UpdatedAt:              updatedAt,
	}
}

func accessControlSettingsValueFromSettings(settings AccessControlSettings) accessControlSettingsValue {
	return accessControlSettingsValue{
		Enabled:                settings.Enabled,
		Vendor:                 settings.Vendor,
		Endpoint:               settings.Endpoint,
		ClientID:               settings.ClientID,
		AccessGroup:            settings.AccessGroup,
		AutoGrantOnApproval:    settings.AutoGrantOnApproval,
		AutoRevokeOnCompletion: settings.AutoRevokeOnCompletion,
	}
}

type accessControlBinding struct {
	group string
	point string
}

func (r *Repository) accessControlBindingForReservation(ctx context.Context, reservation Reservation, settings accessControlSettingsValue) (accessControlBinding, bool) {
	instrument, err := r.Instrument(ctx, reservation.InstrumentID)
	if err != nil {
		slog.Warn("resolve instrument access control binding", "reservation_id", reservation.ID, "instrument_id", reservation.InstrumentID, "error", err)
		return accessControlBinding{}, false
	}
	if !instrument.AccessControlEnabled {
		return accessControlBinding{}, false
	}
	group := strings.TrimSpace(instrument.AccessControlGroup)
	if group == "" {
		group = strings.TrimSpace(settings.AccessGroup)
	}
	if group == "" {
		slog.Warn("instrument access control binding missing group", "reservation_id", reservation.ID, "instrument_id", reservation.InstrumentID)
		return accessControlBinding{}, false
	}
	return accessControlBinding{
		group: group,
		point: strings.TrimSpace(instrument.AccessControlPoint),
	}, true
}

func (r *Repository) emitAccessControlGrant(ctx context.Context, reservation Reservation) {
	settings, err := r.accessControlSettingsValue(ctx)
	if err != nil || !settings.Enabled || !settings.AutoGrantOnApproval {
		return
	}
	binding, ok := r.accessControlBindingForReservation(ctx, reservation, settings)
	if !ok {
		return
	}
	r.enqueueEvent(ctx, "access_control.authorization_granted", map[string]any{
		"reservationId":  reservation.ID,
		"tenantId":       reservation.TenantID,
		"instrumentId":   reservation.InstrumentID,
		"instrumentName": reservation.InstrumentName,
		"userId":         reservation.UserID,
		"userName":       reservation.UserName,
		"groupName":      reservation.GroupName,
		"startTime":      reservation.StartTime.UTC().Format(time.RFC3339),
		"endTime":        reservation.EndTime.UTC().Format(time.RFC3339),
		"vendor":         settings.Vendor,
		"accessGroup":    binding.group,
		"accessPoint":    binding.point,
	})
}

func (r *Repository) emitAccessControlRevoke(ctx context.Context, reservation Reservation, reason string) {
	settings, err := r.accessControlSettingsValue(ctx)
	if err != nil || !settings.Enabled || !settings.AutoRevokeOnCompletion {
		return
	}
	binding, ok := r.accessControlBindingForReservation(ctx, reservation, settings)
	if !ok {
		return
	}
	r.enqueueEvent(ctx, "access_control.authorization_revoked", map[string]any{
		"reservationId":  reservation.ID,
		"tenantId":       reservation.TenantID,
		"instrumentId":   reservation.InstrumentID,
		"instrumentName": reservation.InstrumentName,
		"userId":         reservation.UserID,
		"userName":       reservation.UserName,
		"groupName":      reservation.GroupName,
		"status":         reservation.Status,
		"reason":         reason,
		"vendor":         settings.Vendor,
		"accessGroup":    binding.group,
		"accessPoint":    binding.point,
	})
}

func dingTalkOAuthURL(settings dingTalkSettingsValue, state string) string {
	query := url.Values{}
	query.Set("redirect_uri", settings.OAuthRedirectURI)
	query.Set("response_type", "code")
	query.Set("client_id", settings.ClientID)
	query.Set("scope", "openid")
	query.Set("state", state)
	query.Set("prompt", "consent")
	return "https://login.dingtalk.com/oauth2/auth?" + query.Encode()
}

func generatedDingTalkEventCallbackURL(baseURI string, tenantCode string) string {
	baseURI = strings.TrimSpace(baseURI)
	tenantCode = strings.TrimSpace(tenantCode)
	if baseURI == "" || tenantCode == "" {
		return ""
	}
	parsed, err := url.Parse(baseURI)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host + "/api/dingtalk/events/" + url.PathEscape(tenantCode)
}

func (r *Repository) saveDingTalkOAuthState(ctx context.Context, tenantID string, userID string, state string) error {
	if r.redis == nil {
		return errors.New("redis is not configured")
	}
	return r.redis.Set(ctx, dingTalkOAuthStateKey(tenantID, userID, state), "1", 10*time.Minute).Err()
}

func (r *Repository) consumeDingTalkOAuthState(ctx context.Context, tenantID string, userID string, state string) error {
	state = strings.TrimSpace(state)
	if state == "" {
		return clientError("dingtalk oauth state is required")
	}
	if r.redis == nil {
		return errors.New("redis is not configured")
	}
	key := dingTalkOAuthStateKey(tenantID, userID, state)
	value, err := r.redis.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) || value == "" {
		return clientError("dingtalk oauth state is invalid or expired")
	}
	if err != nil {
		return err
	}
	return nil
}

func dingTalkOAuthStateKey(tenantID string, userID string, state string) string {
	return "lirs:dingtalk:oauth:" + tenantID + ":" + userID + ":" + state
}

func (r *Repository) saveDingTalkWebLoginState(ctx context.Context, tenantID string, state string) error {
	if r.redis == nil {
		return errors.New("redis is not configured")
	}
	return r.redis.Set(ctx, dingTalkWebLoginStateKey(state), strings.TrimSpace(tenantID), 10*time.Minute).Err()
}

func (r *Repository) consumeDingTalkWebLoginState(ctx context.Context, state string) (string, error) {
	state = strings.TrimSpace(state)
	if state == "" {
		return "", clientError("dingtalk login state is required")
	}
	if r.redis == nil {
		return "", errors.New("redis is not configured")
	}
	key := dingTalkWebLoginStateKey(state)
	value, err := r.redis.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) || value == "" {
		return "", clientError("dingtalk login state is invalid or expired")
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func dingTalkWebLoginStateKey(state string) string {
	return "lirs:dingtalk:web_login:" + tokenHash(state)
}

type dingTalkLoginBindingIntentValue struct {
	TenantID     string           `json:"tenantId"`
	TenantCode   string           `json:"tenantCode"`
	Identity     dingTalkIdentity `json:"identity"`
	DingTalkName string           `json:"dingTalkName"`
}

func (r *Repository) saveDingTalkLoginBindingIntent(ctx context.Context, value dingTalkLoginBindingIntentValue) (string, error) {
	if r.redis == nil {
		return "", errors.New("redis is not configured")
	}
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	if err := r.redis.Set(ctx, dingTalkLoginBindingIntentKey(token), raw, 10*time.Minute).Err(); err != nil {
		return "", err
	}
	return token, nil
}

func (r *Repository) consumeDingTalkLoginBindingIntent(ctx context.Context, token string) (dingTalkLoginBindingIntentValue, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return dingTalkLoginBindingIntentValue{}, clientError("dingtalk binding token is required")
	}
	if r.redis == nil {
		return dingTalkLoginBindingIntentValue{}, errors.New("redis is not configured")
	}
	raw, err := r.redis.GetDel(ctx, dingTalkLoginBindingIntentKey(token)).Bytes()
	if errors.Is(err, redis.Nil) || len(raw) == 0 {
		return dingTalkLoginBindingIntentValue{}, clientError("dingtalk binding token is invalid or expired")
	}
	if err != nil {
		return dingTalkLoginBindingIntentValue{}, err
	}
	var value dingTalkLoginBindingIntentValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return dingTalkLoginBindingIntentValue{}, err
	}
	return value, nil
}

func dingTalkLoginBindingIntentKey(token string) string {
	return "lirs:dingtalk:login_binding:" + tokenHash(token)
}

func (r *Repository) dingTalkIdentityByAuthCode(ctx context.Context, settings dingTalkSettingsValue, code string) (dingTalkIdentity, error) {
	userToken, err := r.dingTalkUserAccessToken(ctx, settings, code)
	if err != nil {
		return dingTalkIdentity{}, err
	}
	identity, err := r.dingTalkUserMe(ctx, userToken)
	if err != nil {
		return dingTalkIdentity{}, err
	}
	if identity.UnionID == "" {
		return dingTalkIdentity{}, clientError("dingtalk union id is required")
	}
	return r.dingTalkIdentityByUnionID(ctx, settings, identity)
}

func (r *Repository) dingTalkIdentityByQuickAuthCode(ctx context.Context, settings dingTalkSettingsValue, code string) (dingTalkIdentity, error) {
	token, err := r.dingTalkAppAccessToken(ctx, settings)
	if err != nil {
		return dingTalkIdentity{}, err
	}
	var response struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Result  struct {
			UserID   string `json:"userid"`
			UnionID  string `json:"unionid"`
			DeviceID string `json:"device_id"`
			Sys      bool   `json:"sys"`
		} `json:"result"`
		UserID   string `json:"userid"`
		UnionID  string `json:"unionid"`
		DeviceID string `json:"device_id"`
		Sys      bool   `json:"sys"`
	}
	payload := map[string]string{"code": strings.TrimSpace(code)}
	if err := r.dingTalkPost(ctx, "https://oapi.dingtalk.com/topapi/v2/user/getuserinfo?access_token="+url.QueryEscape(token), payload, &response); err != nil {
		return dingTalkIdentity{}, err
	}
	if response.ErrCode != 0 {
		return dingTalkIdentity{}, fmt.Errorf("dingtalk quick login failed: %s", response.ErrMsg)
	}
	userID := firstNonEmpty(response.Result.UserID, response.UserID)
	unionID := firstNonEmpty(response.Result.UnionID, response.UnionID)
	if userID == "" {
		return dingTalkIdentity{}, clientError("dingtalk user id is required")
	}
	identity, err := r.dingTalkIdentityByUserID(ctx, settings, userID)
	if err != nil {
		return dingTalkIdentity{UserID: userID, UnionID: unionID}, nil
	}
	if identity.UnionID == "" {
		identity.UnionID = unionID
	}
	return identity, nil
}

func (r *Repository) dingTalkIdentityByUserID(ctx context.Context, settings dingTalkSettingsValue, userID string) (dingTalkIdentity, error) {
	token, err := r.dingTalkAppAccessToken(ctx, settings)
	if err != nil {
		return dingTalkIdentity{}, err
	}
	var response struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Result  struct {
			UserID  string `json:"userid"`
			UnionID string `json:"unionid"`
			Name    string `json:"name"`
			Mobile  string `json:"mobile"`
		} `json:"result"`
	}
	payload := map[string]any{"userid": userID, "language": "zh_CN"}
	if err := r.dingTalkPost(ctx, "https://oapi.dingtalk.com/topapi/v2/user/get?access_token="+url.QueryEscape(token), payload, &response); err != nil {
		return dingTalkIdentity{}, err
	}
	if response.ErrCode != 0 {
		return dingTalkIdentity{}, fmt.Errorf("dingtalk user lookup failed: %s", response.ErrMsg)
	}
	return dingTalkIdentity{UserID: firstNonEmpty(response.Result.UserID, userID), UnionID: response.Result.UnionID, Name: response.Result.Name, Mobile: response.Result.Mobile}, nil
}

func (r *Repository) dingTalkIdentityByUnionID(ctx context.Context, settings dingTalkSettingsValue, fallback dingTalkIdentity) (dingTalkIdentity, error) {
	token, err := r.dingTalkAppAccessToken(ctx, settings)
	if err != nil {
		return dingTalkIdentity{}, err
	}
	var response struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Result  struct {
			UserID string `json:"userid"`
		} `json:"result"`
	}
	payload := map[string]string{"unionid": fallback.UnionID}
	if err := r.dingTalkPost(ctx, "https://oapi.dingtalk.com/topapi/user/getbyunionid?access_token="+url.QueryEscape(token), payload, &response); err != nil {
		return dingTalkIdentity{}, err
	}
	if response.ErrCode != 0 || response.Result.UserID == "" {
		return dingTalkIdentity{}, fmt.Errorf("dingtalk union id lookup failed: %s", response.ErrMsg)
	}
	identity, err := r.dingTalkIdentityByUserID(ctx, settings, response.Result.UserID)
	if err != nil {
		return dingTalkIdentity{UserID: response.Result.UserID, UnionID: fallback.UnionID, Name: fallback.Name, Mobile: fallback.Mobile}, nil
	}
	if identity.UnionID == "" {
		identity.UnionID = fallback.UnionID
	}
	if identity.Name == "" {
		identity.Name = fallback.Name
	}
	if identity.Mobile == "" {
		identity.Mobile = fallback.Mobile
	}
	return identity, nil
}

func (r *Repository) dingTalkAppAccessToken(ctx context.Context, settings dingTalkSettingsValue) (string, error) {
	var response struct {
		AccessToken string `json:"access_token"`
		ExpireIn    int    `json:"expires_in"`
	}
	payload := map[string]string{
		"client_id":     settings.ClientID,
		"client_secret": settings.ClientSecret,
		"grant_type":    "client_credentials",
	}
	endpoint := "https://api.dingtalk.com/v1.0/oauth2/" + url.PathEscape(settings.CorpID) + "/token"
	if err := r.dingTalkPost(ctx, endpoint, payload, &response); err != nil {
		return "", err
	}
	if response.AccessToken == "" {
		return "", clientError("dingtalk access token is empty")
	}
	return response.AccessToken, nil
}

func (r *Repository) dingTalkUserAccessToken(ctx context.Context, settings dingTalkSettingsValue, code string) (string, error) {
	var response struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int    `json:"expireIn"`
	}
	payload := map[string]string{
		"clientId":     settings.ClientID,
		"clientSecret": settings.ClientSecret,
		"code":         code,
		"grantType":    "authorization_code",
	}
	if err := r.dingTalkPost(ctx, "https://api.dingtalk.com/v1.0/oauth2/userAccessToken", payload, &response); err != nil {
		return "", err
	}
	if response.AccessToken == "" {
		return "", clientError("dingtalk user access token is empty")
	}
	return response.AccessToken, nil
}

func (r *Repository) dingTalkUserMe(ctx context.Context, userAccessToken string) (dingTalkIdentity, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.dingtalk.com/v1.0/contact/users/me", nil)
	if err != nil {
		return dingTalkIdentity{}, err
	}
	request.Header.Set("x-acs-dingtalk-access-token", userAccessToken)
	var response struct {
		UnionID string `json:"unionId"`
		OpenID  string `json:"openId"`
		Nick    string `json:"nick"`
		Mobile  string `json:"mobile"`
	}
	if err := r.dingTalkDo(request, &response); err != nil {
		return dingTalkIdentity{}, err
	}
	return dingTalkIdentity{UserID: firstNonEmpty(response.OpenID, response.UnionID), UnionID: response.UnionID, Name: response.Nick, Mobile: response.Mobile}, nil
}

func (r *Repository) dingTalkBoundUser(ctx context.Context, tenantID string, userID string) (User, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return User{}, clientError("dingtalk test user is required")
	}
	var user User
	err := r.db.QueryRow(ctx, `
SELECT u.id::text, u.tenant_id::text, t.name, t.code, u.name, u.email, u.phone, u.department, u.group_name, u.role, u.status, u.email_verified,
       u.dingtalk_user_id, u.dingtalk_union_id, u.dingtalk_name, u.dingtalk_user_id <> '',
       t.finance_enabled, u.auth_epoch
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE u.id = $1
  AND u.tenant_id = $2::uuid
  AND u.status = 'active'
`, userID, tenantID).Scan(&user.ID, &user.TenantID, &user.TenantName, &user.TenantCode, &user.Name, &user.Email, &user.Phone, &user.Department, &user.GroupName, &user.Role, &user.Status, &user.EmailVerified, &user.DingTalkUserID, &user.DingTalkUnionID, &user.DingTalkName, &user.DingTalkBound, &user.FinanceEnabled, &user.AuthEpoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, clientError("dingtalk test user is not active in current tenant")
	}
	if err != nil {
		return User{}, err
	}
	if strings.TrimSpace(user.DingTalkUserID) == "" {
		return User{}, clientError("dingtalk test user is not bound")
	}
	return user, nil
}

func (r *Repository) HandleDingTalkEventCallback(ctx context.Context, input DingTalkEventCallbackInput) (DingTalkEventCallbackResponse, error) {
	tenant, err := r.resolveActiveTenant(ctx, input.TenantID, input.TenantCode)
	if err != nil {
		return DingTalkEventCallbackResponse{}, err
	}
	settings, _, err := r.readDingTalkSettingsByTenantID(ctx, tenant.ID)
	if err != nil {
		return DingTalkEventCallbackResponse{}, err
	}
	if !settings.Enabled || settings.EventAesKey == "" || settings.EventToken == "" {
		return DingTalkEventCallbackResponse{}, clientError("dingtalk event callback is not configured")
	}
	input.Signature = strings.TrimSpace(input.Signature)
	input.Timestamp = strings.TrimSpace(input.Timestamp)
	input.Nonce = strings.TrimSpace(input.Nonce)
	input.Encrypt = strings.TrimSpace(input.Encrypt)
	if input.Signature == "" || input.Timestamp == "" || input.Nonce == "" || input.Encrypt == "" {
		return DingTalkEventCallbackResponse{}, clientError("dingtalk event callback input is invalid")
	}
	expectedSignature := dingTalkEventSignature(settings.EventToken, input.Timestamp, input.Nonce, input.Encrypt)
	if subtle.ConstantTimeCompare([]byte(input.Signature), []byte(expectedSignature)) != 1 {
		return DingTalkEventCallbackResponse{}, clientError("dingtalk event signature is invalid")
	}
	event, err := decryptDingTalkEvent(input.Encrypt, settings.EventAesKey, settings.CorpID)
	if err != nil {
		return DingTalkEventCallbackResponse{}, err
	}
	r.handleDingTalkHTTPEvent(WithTenantContext(ctx, TenantContext{TenantID: tenant.ID, TenantName: tenant.Name, FinanceEnabled: tenant.FinanceEnabled}), tenant, event)
	responseNonce, err := randomToken()
	if err != nil {
		return DingTalkEventCallbackResponse{}, err
	}
	responseTimestamp := strconv.FormatInt(time.Now().Unix(), 10)
	responseEncrypt, err := encryptDingTalkEvent("success", settings.EventAesKey, settings.CorpID)
	if err != nil {
		return DingTalkEventCallbackResponse{}, err
	}
	return DingTalkEventCallbackResponse{
		MsgSignature: dingTalkEventSignature(settings.EventToken, responseTimestamp, responseNonce, responseEncrypt),
		TimeStamp:    responseTimestamp,
		Nonce:        responseNonce,
		Encrypt:      responseEncrypt,
	}, nil
}

func (r *Repository) handleDingTalkHTTPEvent(ctx context.Context, tenant Tenant, event map[string]any) {
	eventType := stringFromAny(event["EventType"])
	corpID := stringFromAny(event["CorpId"])
	userID := firstStringFromAny(event["UserId"], event["UserID"], event["userid"])
	slog.Info("收到钉钉 HTTP 事件", "tenantId", tenant.ID, "tenantCode", tenant.Code, "eventType", eventType, "corpId", corpID, "userId", userID)
	r.enqueueEvent(ctx, "dingtalk.http_event", map[string]any{
		"tenantId":   tenant.ID,
		"tenantCode": tenant.Code,
		"eventType":  eventType,
		"corpId":     corpID,
		"userId":     userID,
		"payload":    event,
	})
}

func dingTalkEventSignature(token string, timestamp string, nonce string, encrypt string) string {
	parts := []string{token, timestamp, nonce, encrypt}
	sort.Strings(parts)
	sum := sha1Bytes(strings.Join(parts, ""))
	return hex.EncodeToString(sum)
}

func sha1Bytes(value string) []byte {
	hash := sha1.New()
	_, _ = hash.Write([]byte(value))
	return hash.Sum(nil)
}

func decryptDingTalkEvent(encrypted string, aesKey string, expectedCorpID string) (map[string]any, error) {
	plain, err := dingTalkCryptPayload(encrypted, aesKey, false)
	if err != nil {
		return nil, err
	}
	if len(plain) < 20 {
		return nil, clientError("dingtalk event payload is invalid")
	}
	msgLen := int(plain[16])<<24 | int(plain[17])<<16 | int(plain[18])<<8 | int(plain[19])
	if msgLen < 0 || 20+msgLen > len(plain) {
		return nil, clientError("dingtalk event payload length is invalid")
	}
	msg := []byte(plain[20 : 20+msgLen])
	receiveID := plain[20+msgLen:]
	if expectedCorpID != "" && receiveID != "" && receiveID != expectedCorpID {
		return nil, clientError("dingtalk event corp id is invalid")
	}
	var event map[string]any
	if err := json.Unmarshal(msg, &event); err != nil {
		return nil, err
	}
	return event, nil
}

func encryptDingTalkEvent(message string, aesKey string, corpID string) (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	msg := []byte(message)
	payload := make([]byte, 0, 20+len(msg)+len(corpID))
	payload = append(payload, random...)
	payload = append(payload, byte(len(msg)>>24), byte(len(msg)>>16), byte(len(msg)>>8), byte(len(msg)))
	payload = append(payload, msg...)
	payload = append(payload, []byte(corpID)...)
	return dingTalkCryptPayload(string(payload), aesKey, true)
}

func dingTalkCryptPayload(payload string, aesKey string, encrypt bool) (string, error) {
	key, err := base64.StdEncoding.DecodeString(aesKey + "=")
	if err != nil {
		return "", err
	}
	if len(key) != 32 {
		return "", clientError("dingtalk event aes key is invalid")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	iv := key[:aes.BlockSize]
	if encrypt {
		raw := pkcs7Pad([]byte(payload), aes.BlockSize)
		encrypted := make([]byte, len(raw))
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, raw)
		return base64.StdEncoding.EncodeToString(encrypted), nil
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || len(raw)%aes.BlockSize != 0 {
		return "", clientError("dingtalk event encrypted payload is invalid")
	}
	plain := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, raw)
	unpadded, err := pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return "", err
	}
	return string(unpadded), nil
}

func pkcs7Pad(raw []byte, blockSize int) []byte {
	padding := blockSize - len(raw)%blockSize
	if padding == 0 {
		padding = blockSize
	}
	result := make([]byte, 0, len(raw)+padding)
	result = append(result, raw...)
	for i := 0; i < padding; i++ {
		result = append(result, byte(padding))
	}
	return result
}

func pkcs7Unpad(raw []byte, blockSize int) ([]byte, error) {
	if len(raw) == 0 || len(raw)%blockSize != 0 {
		return nil, clientError("dingtalk event payload padding is invalid")
	}
	padding := int(raw[len(raw)-1])
	if padding == 0 || padding > blockSize || padding > len(raw) {
		return nil, clientError("dingtalk event payload padding is invalid")
	}
	for _, value := range raw[len(raw)-padding:] {
		if int(value) != padding {
			return nil, clientError("dingtalk event payload padding is invalid")
		}
	}
	return raw[:len(raw)-padding], nil
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func firstStringFromAny(values ...any) string {
	for _, value := range values {
		if text := stringFromAny(value); text != "" {
			return text
		}
	}
	return ""
}

func mustJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func mustJSONBytes(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

func randomNumericCode(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	limit := big.NewInt(10)
	var builder strings.Builder
	for i := 0; i < length; i++ {
		value, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		builder.WriteByte(byte('0' + value.Int64()))
	}
	return builder.String(), nil
}

func randomTenantCode() (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	return "org-" + token[:8], nil
}

func verificationCodeHash(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func defaultFooterSettingsValue() footerSettingsValue {
	return footerSettingsValue{
		BrandName:    "LIRS 2026 实验室运营系统",
		BrandTagline: "仪器预约、审批、使用、耗材、财务与审计闭环平台",
		Description:  "系统数据统一写入 PostgreSQL，登录会话、审批、库存、财务流水和审计记录均从数据库读取；Redis 用于缓存与事件队列。",
		Sections: []FooterSection{
			{
				Title: "技术栈",
				Lines: []string{
					"TypeScript / Next.js / React / Tailwind CSS / shadcn/ui / Lucide Icons",
					"Go / Gin / Hono / Zod / Drizzle ORM / PostgreSQL 15+ / Redis 7+",
				},
			},
			{
				Title: "运行信息",
				Lines: []string{
					"Hono API Gateway: 8090",
					"Go Core API: 8081",
				},
			},
		},
		Copyright: "© 2026 LIRS. All rights reserved.",
	}
}

func defaultFooterSettings() FooterSettings {
	value := defaultFooterSettingsValue()
	return footerSettingsFromValue(footerSettingsKey, value, "system", time.Time{})
}

func normalizeFooterSettingsValue(value footerSettingsValue) footerSettingsValue {
	defaults := defaultFooterSettingsValue()
	value.BrandName = strings.TrimSpace(value.BrandName)
	if value.BrandName == "" {
		value.BrandName = defaults.BrandName
	}
	value.BrandTagline = strings.TrimSpace(value.BrandTagline)
	if value.BrandTagline == "" {
		value.BrandTagline = defaults.BrandTagline
	}
	value.BaseURL = strings.TrimRight(strings.TrimSpace(value.BaseURL), "/")
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		value.Description = defaults.Description
	}
	value.Copyright = strings.TrimSpace(value.Copyright)
	if value.Copyright == "" {
		value.Copyright = defaults.Copyright
	}

	sections := make([]FooterSection, 0, len(value.Sections))
	for idx, section := range value.Sections {
		title := strings.TrimSpace(section.Title)
		lines := make([]string, 0, len(section.Lines))
		for _, line := range section.Lines {
			line = strings.TrimSpace(line)
			if line != "" {
				lines = append(lines, line)
			}
		}
		if title == "" && len(lines) == 0 {
			continue
		}
		if title == "" {
			title = fmt.Sprintf("栏目 %d", idx+1)
		}
		sections = append(sections, FooterSection{Title: title, Lines: lines})
	}
	if len(sections) == 0 {
		sections = defaults.Sections
	}
	value.Sections = sections
	return value
}

func footerSettingsFromValue(key string, value footerSettingsValue, updatedBy string, updatedAt time.Time) FooterSettings {
	return FooterSettings{
		Key:          key,
		BrandName:    value.BrandName,
		BrandTagline: value.BrandTagline,
		BaseURL:      value.BaseURL,
		Description:  value.Description,
		Sections:     value.Sections,
		Copyright:    value.Copyright,
		UpdatedBy:    updatedBy,
		UpdatedAt:    updatedAt,
	}
}

func footerSettingsValueFromSettings(settings FooterSettings) footerSettingsValue {
	return footerSettingsValue{
		BrandName:    settings.BrandName,
		BrandTagline: settings.BrandTagline,
		BaseURL:      settings.BaseURL,
		Description:  settings.Description,
		Sections:     settings.Sections,
		Copyright:    settings.Copyright,
	}
}

func validSiteBaseURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && parsed.RawQuery == "" && parsed.Fragment == ""
}

func materialDetailURL(baseURL string, materialID string) string {
	materialID = strings.TrimSpace(materialID)
	if materialID == "" {
		return ""
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "/materials/" + url.PathEscape(materialID)
	}
	return baseURL + "/materials/" + url.PathEscape(materialID)
}

func copyEntryValue(key string, label string, value string, scope string, description string) CopyEntry {
	return CopyEntry{
		Key:         key,
		Label:       label,
		Value:       value,
		Scope:       scope,
		Description: description,
	}
}

func normalizeInstrument(input InstrumentInput) InstrumentInput {
	input.Name = strings.TrimSpace(input.Name)
	input.Category = strings.TrimSpace(input.Category)
	input.Department = strings.TrimSpace(input.Department)
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.Status = strings.TrimSpace(input.Status)
	input.Location = strings.TrimSpace(input.Location)
	input.Brand = strings.TrimSpace(input.Brand)
	input.Model = strings.TrimSpace(input.Model)
	input.AssetCode = strings.TrimSpace(input.AssetCode)
	input.AccessControlGroup = strings.TrimSpace(input.AccessControlGroup)
	input.AccessControlPoint = strings.TrimSpace(input.AccessControlPoint)
	input.Description = strings.TrimSpace(input.Description)
	input.TechnicalSpecs = strings.TrimSpace(input.TechnicalSpecs)
	input.BookingRule = strings.TrimSpace(input.BookingRule)
	input.MaintenanceSummary = strings.TrimSpace(input.MaintenanceSummary)
	if input.BookingRule == "" {
		input.BookingRule = "最小预约 1 小时；审批中时段会被锁定；使用前 2 小时可取消。"
	}
	if input.MaxBookingHours <= 0 {
		input.MaxBookingHours = 72
	}
	if input.MinAdvanceHours < 0 {
		input.MinAdvanceHours = 0
	}
	if input.CancelCutoffHours < 0 {
		input.CancelCutoffHours = 0
	}
	if input.CheckinWindowMins <= 0 {
		input.CheckinWindowMins = 30
	}
	if input.BookingWindowDays <= 0 {
		input.BookingWindowDays = 30
	}
	if input.BookingIntervalHours <= 0 {
		input.BookingIntervalHours = 1
	}
	if input.BookingIntervalHours > 12 {
		input.BookingIntervalHours = 12
	}
	input.ServiceStartHour, input.ServiceEndHour = normalizeServiceHours(input.ServiceStartHour, input.ServiceEndHour)
	return input
}

func normalizeMaterial(input MaterialInput) MaterialInput {
	input.Name = strings.TrimSpace(input.Name)
	input.ProductType = strings.TrimSpace(input.ProductType)
	input.Category = strings.TrimSpace(input.Category)
	input.Subcategory = strings.TrimSpace(input.Subcategory)
	input.Spec = strings.TrimSpace(input.Spec)
	input.Unit = strings.TrimSpace(input.Unit)
	input.Supplier = strings.TrimSpace(input.Supplier)
	input.Manufacturer = strings.TrimSpace(input.Manufacturer)
	input.BatchNo = strings.TrimSpace(input.BatchNo)
	input.CatalogNo = strings.TrimSpace(input.CatalogNo)
	input.CASNo = strings.TrimSpace(input.CASNo)
	input.Grade = strings.TrimSpace(input.Grade)
	input.Concentration = strings.TrimSpace(input.Concentration)
	input.ParentMaterialID = strings.TrimSpace(input.ParentMaterialID)
	input.DilutionFactor = strings.TrimSpace(input.DilutionFactor)
	input.PreparationMethod = strings.TrimSpace(input.PreparationMethod)
	input.StorageCondition = strings.TrimSpace(input.StorageCondition)
	input.StorageRoom = strings.TrimSpace(input.StorageRoom)
	input.StorageCabinet = strings.TrimSpace(input.StorageCabinet)
	input.StorageLayer = strings.TrimSpace(input.StorageLayer)
	input.StorageSlot = strings.TrimSpace(input.StorageSlot)
	input.TenderContract = strings.TrimSpace(input.TenderContract)
	input.ContractNo = strings.TrimSpace(input.ContractNo)
	input.Remark = strings.TrimSpace(input.Remark)
	input.CertificateURL = strings.TrimSpace(input.CertificateURL)
	input.StandardCertificateURL = strings.TrimSpace(input.StandardCertificateURL)
	input.AttachmentURL = strings.TrimSpace(input.AttachmentURL)
	input.QRCode = strings.TrimSpace(input.QRCode)
	input.PurchaseSerialNo = strings.TrimSpace(input.PurchaseSerialNo)
	input.ExpiresAt = strings.TrimSpace(input.ExpiresAt)
	input.OpenedAt = strings.TrimSpace(input.OpenedAt)
	input.Status = strings.TrimSpace(input.Status)
	if input.ProductType == "standard" {
		input.ParentMaterialID = ""
		input.DilutionFactor = ""
		input.PreparationMethod = ""
	}
	return input
}

func materialImportHeaderIndex(header []string) map[string]int {
	index := make(map[string]int, len(header))
	for i, name := range header {
		index[strings.TrimSpace(name)] = i
	}
	return index
}

func materialCSVValue(index map[string]int, row []string, names ...string) string {
	for _, name := range names {
		if position, ok := index[name]; ok && position < len(row) {
			return strings.TrimSpace(row[position])
		}
	}
	return ""
}

func materialCSVInt(index map[string]int, row []string, fallback int, names ...string) int {
	value := materialCSVValue(index, row, names...)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func materialCSVFloat(index map[string]int, row []string, fallback float64, names ...string) float64 {
	value := materialCSVValue(index, row, names...)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func materialCSVBool(index map[string]int, row []string, names ...string) bool {
	value := strings.ToLower(materialCSVValue(index, row, names...))
	return value == "是" || value == "true" || value == "1" || value == "yes"
}

func materialLooksLikeHeader(row []string) bool {
	joined := strings.Join(row, "|")
	return (strings.Contains(joined, "资源名称") && strings.Contains(joined, "一级目录") && strings.Contains(joined, "规格") && strings.Contains(joined, "单位")) ||
		(strings.Contains(joined, "类别") && strings.Contains(joined, "产品名") && strings.Contains(joined, "规格型号") && strings.Contains(joined, "数量单位"))
}

func materialInputFromCSVRow(index map[string]int, row []string) MaterialInput {
	procurementProject := materialCSVValue(index, row, "采购项目名称及编号", "采购项目", "采购项目及编号", "招标合同", "tenderContract")
	return normalizeMaterial(MaterialInput{
		Name:                   materialCSVValue(index, row, "资源名称", "产品名称", "产品名", "名称", "name"),
		ProductType:            productTypeCode(materialCSVValue(index, row, "资源类型", "产品分类", "产品类型", "类型", "类别", "productType")),
		Category:               materialCSVValue(index, row, "一级目录", "一级分类", "分类", "类别", "category"),
		Subcategory:            materialCSVValue(index, row, "二级目录", "二级分类", "子分类", "subcategory"),
		Spec:                   materialCSVValue(index, row, "规格", "规格型号", "spec"),
		Unit:                   materialCSVValue(index, row, "单位", "数量单位", "unit"),
		UnitPrice:              materialCSVFloat(index, row, 0, "单价", "unitPrice"),
		Stock:                  materialCSVInt(index, row, 0, "库存", "初始库存", "数量", "stock"),
		WarningLine:            materialCSVInt(index, row, 0, "低库存线", "库存告警线", "warningLine"),
		Supplier:               materialCSVValue(index, row, "供应商", "供应商名", "supplier"),
		Manufacturer:           materialCSVValue(index, row, "生产商", "manufacturer"),
		BatchNo:                materialCSVValue(index, row, "批号", "批次号", "batchNo"),
		CatalogNo:              materialCSVValue(index, row, "货号", "catalogNo"),
		CASNo:                  materialCSVValue(index, row, "CAS号", "CAS", "casNo"),
		Grade:                  materialCSVValue(index, row, "级别", "grade"),
		Concentration:          materialCSVValue(index, row, "浓度", "concentration"),
		ParentMaterialID:       materialCSVValue(index, row, "母液ID", "来源产品ID", "parentMaterialId"),
		DilutionFactor:         materialCSVValue(index, row, "稀释倍数", "dilutionFactor"),
		PreparationMethod:      materialCSVValue(index, row, "配制方法", "preparationMethod"),
		StorageCondition:       materialCSVValue(index, row, "保存条件", "storageCondition"),
		StorageRoom:            materialCSVValue(index, row, "库房/冰箱", "库房", "存放地", "storageRoom"),
		StorageCabinet:         materialCSVValue(index, row, "柜/架", "storageCabinet"),
		StorageLayer:           materialCSVValue(index, row, "层/盒", "storageLayer"),
		StorageSlot:            materialCSVValue(index, row, "孔位", "storageSlot"),
		TenderContract:         procurementProject,
		ContractNo:             firstNonEmpty(materialCSVValue(index, row, "合同序号", "contractNo"), procurementProject),
		Remark:                 materialCSVValue(index, row, "备注", "remark"),
		CertificateURL:         materialCSVValue(index, row, "资源证书地址", "产品证书地址", "certificateUrl"),
		StandardCertificateURL: materialCSVValue(index, row, "标准证书地址", "standardCertificateUrl"),
		AttachmentURL:          materialCSVValue(index, row, "附件地址", "attachmentUrl"),
		QRCode:                 materialCSVValue(index, row, "二维码编码", "qrCode"),
		ExpiresAt:              materialCSVValue(index, row, "有效期", "过期时间", "过期时间(yyyy/MM/dd)", "expiresAt"),
		OpenedAt:               materialCSVValue(index, row, "开封日期", "openedAt"),
		OpenExpireDays:         materialCSVInt(index, row, 0, "开封有效天数", "openExpireDays"),
		FreezeThawCount:        materialCSVInt(index, row, 0, "冻融次数", "freezeThawCount"),
		FreezeThawLimit:        materialCSVInt(index, row, 0, "冻融上限", "freezeThawLimit"),
		ApprovalRequired:       materialCSVBool(index, row, "是否需要审批", "approvalRequired"),
		NearExpiryDays:         materialCSVInt(index, row, 30, "临期预警天数", "nearExpiryDays"),
		Status:                 materialStatusCode(materialCSVValue(index, row, "状态", "status")),
	})
}

func normalizePurchasableMaterial(input PurchasableMaterialInput) PurchasableMaterialInput {
	input.IDNo = strings.TrimSpace(input.IDNo)
	input.SequenceNo = strings.TrimSpace(input.SequenceNo)
	input.ProcurementProjectID = strings.TrimSpace(input.ProcurementProjectID)
	input.ProcurementProject = strings.TrimSpace(input.ProcurementProject)
	input.ProjectName = strings.TrimSpace(input.ProjectName)
	input.Brand = strings.TrimSpace(input.Brand)
	input.Spec = strings.TrimSpace(input.Spec)
	input.Unit = strings.TrimSpace(input.Unit)
	input.Remark = strings.TrimSpace(input.Remark)
	input.TechnicalRequirement = strings.TrimSpace(input.TechnicalRequirement)
	input.MinSpec = strings.TrimSpace(input.MinSpec)
	input.Actor = strings.TrimSpace(input.Actor)
	return input
}

func normalizeProcurementProject(input ProcurementProjectInput) ProcurementProjectInput {
	input.Name = strings.TrimSpace(input.Name)
	input.ExpiresAt = strings.TrimSpace(input.ExpiresAt)
	input.Status = strings.TrimSpace(input.Status)
	input.Actor = strings.TrimSpace(input.Actor)
	return input
}

func (r *Repository) ensureProcurementProject(ctx context.Context, tx pgx.Tx, projectID string, name string) (string, string, error) {
	tenant := TenantFromContext(ctx)
	projectID = strings.TrimSpace(projectID)
	name = strings.TrimSpace(name)
	if projectID != "" {
		var foundName string
		query := `
SELECT name
FROM procurement_projects
WHERE id = $1
  AND status = 'active'
  AND ($2::boolean OR tenant_id = $3::uuid)
`
		var err error
		if tx != nil {
			err = tx.QueryRow(ctx, query, projectID, tenant.AllTenants, tenant.TenantID).Scan(&foundName)
		} else {
			err = r.db.QueryRow(ctx, query, projectID, tenant.AllTenants, tenant.TenantID).Scan(&foundName)
		}
		if err != nil {
			return "", "", err
		}
		return projectID, foundName, nil
	}
	if name == "" {
		return "", "", nil
	}
	query := `
INSERT INTO procurement_projects (tenant_id, name)
VALUES ($1, $2)
ON CONFLICT (tenant_id, name) DO UPDATE
SET updated_at = procurement_projects.updated_at
RETURNING id::text, name
`
	var id string
	var foundName string
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, tenant.TenantID, name).Scan(&id, &foundName)
	} else {
		err = r.db.QueryRow(ctx, query, tenant.TenantID, name).Scan(&id, &foundName)
	}
	if err != nil {
		return "", "", err
	}
	return id, foundName, nil
}

func ensureProcurementProjectsTx(ctx context.Context, tx pgx.Tx, tenantID string, items []PurchasableMaterialInput) (map[string]string, error) {
	names := make([]string, 0)
	seen := map[string]struct{}{}
	for _, item := range items {
		name := strings.TrimSpace(item.ProcurementProject)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	ids := make(map[string]string, len(names))
	for _, name := range names {
		var id string
		if err := tx.QueryRow(ctx, `
INSERT INTO procurement_projects (tenant_id, name)
VALUES ($1, $2)
ON CONFLICT (tenant_id, name) DO UPDATE
SET updated_at = procurement_projects.updated_at
RETURNING id::text
`, tenantID, name).Scan(&id); err != nil {
			return nil, err
		}
		ids[name] = id
	}
	return ids, nil
}

func purchasableMaterialInputFromRow(index map[string]int, row []string, currentProject string) PurchasableMaterialInput {
	projectName := materialCSVValue(index, row, "项目名称", "采购项目名称", "项目", "projectName")
	brand := materialCSVValue(index, row, "品牌", "brand")
	spec := materialCSVValue(index, row, "规格", "spec")
	unit := materialCSVValue(index, row, "单位", "unit")
	remark := materialCSVValue(index, row, "备注", "remark")
	price := materialCSVFloat(index, row, 0, "采购价（元）", "采购价", "采购价格", "purchasePrice")
	brand, spec, unit, remark, price = normalizePurchasableMaterialLooseColumns(brand, spec, unit, remark, price)
	return normalizePurchasableMaterial(PurchasableMaterialInput{
		IDNo:                 materialCSVValue(index, row, "ID号", "ID", "idNo"),
		SequenceNo:           materialCSVValue(index, row, "序号", "sequenceNo"),
		ProcurementProject:   firstNonEmpty(materialCSVValue(index, row, "采购项目名称及编号", "采购项目", "采购项目及编号", "procurementProject"), currentProject),
		ProjectName:          projectName,
		Brand:                brand,
		Spec:                 spec,
		Unit:                 unit,
		PurchasePrice:        price,
		Remark:               remark,
		TechnicalRequirement: materialCSVValue(index, row, "技术要求", "technicalRequirement"),
		MinSpec:              materialCSVValue(index, row, "最小规格", "minSpec"),
	})
}

func normalizePurchasableMaterialLooseColumns(brand string, spec string, unit string, remark string, price float64) (string, string, string, string, float64) {
	if brand == "" {
		brand = "未标明"
	}
	if spec == "" {
		spec = "未标明"
	}
	if unit == "" && purchasableMaterialLooksLikePrice(spec) {
		unit = "未标明"
	}
	if unit == "" && remark != "" && purchasableMaterialLooksLikePrice(remark) {
		unit = spec
		spec = "未标明"
		price = parsePurchasableMaterialPrice(remark, price)
		remark = ""
	}
	if unit == "" {
		unit = "未标明"
	}
	return brand, spec, unit, remark, price
}

func purchasableMaterialLooksLikePrice(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func parsePurchasableMaterialPrice(value string, fallback float64) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func purchasableMaterialLooksLikeHeader(row []string) bool {
	joined := strings.Join(row, "|")
	return strings.Contains(joined, "ID号") && strings.Contains(joined, "序号") && strings.Contains(joined, "项目名称")
}

func purchasableMaterialProjectHeader(row []string) string {
	visible := make([]string, 0, len(row))
	for _, cell := range row {
		cell = strings.TrimSpace(cell)
		if cell != "" {
			visible = append(visible, cell)
		}
	}
	if len(visible) != 1 {
		return ""
	}
	value := visible[0]
	if strings.Contains(value, "编号：") || strings.Contains(value, "采购项目") {
		return value
	}
	return ""
}

func purchasableMaterialLooksLikeNoteRow(row []string) bool {
	visible := make([]string, 0, len(row))
	for _, cell := range row {
		cell = strings.TrimSpace(cell)
		if cell != "" {
			visible = append(visible, cell)
		}
	}
	if len(visible) == 0 {
		return true
	}
	if len(visible) <= 2 {
		joined := strings.Join(visible, "")
		return strings.Contains(joined, "注") || strings.Contains(joined, "说明") || strings.Contains(joined, "以下空白") || strings.Contains(joined, "合计")
	}
	return false
}

func purchasableMaterialRowPreview(row []string) string {
	visible := make([]string, 0, len(row))
	for _, cell := range row {
		cell = strings.TrimSpace(cell)
		if cell != "" {
			visible = append(visible, cell)
		}
	}
	if len(visible) == 0 {
		return "空行"
	}
	preview := strings.Join(limitStrings(visible, 6), " | ")
	if len(preview) > 160 {
		return preview[:160] + "..."
	}
	return preview
}

func rowBlank(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func purchasableMaterialImportRecords(filename string, content []byte) ([][]string, error) {
	filename = normalizeImportFilename(filename)
	if strings.HasSuffix(filename, ".xlsx") || looksLikeXLSX(content) {
		workbook, err := excelize.OpenReader(bytes.NewReader(content))
		if err != nil {
			return nil, fmt.Errorf("无法读取 XLSX 文件：%w", err)
		}
		defer func() {
			_ = workbook.Close()
		}()
		sheets := workbook.GetSheetList()
		if len(sheets) == 0 {
			return nil, clientError("XLSX 文件没有工作表")
		}
		return workbook.GetRows(sheets[0])
	}
	if strings.HasSuffix(filename, ".xls") || looksLikeXLS(content) {
		return xlsImportRecords(content)
	}
	if filename != "" && !strings.HasSuffix(filename, ".csv") {
		return nil, fmt.Errorf("不支持的文件类型 %q，请上传 CSV、XLS 或 XLSX 文件", filename)
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(content), "\ufeff")))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("无法读取 CSV 文件：%w", err)
	}
	return records, nil
}

func normalizeImportFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	filename = strings.ReplaceAll(filename, "。", ".")
	return strings.ToLower(filename)
}

func xlsImportRecords(content []byte) ([][]string, error) {
	workbook, err := xls.OpenReader(bytes.NewReader(content), "utf-8")
	if err != nil {
		return nil, fmt.Errorf("无法读取 XLS 文件：%w", err)
	}
	if workbook == nil || workbook.NumSheets() == 0 {
		return nil, clientError("XLS 文件没有工作表")
	}
	sheet := workbook.GetSheet(0)
	if sheet == nil {
		return nil, clientError("XLS 文件没有工作表")
	}
	records := make([][]string, 0, int(sheet.MaxRow)+1)
	for rowIndex := 0; rowIndex <= int(sheet.MaxRow); rowIndex++ {
		row := xlsImportRow(sheet, rowIndex)
		if row == nil {
			records = append(records, nil)
			continue
		}
		lastCol := row.LastCol()
		if lastCol < 0 {
			records = append(records, nil)
			continue
		}
		values := make([]string, lastCol)
		for colIndex := 0; colIndex < lastCol; colIndex++ {
			values[colIndex] = strings.TrimSpace(row.Col(colIndex))
		}
		records = append(records, values)
	}
	return records, nil
}

func xlsImportRow(sheet *xls.WorkSheet, rowIndex int) (row *xls.Row) {
	defer func() {
		if recover() != nil {
			row = nil
		}
	}()
	return sheet.Row(rowIndex)
}

func looksLikeXLSX(content []byte) bool {
	return len(content) >= 4 && bytes.Equal(content[:4], []byte{'P', 'K', 0x03, 0x04})
}

func looksLikeXLS(content []byte) bool {
	return len(content) >= 8 && bytes.Equal(content[:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
}

func limitStrings(values []string, limit int) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func (r *Repository) purchasableMaterialIDByIDNo(ctx context.Context, idNo string) (string, error) {
	tenant := TenantFromContext(ctx)
	var id string
	err := r.db.QueryRow(ctx, `
SELECT id::text
FROM purchasable_materials
WHERE id_no = $1
  AND ($2::boolean OR tenant_id = $3::uuid)
ORDER BY updated_at DESC
LIMIT 1
`, strings.TrimSpace(idNo), tenant.AllTenants, tenant.TenantID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func productTypeCode(value string) string {
	switch strings.TrimSpace(value) {
	case "", "耗材", "consumable":
		return "consumable"
	case "普通试剂", "试剂", "reagent":
		return "reagent"
	case "标准物质", "标准品", "标准品/标准物质", "标准", "工作液", "混标", "standard", "working_solution", "mixed_standard":
		return "standard"
	default:
		return value
	}
}

func materialStatusCode(value string) string {
	switch strings.TrimSpace(value) {
	case "", "正常", "normal":
		return "normal"
	case "临期", "near_expiry":
		return "near_expiry"
	case "低库存", "low":
		return "low"
	case "过期", "已过期", "expired":
		return "expired"
	case "开封超期", "open_expired":
		return "open_expired"
	case "冻融超限", "freeze_thaw_exceeded":
		return "freeze_thaw_exceeded"
	case "损毁", "damaged":
		return "damaged"
	case "停用", "disabled":
		return "disabled"
	default:
		return value
	}
}

func topMaterialConsumption(items map[string]MaterialConsumptionRanking, limit int) []MaterialConsumptionRanking {
	values := make([]MaterialConsumptionRanking, 0, len(items))
	for _, item := range items {
		values = append(values, item)
	}
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j].Quantity > values[i].Quantity {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
	if len(values) > limit {
		return values[:limit]
	}
	return values
}

func validInstrumentStatus(status string) bool {
	switch status {
	case "available", "busy", "maintenance", "disabled":
		return true
	default:
		return false
	}
}

func validTenantStatus(status string) bool {
	switch status {
	case "active", "disabled":
		return true
	default:
		return false
	}
}

func validMaterialStatus(status string) bool {
	switch status {
	case "normal", "near_expiry", "low", "expired", "open_expired", "freeze_thaw_exceeded", "damaged", "disabled":
		return true
	default:
		return false
	}
}

func validMaterialProductType(productType string) bool {
	switch productType {
	case "consumable", "reagent", "standard":
		return true
	default:
		return false
	}
}

func validRole(role string) bool {
	switch role {
	case "unassigned", "student", "teacher", "researcher", "group_leader", "material_admin", "finance_admin", "tenant_admin", "lab_admin", "super_admin":
		return true
	default:
		return false
	}
}

func isScopedAdminRole(role string) bool {
	return role == "material_admin" || role == "finance_admin"
}

func isTenantAdminRole(role string) bool {
	return role == "tenant_admin" || role == "lab_admin" || role == "super_admin"
}

func isAdministratorRole(role string) bool {
	return isScopedAdminRole(role) || isTenantAdminRole(role)
}

func canActorManageUserRole(actorRole string, oldRole string, newRole string) bool {
	if actorRole == "super_admin" {
		return true
	}
	if actorRole == "tenant_admin" || actorRole == "lab_admin" {
		return !isTenantAdminRole(oldRole) && !isTenantAdminRole(newRole)
	}
	return !isAdministratorRole(oldRole) && !isAdministratorRole(newRole)
}

func validRegisterAccountType(accountType string) bool {
	switch accountType {
	case "user":
		return true
	default:
		return false
	}
}

func validUserStatus(status string) bool {
	switch status {
	case "pending_approval", "active", "disabled", "deleted":
		return true
	default:
		return false
	}
}

func validOrganizationUnitKind(kind string) bool {
	switch kind {
	case "department", "group":
		return true
	default:
		return false
	}
}

func (r *Repository) organizationUnitDeletionUsage(ctx context.Context, tx pgx.Tx, unit OrganizationUnit) (int, int, error) {
	tenant := TenantFromContext(ctx)
	var instrumentCount int
	var dependentCount int
	switch unit.Kind {
	case "department":
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM instruments WHERE department = $1 AND ($2::boolean OR tenant_id = $3::uuid)`, unit.Name, tenant.AllTenants, tenant.TenantID).Scan(&instrumentCount); err != nil {
			return 0, 0, err
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM users WHERE department = $1 AND ($2::boolean OR tenant_id = $3::uuid)`, unit.Name, tenant.AllTenants, tenant.TenantID).Scan(&dependentCount); err != nil {
			return 0, 0, err
		}
		var teamCount int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM organization_units WHERE kind = 'group' AND parent_name = $1 AND ($2::boolean OR tenant_id = $3::uuid)`, unit.Name, tenant.AllTenants, tenant.TenantID).Scan(&teamCount); err != nil {
			return 0, 0, err
		}
		dependentCount += teamCount
	case "group":
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM instruments WHERE group_name = $1 AND ($2::boolean OR tenant_id = $3::uuid)`, unit.Name, tenant.AllTenants, tenant.TenantID).Scan(&instrumentCount); err != nil {
			return 0, 0, err
		}
	default:
		return 0, 0, clientError("invalid organization unit kind")
	}
	return instrumentCount, dependentCount, nil
}

func (r *Repository) updateOrganizationUnitReferences(ctx context.Context, tx pgx.Tx, oldUnit OrganizationUnit, newUnit OrganizationUnit) error {
	if oldUnit.Name == newUnit.Name && oldUnit.ParentName == newUnit.ParentName {
		return nil
	}
	tenant := TenantFromContext(ctx)
	switch newUnit.Kind {
	case "department":
		if oldUnit.Name != newUnit.Name {
			if _, err := tx.Exec(ctx, `UPDATE users SET department = $2, updated_at = now() WHERE department = $1 AND ($3::boolean OR tenant_id = $4::uuid)`, oldUnit.Name, newUnit.Name, tenant.AllTenants, tenant.TenantID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE instruments SET department = $2 WHERE department = $1 AND ($3::boolean OR tenant_id = $4::uuid)`, oldUnit.Name, newUnit.Name, tenant.AllTenants, tenant.TenantID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE notifications SET department = $2 WHERE department = $1 AND ($3::boolean OR tenant_id = $4::uuid)`, oldUnit.Name, newUnit.Name, tenant.AllTenants, tenant.TenantID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE organization_units SET parent_name = $2, updated_at = now() WHERE kind = 'group' AND parent_name = $1 AND ($3::boolean OR tenant_id = $4::uuid)`, oldUnit.Name, newUnit.Name, tenant.AllTenants, tenant.TenantID); err != nil {
				return err
			}
		}
	case "group":
		if oldUnit.Name != newUnit.Name {
			if _, err := tx.Exec(ctx, `UPDATE instruments SET group_name = $2 WHERE group_name = $1 AND ($3::boolean OR tenant_id = $4::uuid)`, oldUnit.Name, newUnit.Name, tenant.AllTenants, tenant.TenantID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE reservations SET group_name = $2 WHERE group_name = $1 AND ($3::boolean OR tenant_id = $4::uuid)`, oldUnit.Name, newUnit.Name, tenant.AllTenants, tenant.TenantID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE notifications SET group_name = $2 WHERE group_name = $1 AND ($3::boolean OR tenant_id = $4::uuid)`, oldUnit.Name, newUnit.Name, tenant.AllTenants, tenant.TenantID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE material_requests SET group_name = $2 WHERE group_name = $1 AND ($3::boolean OR tenant_id = $4::uuid)`, oldUnit.Name, newUnit.Name, tenant.AllTenants, tenant.TenantID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE material_purchases SET group_name = $2 WHERE group_name = $1 AND ($3::boolean OR tenant_id = $4::uuid)`, oldUnit.Name, newUnit.Name, tenant.AllTenants, tenant.TenantID); err != nil {
				return err
			}
		}
		if oldUnit.ParentName != newUnit.ParentName && newUnit.ParentName != "" {
			if _, err := tx.Exec(ctx, `
UPDATE instruments
SET department = $2
WHERE group_name = $1
  AND ($3 = '' OR department = $3)
  AND ($4::boolean OR tenant_id = $5::uuid)
`, newUnit.Name, newUnit.ParentName, oldUnit.ParentName, tenant.AllTenants, tenant.TenantID); err != nil {
				return err
			}
		}
	default:
		return clientError("invalid organization unit kind")
	}
	return nil
}

func reservationServiceLocation() *time.Location {
	return time.FixedZone("Asia/Shanghai", 8*60*60)
}

func normalizeServiceHours(startHour int, endHour int) (int, int) {
	if startHour < 0 || startHour > 23 {
		startHour = 0
	}
	if endHour <= startHour || endHour > 24 {
		endHour = 24
	}
	return startHour, endHour
}

func isWithinServiceHours(start time.Time, end time.Time, serviceStartHour int, serviceEndHour int) bool {
	if !end.After(start) {
		return false
	}
	serviceStartHour, serviceEndHour = normalizeServiceHours(serviceStartHour, serviceEndHour)
	loc := reservationServiceLocation()
	localStart := start.In(loc)
	localEnd := end.In(loc)
	for day := time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, loc); day.Before(localEnd); day = day.AddDate(0, 0, 1) {
		dayEnd := day.AddDate(0, 0, 1)
		segmentStart := localStart
		if segmentStart.Before(day) {
			segmentStart = day
		}
		segmentEnd := localEnd
		if segmentEnd.After(dayEnd) {
			segmentEnd = dayEnd
		}
		if !segmentEnd.After(segmentStart) {
			continue
		}
		serviceStart := time.Date(day.Year(), day.Month(), day.Day(), serviceStartHour, 0, 0, 0, loc)
		serviceEnd := time.Date(day.Year(), day.Month(), day.Day(), serviceEndHour, 0, 0, 0, loc)
		if segmentStart.Before(serviceStart) || segmentEnd.After(serviceEnd) {
			return false
		}
	}
	return true
}

func profileFieldChanged(value *string, current string) bool {
	return value != nil && strings.TrimSpace(*value) != current
}

func isAlignedToReservationInterval(value time.Time, intervalHours int, serviceStartHour int) bool {
	if !isHourAligned(value) {
		return false
	}
	if intervalHours <= 1 {
		return true
	}
	serviceStartHour, _ = normalizeServiceHours(serviceStartHour, 24)
	localValue := value.In(reservationServiceLocation())
	base := time.Date(localValue.Year(), localValue.Month(), localValue.Day(), serviceStartHour, 0, 0, 0, localValue.Location())
	if localValue.Before(base) {
		base = base.AddDate(0, 0, -1)
	}
	if localValue.Before(base) {
		return false
	}
	diffHours := int(localValue.Sub(base).Hours())
	return diffHours >= 0 && diffHours%intervalHours == 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func roleName(role string) string {
	names := map[string]string{
		"unassigned":     "待分配",
		"student":        "学生",
		"teacher":        "教师",
		"researcher":     "研究员",
		"group_leader":   "负责人",
		"material_admin": "试剂管理员",
		"finance_admin":  "财务管理员",
		"tenant_admin":   "机构管理员",
		"lab_admin":      "实验室管理员",
		"super_admin":    "系统超级管理员",
	}
	if name, ok := names[role]; ok {
		return name
	}
	return role
}

func userStatusLabel(status string) string {
	labels := map[string]string{
		"pending_approval": "待审核",
		"active":           "已通过",
		"disabled":         "已停用",
		"deleted":          "已删除",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}

func reservationStatusLabel(status string) string {
	labels := map[string]string{
		"pending":   "待审批",
		"approved":  "已通过",
		"rejected":  "已拒绝",
		"cancelled": "已取消",
		"in_use":    "使用中",
		"completed": "已完成",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}

func materialWorkflowStatusLabel(status string) string {
	labels := map[string]string{
		"pending":    "待审批",
		"registered": "已登记",
		"approved":   "已通过",
		"rejected":   "已拒绝",
		"returned":   "退回修改",
		"outbound":   "已出库",
		"ordered":    "已下单",
		"received":   "已到货",
		"processed":  "已处理",
		"cancelled":  "已取消",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}

func notificationLevelForStatus(status string) string {
	switch status {
	case "approved", "active", "outbound", "received", "processed", "completed":
		return "success"
	case "rejected", "disabled", "cancelled":
		return "warning"
	default:
		return "info"
	}
}

func isHourAligned(value time.Time) bool {
	return value.Minute() == 0 && value.Second() == 0 && value.Nanosecond() == 0
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func passwordMatches(hash string, password string) bool {
	if strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$") || strings.HasPrefix(hash, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	return false
}

func randomToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (r *Repository) cacheDashboard(ctx context.Context, dashboard Dashboard) {
	if r.redis == nil {
		return
	}
	payload, err := json.Marshal(dashboard)
	if err != nil {
		slog.Warn("marshal dashboard cache", "error", err)
		return
	}
	if err := r.redis.Set(ctx, dashboardCacheKey(ctx), payload, 15*time.Second).Err(); err != nil {
		slog.Warn("write dashboard cache", "error", err)
	}
}

func (r *Repository) invalidateDashboard(ctx context.Context) {
	if r.redis == nil {
		return
	}
	if err := r.redis.Del(ctx, dashboardCacheKey(ctx)).Err(); err != nil {
		slog.Warn("invalidate dashboard cache", "error", err)
	}
}

func (r *Repository) enqueueEvent(ctx context.Context, eventType string, fields map[string]any) {
	if r.redis == nil {
		return
	}
	values := map[string]any{
		"type":      eventType,
		"createdAt": time.Now().UTC().Format(time.RFC3339Nano),
	}
	for key, value := range fields {
		values[key] = value
	}
	if err := r.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: "lirs:events",
		Values: values,
	}).Err(); err != nil {
		slog.Warn("enqueue redis event", "type", eventType, "error", err)
	}
}
