package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	notificationDeliveryWorkers        = 8
	notificationDeliveryQueueSize      = 256
	notificationFanoutConcurrencyLimit = 8
)

var notificationDeliveryQueue = make(chan notificationDeliveryJob, notificationDeliveryQueueSize)
var notificationDeliveryWG sync.WaitGroup
var startNotificationWorkersOnce sync.Once

type notificationDeliveryJob struct {
	repo *Repository
	item Notification
}

func startNotificationDeliveryWorkers() {
	startNotificationWorkersOnce.Do(func() {
		for i := 0; i < notificationDeliveryWorkers; i++ {
			go notificationDeliveryWorker()
		}
	})
}

func notificationDeliveryWorker() {
	for job := range notificationDeliveryQueue {
		func() {
			defer notificationDeliveryWG.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					slog.Warn("push notification panic", "notificationId", job.item.ID, "panic", recovered)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			job.repo.pushNotificationTargets(ctx, job.item)
		}()
	}
}

func DrainNotificationDelivery(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		notificationDeliveryWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func graphMailReady(settings graphMailSettingsValue) bool {
	return settings.Enabled &&
		strings.TrimSpace(settings.TenantID) != "" &&
		strings.TrimSpace(settings.ClientID) != "" &&
		strings.TrimSpace(settings.ClientSecret) != "" &&
		strings.TrimSpace(settings.SenderUserPrincipalName) != ""
}

func (r *Repository) sendGraphMail(ctx context.Context, settings graphMailSettingsValue, to string, subject string, body string) error {
	if !graphMailReady(settings) {
		return clientError("graph mail is not configured")
	}
	if _, err := mail.ParseAddress(to); err != nil {
		return clientError("graph mail recipient email is invalid")
	}
	token, err := r.graphMailAccessToken(ctx, settings)
	if err != nil {
		return err
	}
	sender := strings.TrimSpace(settings.SenderUserPrincipalName)
	payload := map[string]any{
		"message": map[string]any{
			"subject": subject,
			"body": map[string]string{
				"contentType": "Text",
				"content":     body,
			},
			"toRecipients": []map[string]any{
				{
					"emailAddress": map[string]string{
						"address": strings.TrimSpace(to),
					},
				},
			},
		},
		"saveToSentItems": settings.SaveToSentItems,
	}
	endpoint := "https://graph.microsoft.com/v1.0/users/" + url.PathEscape(sender) + "/sendMail"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(mustJSONBytes(payload)))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := r.httpClient().Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return clientErrorf("graph mail send failed: %s", graphHTTPErrorMessage(response.StatusCode, raw))
	}
	return nil
}

func (r *Repository) graphMailAccessToken(ctx context.Context, settings graphMailSettingsValue) (string, error) {
	if settings.TenantID == "" || settings.ClientID == "" || settings.ClientSecret == "" {
		return "", clientError("graph mail tenant, client, and secret are required")
	}
	form := url.Values{}
	form.Set("client_id", settings.ClientID)
	form.Set("client_secret", settings.ClientSecret)
	form.Set("grant_type", "client_credentials")
	form.Set("scope", "https://graph.microsoft.com/.default")
	endpoint := "https://login.microsoftonline.com/" + url.PathEscape(settings.TenantID) + "/oauth2/v2.0/token"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := r.httpClient().Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", clientErrorf("graph mail token failed: %s", graphHTTPErrorMessage(response.StatusCode, raw))
	}
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &tokenResponse); err != nil {
		return "", err
	}
	if tokenResponse.AccessToken == "" {
		return "", clientError("graph token response missing access_token")
	}
	return tokenResponse.AccessToken, nil
}

func graphHTTPErrorMessage(statusCode int, raw []byte) string {
	body := strings.TrimSpace(string(raw))
	var payload struct {
		Error any `json:"error"`
	}
	if body != "" && json.Unmarshal(raw, &payload) == nil {
		switch value := payload.Error.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				return fmt.Sprintf("status=%d message=%s", statusCode, strings.TrimSpace(value))
			}
		case map[string]any:
			code, _ := value["code"].(string)
			message, _ := value["message"].(string)
			if strings.TrimSpace(code) != "" || strings.TrimSpace(message) != "" {
				return fmt.Sprintf("status=%d code=%s message=%s", statusCode, strings.TrimSpace(code), strings.TrimSpace(message))
			}
		}
	}
	if body == "" {
		return fmt.Sprintf("status=%d", statusCode)
	}
	if len(body) > 500 {
		body = body[:500] + "..."
	}
	return fmt.Sprintf("status=%d body=%s", statusCode, body)
}

func (r *Repository) pushDingTalkNotification(ctx context.Context, tenantID string, userID string, title string, body string) {
	r.pushDingTalkNotificationTargets(ctx, Notification{
		TenantID:    tenantID,
		UserID:      userID,
		TargetScope: "personal",
		Title:       title,
		Body:        body,
	})
}

func (r *Repository) pushNotificationTargets(ctx context.Context, item Notification) {
	r.pushDingTalkNotificationTargets(ctx, item)
	r.pushGraphMailNotificationTargets(ctx, item)
}

func (r *Repository) pushDingTalkNotificationTargets(ctx context.Context, item Notification) {
	tenantID := strings.TrimSpace(item.TenantID)
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	item.TenantID = tenantID
	ctx = WithTenantContext(ctx, TenantContext{TenantID: tenantID})
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	settings, err := r.dingTalkSettingsValue(ctx)
	if err != nil || !settings.Enabled || settings.ClientID == "" || settings.ClientSecret == "" || settings.RobotCode == "" {
		return
	}
	userIDs, err := r.notificationTargetUserIDs(ctx, item, true, false)
	if err != nil {
		slog.Warn("load dingtalk notification targets", "notificationId", item.ID, "error", err)
		return
	}
	runNotificationFanout(ctx, userIDs, func(userID string) {
		r.pushDingTalkNotificationToUser(ctx, settings, tenantID, userID, item.Title, item.Body)
	})
}

func (r *Repository) pushGraphMailNotificationTargets(ctx context.Context, item Notification) {
	tenantID := strings.TrimSpace(item.TenantID)
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	item.TenantID = tenantID
	ctx = WithTenantContext(ctx, TenantContext{TenantID: tenantID})
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	settings, err := r.graphMailSettingsValue(ctx)
	if err != nil || !graphMailReady(settings) {
		return
	}
	userIDs, err := r.notificationTargetUserIDs(ctx, item, false, true)
	if err != nil {
		slog.Warn("load graph mail notification targets", "notificationId", item.ID, "error", err)
		return
	}
	runNotificationFanout(ctx, userIDs, func(userID string) {
		r.pushGraphMailNotificationToUser(ctx, settings, tenantID, userID, item.Title, item.Body)
	})
}

func runNotificationFanout(ctx context.Context, userIDs []string, push func(string)) {
	if len(userIDs) == 0 {
		return
	}
	limit := notificationFanoutConcurrencyLimit
	if limit <= 0 {
		limit = 1
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for _, userID := range userIDs {
		userID := userID
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				<-sem
				if recovered := recover(); recovered != nil {
					slog.Warn("push notification target panic", "userId", userID, "panic", recovered)
				}
			}()
			push(userID)
		}()
	}
	wg.Wait()
}

func (r *Repository) notificationTargetUserIDs(ctx context.Context, item Notification, requireDingTalk bool, requireEmail bool) ([]string, error) {
	scope := strings.TrimSpace(item.TargetScope)
	if scope == "" {
		scope = "global"
	}
	switch scope {
	case "personal":
		userID := strings.TrimSpace(item.UserID)
		if userID == "" {
			return nil, nil
		}
		return []string{userID}, nil
	case "group":
		return r.notificationUsersByFilter(ctx, item.TenantID, "group", item.GroupName, requireDingTalk, requireEmail)
	case "department":
		return r.notificationUsersByFilter(ctx, item.TenantID, "department", item.Department, requireDingTalk, requireEmail)
	case "global":
		return r.notificationUsersByFilter(ctx, item.TenantID, "global", "", requireDingTalk, requireEmail)
	default:
		return nil, nil
	}
}

func (r *Repository) notificationUsersByFilter(ctx context.Context, tenantID string, kind string, value string, requireDingTalk bool, requireEmail bool) ([]string, error) {
	value = strings.TrimSpace(value)
	rows, err := r.db.Query(ctx, `
SELECT id::text
FROM users
WHERE tenant_id = $1::uuid
  AND status = 'active'
  AND ($4::boolean = false OR dingtalk_user_id <> '')
  AND ($5::boolean = false OR email <> '')
  AND (
      $2 = 'global'
      OR ($2 = 'group' AND group_name = $3)
      OR ($2 = 'department' AND department = $3)
  )
ORDER BY created_at DESC
`, tenantID, kind, value, requireDingTalk, requireEmail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := make(map[string]struct{})
	userIDs := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}
	return userIDs, rows.Err()
}

func (r *Repository) pushDingTalkNotificationToUser(ctx context.Context, settings dingTalkSettingsValue, tenantID string, userID string, title string, body string) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	var dingTalkUserID string
	err := r.db.QueryRow(ctx, `
SELECT dingtalk_user_id
FROM users
WHERE id = $1
  AND tenant_id = $2::uuid
  AND dingtalk_user_id <> ''
`, userID, tenantID).Scan(&dingTalkUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		slog.Warn("load dingtalk binding", "userId", userID, "error", err)
		return
	}
	if err := r.sendDingTalkWorkNotification(ctx, settings, dingTalkUserID, title, body); err != nil {
		slog.Warn("send dingtalk notification", "userId", userID, "dingtalkUserId", dingTalkUserID, "error", err)
	}
}

func (r *Repository) pushGraphMailNotificationToUser(ctx context.Context, settings graphMailSettingsValue, tenantID string, userID string, title string, body string) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	var email string
	err := r.db.QueryRow(ctx, `
SELECT email
FROM users
WHERE id = $1
  AND tenant_id = $2::uuid
  AND status = 'active'
  AND email <> ''
`, userID, tenantID).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		slog.Warn("load graph mail notification email", "userId", userID, "error", err)
		return
	}
	subject := strings.TrimSpace(title)
	if subject == "" {
		subject = "实验室运营系统通知"
	}
	if err := r.sendGraphMail(ctx, settings, email, subject, body); err != nil {
		slog.Warn("send graph mail notification", "userId", userID, "email", email, "error", err)
	}
}

func (r *Repository) sendDingTalkWorkNotification(ctx context.Context, settings dingTalkSettingsValue, userID string, title string, body string) error {
	token, err := r.dingTalkAppAccessToken(ctx, settings)
	if err != nil {
		return err
	}
	message := strings.TrimSpace(body)
	if message == "" {
		message = title
	}
	payload := map[string]any{
		"robotCode": settings.RobotCode,
		"userIds":   []string{userID},
		"msgKey":    "sampleMarkdown",
		"msgParam": mustJSON(map[string]string{
			"title": title,
			"text":  "### " + title + "\n\n" + message,
		}),
	}
	var response struct {
		ProcessQueryKey string `json:"processQueryKey"`
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend", bytes.NewReader(mustJSONBytes(payload)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-acs-dingtalk-access-token", token)
	if err := r.dingTalkDo(request, &response); err != nil {
		return err
	}
	return nil
}

func (r *Repository) dingTalkPost(ctx context.Context, endpoint string, payload any, target any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	return r.dingTalkDo(request, target)
}

func (r *Repository) dingTalkDo(request *http.Request, target any) error {
	response, err := r.httpClient().Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return clientErrorf("dingtalk http %d: %s", response.StatusCode, strings.TrimSpace(string(raw)))
	}
	if target == nil {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}
