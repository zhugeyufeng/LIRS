package httpapi

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"lirs/apps/server/internal/store"
)

type repository interface {
	Health(ctx context.Context) error
	Dashboard(ctx context.Context) (store.Dashboard, error)
	Tenants(ctx context.Context) ([]store.Tenant, error)
	SaveTenant(ctx context.Context, id string, input store.TenantInput) (store.Tenant, error)
	FooterSettings(ctx context.Context) (store.FooterSettings, error)
	SaveFooterSettings(ctx context.Context, input store.FooterSettingsInput) (store.FooterSettings, error)
	CopySettings(ctx context.Context) (store.CopySettings, error)
	SaveCopySettings(ctx context.Context, input store.CopySettingsInput) (store.CopySettings, error)
	NotificationChannelSettings(ctx context.Context) (store.NotificationChannelSettings, error)
	SaveSMTPSettings(ctx context.Context, input store.SMTPSettingsInput) (store.SMTPSettings, error)
	SaveWeChatSettings(ctx context.Context, input store.WeChatSettingsInput) (store.WeChatSettings, error)
	DingTalkSettings(ctx context.Context) (store.DingTalkSettings, error)
	SaveDingTalkSettings(ctx context.Context, input store.DingTalkSettingsInput) (store.DingTalkSettings, error)
	HandleDingTalkEventCallback(ctx context.Context, input store.DingTalkEventCallbackInput) (store.DingTalkEventCallbackResponse, error)
	AccessControlSettings(ctx context.Context) (store.AccessControlSettings, error)
	SaveAccessControlSettings(ctx context.Context, input store.AccessControlSettingsInput) (store.AccessControlSettings, error)
	Instruments(ctx context.Context, filter store.InstrumentFilter) ([]store.Instrument, error)
	Instrument(ctx context.Context, id string) (store.Instrument, error)
	DeleteInstrument(ctx context.Context, id string, actor string) (store.Instrument, error)
	TrainingCourses(ctx context.Context) ([]store.TrainingCourse, error)
	SaveTrainingCourse(ctx context.Context, id string, input store.TrainingCourseInput) (store.TrainingCourse, error)
	TrainingAuthorizations(ctx context.Context) ([]store.TrainingAuthorization, error)
	SaveTrainingAuthorization(ctx context.Context, id string, input store.TrainingAuthorizationInput) (store.TrainingAuthorization, error)
	TrainingQuestions(ctx context.Context) ([]store.TrainingQuestion, error)
	SaveTrainingQuestion(ctx context.Context, id string, input store.TrainingQuestionInput) (store.TrainingQuestion, error)
	TrainingExams(ctx context.Context) ([]store.TrainingExam, error)
	SaveTrainingExam(ctx context.Context, id string, input store.TrainingExamInput) (store.TrainingExam, error)
	TrainingPracticals(ctx context.Context) ([]store.TrainingPractical, error)
	SaveTrainingPractical(ctx context.Context, id string, input store.TrainingPracticalInput) (store.TrainingPractical, error)
	TrainingRules(ctx context.Context) ([]store.TrainingRule, error)
	SaveTrainingRule(ctx context.Context, id string, input store.TrainingRuleInput) (store.TrainingRule, error)
	BusinessConfigs(ctx context.Context, domain string, kind string) ([]store.BusinessConfig, error)
	SaveBusinessConfig(ctx context.Context, domain string, kind string, id string, input store.BusinessConfigInput) (store.BusinessConfig, error)
	Spaces(ctx context.Context) ([]store.Space, error)
	SaveSpace(ctx context.Context, id string, input store.SpaceInput) (store.Space, error)
	SpaceReservations(ctx context.Context) ([]store.SpaceReservation, error)
	CreateSpaceReservation(ctx context.Context, input store.SpaceReservationInput) (store.SpaceReservation, error)
	Samples(ctx context.Context) ([]store.Sample, error)
	SaveSample(ctx context.Context, id string, input store.SampleInput) (store.Sample, error)
	SampleMovements(ctx context.Context) ([]store.SampleMovement, error)
	CreateSampleMovement(ctx context.Context, input store.SampleMovementInput) (store.SampleMovement, error)
	LimsTasks(ctx context.Context) ([]store.LimsTask, error)
	SaveLimsTask(ctx context.Context, id string, input store.LimsTaskInput) (store.LimsTask, error)
	ElnRecords(ctx context.Context) ([]store.ElnRecord, error)
	SaveElnRecord(ctx context.Context, id string, input store.ElnRecordInput) (store.ElnRecord, error)
	IotDevices(ctx context.Context) ([]store.IotDevice, error)
	SaveIotDevice(ctx context.Context, id string, input store.IotDeviceInput) (store.IotDevice, error)
	AssistantQueries(ctx context.Context) ([]store.AssistantQuery, error)
	AskAssistant(ctx context.Context, input store.AssistantQueryInput) (store.AssistantQuery, error)
	Reservation(ctx context.Context, id string) (store.Reservation, error)
	InstrumentSlots(ctx context.Context, id string, start time.Time, days int) ([]store.Slot, error)
	SaveInstrument(ctx context.Context, id string, input store.InstrumentInput) (store.Instrument, error)
	Reservations(ctx context.Context) ([]store.Reservation, error)
	Users(ctx context.Context) ([]store.User, error)
	ReviewUser(ctx context.Context, id string, input store.UserReviewInput) (store.User, error)
	SaveUserMembership(ctx context.Context, id string, input store.UserMembershipInput) (store.User, error)
	DeleteUser(ctx context.Context, id string, actor string) (store.User, error)
	OrganizationUnits(ctx context.Context, kind string) ([]store.OrganizationUnit, error)
	SaveOrganizationUnit(ctx context.Context, id string, input store.OrganizationUnitInput) (store.OrganizationUnit, error)
	DeleteOrganizationUnit(ctx context.Context, id string, actor string) (store.OrganizationUnit, error)
	Notifications(ctx context.Context, actor store.Actor) ([]store.Notification, error)
	MarkNotificationRead(ctx context.Context, id string, actor store.Actor) (store.Notification, error)
	MarkAllNotificationsRead(ctx context.Context, actor store.Actor) (int, error)
	DeleteNotification(ctx context.Context, id string, actor string) (store.Notification, error)
	Announce(ctx context.Context, input store.AnnouncementInput) (store.Notification, error)
	UpdateNotification(ctx context.Context, id string, input store.AnnouncementInput) (store.Notification, error)
	Ledger(ctx context.Context, actor store.Actor) ([]store.LedgerEntry, error)
	AdjustLedger(ctx context.Context, input store.LedgerAdjustmentInput) (store.LedgerEntry, error)
	FinancialAccounts(ctx context.Context, actor store.Actor) ([]store.FinancialAccount, error)
	SaveFinancialAccount(ctx context.Context, id string, input store.FinancialAccountInput) (store.FinancialAccount, error)
	Register(ctx context.Context, input store.RegisterInput) (store.User, error)
	RequestEmailVerificationCode(ctx context.Context, input store.EmailVerificationCodeInput) (store.EmailVerificationCodeResponse, error)
	CreateReservation(ctx context.Context, input store.ReservationInput) (store.Reservation, error)
	CreateReservationBatch(ctx context.Context, input store.ReservationBatchInput) ([]store.Reservation, error)
	ApproveReservation(ctx context.Context, id string, approved bool, actor string, comment string) (store.Reservation, error)
	CheckInReservation(ctx context.Context, id string) (store.Reservation, error)
	CompleteReservation(ctx context.Context, id string) (store.Reservation, error)
	CancelReservation(ctx context.Context, id string, reason string, bypassCutoff bool) (store.Reservation, error)
	Materials(ctx context.Context) ([]store.Material, error)
	Material(ctx context.Context, id string) (store.Material, error)
	MaterialByQRCode(ctx context.Context, code string) (store.Material, error)
	DeleteMaterial(ctx context.Context, id string, actor string) (store.Material, error)
	MaterialCategories(ctx context.Context) ([]store.MaterialCategory, error)
	SaveMaterialCategory(ctx context.Context, id string, input store.MaterialCategoryInput) (store.MaterialCategory, error)
	DeleteMaterialCategory(ctx context.Context, id string, actor string) (store.MaterialCategory, error)
	MaterialRequest(ctx context.Context, id string) (store.MaterialRequest, error)
	InventoryLedger(ctx context.Context) ([]store.InventoryLedgerEntry, error)
	SaveMaterial(ctx context.Context, id string, input store.MaterialInput) (store.Material, error)
	ImportMaterialsCSV(ctx context.Context, content string, actor string) (store.MaterialImportResult, error)
	MaterialAnalytics(ctx context.Context) (store.MaterialAnalytics, error)
	MaterialAlertActions(ctx context.Context) ([]store.MaterialAlertAction, error)
	CreateMaterialAlertAction(ctx context.Context, materialID string, input store.MaterialAlertActionInput) (store.MaterialAlertAction, error)
	AdjustMaterialStock(ctx context.Context, id string, input store.StockAdjustmentInput) (store.Material, error)
	MaterialRequests(ctx context.Context) ([]store.MaterialRequest, error)
	CreateMaterialRequest(ctx context.Context, input store.MaterialRequestInput) (store.MaterialRequest, error)
	ApproveMaterialRequest(ctx context.Context, id string, approved bool, actor string, comment string) (store.MaterialRequest, error)
	OutboundMaterialRequest(ctx context.Context, id string, actor string) (store.MaterialRequest, error)
	CancelMaterialRequest(ctx context.Context, id string, actor string) (store.MaterialRequest, error)
	MaterialPurchase(ctx context.Context, id string) (store.MaterialPurchase, error)
	MaterialPurchases(ctx context.Context) ([]store.MaterialPurchase, error)
	CreateMaterialPurchase(ctx context.Context, input store.MaterialPurchaseInput) (store.MaterialPurchase, error)
	ApproveMaterialPurchase(ctx context.Context, id string, approved bool, actor string, comment string) (store.MaterialPurchase, error)
	MarkMaterialPurchaseOrdered(ctx context.Context, id string, actor string) (store.MaterialPurchase, error)
	ReceiveMaterialPurchase(ctx context.Context, id string, actor string) (store.MaterialPurchase, error)
	CancelMaterialPurchase(ctx context.Context, id string, actor string) (store.MaterialPurchase, error)
	MaterialDamage(ctx context.Context, id string) (store.MaterialDamage, error)
	MaterialDamages(ctx context.Context) ([]store.MaterialDamage, error)
	CreateMaterialDamage(ctx context.Context, input store.MaterialDamageInput) (store.MaterialDamage, error)
	ApproveMaterialDamage(ctx context.Context, id string, approved bool, actor string, comment string) (store.MaterialDamage, error)
	ProcessMaterialDamage(ctx context.Context, id string, actor string) (store.MaterialDamage, error)
	CancelMaterialDamage(ctx context.Context, id string, actor string) (store.MaterialDamage, error)
	MaintenanceOrders(ctx context.Context) ([]store.MaintenanceOrder, error)
	CreateMaintenanceOrder(ctx context.Context, input store.MaintenanceInput) (store.MaintenanceOrder, error)
	StartMaintenanceOrder(ctx context.Context, id string, actor string) (store.MaintenanceOrder, error)
	CancelMaintenanceOrder(ctx context.Context, id string, reason string, actor string) (store.MaintenanceOrder, error)
	CompleteMaintenanceOrder(ctx context.Context, id string, result string, actor string) (store.MaintenanceOrder, error)
	AuditEvents(ctx context.Context) ([]store.AuditEvent, error)
	Operations(ctx context.Context) (store.Operations, error)
	Login(ctx context.Context, input store.LoginInput) (store.AuthResponse, error)
	DingTalkQuickLogin(ctx context.Context, input store.DingTalkQuickLoginInput) (store.AuthResponse, error)
	CurrentUser(ctx context.Context, token string) (store.User, error)
	Logout(ctx context.Context, token string) error
	LogoutAll(ctx context.Context, userID string) error
	VerifyEmail(ctx context.Context, token string) (store.User, error)
	UpdateCurrentUserProfile(ctx context.Context, id string, input store.UserProfileInput) (store.User, error)
	CurrentUserDingTalkBinding(ctx context.Context, id string) (store.DingTalkBinding, error)
	BindCurrentUserDingTalk(ctx context.Context, id string, input store.DingTalkBindingInput) (store.DingTalkBinding, error)
	UnbindCurrentUserDingTalk(ctx context.Context, id string, actor string) (store.DingTalkBinding, error)
	ChangePassword(ctx context.Context, id string, input store.PasswordChangeInput) error
}

type authRepository interface {
	CurrentUser(ctx context.Context, token string) (store.User, error)
}

var anyAdminRoles = []string{"material_admin", "finance_admin", "tenant_admin", "lab_admin", "super_admin"}
var tenantAdminRoles = []string{"tenant_admin", "lab_admin", "super_admin"}
var userReaderRoles = []string{"finance_admin", "tenant_admin", "lab_admin", "super_admin"}
var materialAdminRoles = []string{"material_admin", "tenant_admin", "lab_admin", "super_admin"}
var financeAdminRoles = []string{"finance_admin", "tenant_admin", "lab_admin", "super_admin"}

func RegisterRoutes(router *gin.Engine, repo repository) {
	router.GET("/healthz", func(c *gin.Context) {
		if err := repo.Health(c.Request.Context()); err != nil {
			slog.Warn("health check failed", "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "error": "service unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	api.Use(tenantContextMiddleware(repo))
	api.GET("/dashboard", get(caller(repo.Dashboard)))
	api.GET("/tenants", func(c *gin.Context) {
		items, err := repo.Tenants(c.Request.Context())
		if err != nil {
			respond(c, items, err)
			return
		}
		if user, ok := optionalCurrentUser(c, repo); ok && user.Role == "super_admin" {
			respond(c, items, nil)
			return
		}
		active := make([]store.Tenant, 0, len(items))
		for _, item := range items {
			if item.Status == "active" {
				active = append(active, item)
			}
		}
		respond(c, active, nil)
	})
	api.POST("/tenants", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "super_admin")
		if !ok {
			return
		}
		var input store.TenantInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTenant(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/tenants/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		if actor.Role != "super_admin" && c.Param("id") != actor.TenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		var input store.TenantInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTenant(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/footer-settings", get(caller(repo.FooterSettings)))
	api.GET("/copy-settings", get(caller(repo.CopySettings)))
	api.PATCH("/copy-settings", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.CopySettingsInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveCopySettings(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/notification-channel-settings", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, "super_admin"); !ok {
			return
		}
		item, err := repo.NotificationChannelSettings(c.Request.Context())
		respond(c, item, err)
	})
	api.PATCH("/notification-channel-settings/smtp", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "super_admin")
		if !ok {
			return
		}
		var input store.SMTPSettingsInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveSMTPSettings(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.PATCH("/notification-channel-settings/wechat", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "super_admin")
		if !ok {
			return
		}
		var input store.WeChatSettingsInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveWeChatSettings(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/notification-channel-settings/dingtalk", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		requestCtx, ok := tenantAdminRequestContext(c, repo, actor)
		if !ok {
			return
		}
		item, err := repo.DingTalkSettings(requestCtx)
		respond(c, item, err)
	})
	api.PATCH("/notification-channel-settings/dingtalk", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		requestCtx, ok := tenantAdminRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.DingTalkSettingsInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveDingTalkSettings(requestCtx, input)
			respond(c, item, err)
		}
	})
	api.POST("/dingtalk/events", func(c *gin.Context) {
		var body struct {
			Encrypt string `json:"encrypt"`
		}
		if !bindJSON(c, &body) {
			return
		}
		item, err := repo.HandleDingTalkEventCallback(c.Request.Context(), store.DingTalkEventCallbackInput{
			TenantID:   c.Param("tenant"),
			TenantCode: c.Param("tenant"),
			Signature:  c.Query("signature"),
			Timestamp:  c.Query("timestamp"),
			Nonce:      c.Query("nonce"),
			Encrypt:    body.Encrypt,
		})
		respond(c, item, err)
	})
	api.POST("/dingtalk/events/:tenant", func(c *gin.Context) {
		var body struct {
			Encrypt string `json:"encrypt"`
		}
		if !bindJSON(c, &body) {
			return
		}
		item, err := repo.HandleDingTalkEventCallback(c.Request.Context(), store.DingTalkEventCallbackInput{
			TenantID:   c.Param("tenant"),
			TenantCode: c.Param("tenant"),
			Signature:  c.Query("signature"),
			Timestamp:  c.Query("timestamp"),
			Nonce:      c.Query("nonce"),
			Encrypt:    body.Encrypt,
		})
		respond(c, item, err)
	})
	api.GET("/access-control-settings", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, "super_admin"); !ok {
			return
		}
		item, err := repo.AccessControlSettings(c.Request.Context())
		respond(c, item, err)
	})
	api.PATCH("/access-control-settings", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "super_admin")
		if !ok {
			return
		}
		var input store.AccessControlSettingsInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveAccessControlSettings(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/instruments", func(c *gin.Context) {
		filter := store.InstrumentFilter{
			Search:     c.Query("search"),
			Category:   c.Query("category"),
			Department: c.Query("department"),
			GroupName:  c.Query("group"),
			Status:     c.Query("status"),
			Limit:      intQuery(c, "limit", 1000),
			Offset:     intQuery(c, "offset", 0),
		}
		item, err := repo.Instruments(c.Request.Context(), filter)
		respond(c, item, err)
	})
	api.GET("/instruments/:id", func(c *gin.Context) {
		item, err := repo.Instrument(c.Request.Context(), c.Param("id"))
		respond(c, item, err)
	})
	api.GET("/instruments/:id/slots", func(c *gin.Context) {
		start := time.Now()
		if raw := strings.TrimSpace(c.Query("start")); raw != "" {
			parsed, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time"})
				return
			}
			start = parsed
		}
		item, err := repo.InstrumentSlots(c.Request.Context(), c.Param("id"), start, intQuery(c, "days", 30))
		respond(c, item, err)
	})
	api.POST("/instruments", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.InstrumentInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveInstrument(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/instruments/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.InstrumentInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveInstrument(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.DELETE("/instruments/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		item, err := repo.DeleteInstrument(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/footer-settings", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.FooterSettingsInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveFooterSettings(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/reservations", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.Reservations(c.Request.Context())
		item = filterReservationsForActor(actor, item)
		respond(c, item, err)
	})
	api.GET("/users", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, userReaderRoles...); !ok {
			return
		}
		item, err := repo.Users(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/organization-units", func(c *gin.Context) {
		ctx := c.Request.Context()
		user, hasUser := optionalCurrentUser(c, repo)
		if tenantID := strings.TrimSpace(c.Query("tenantId")); tenantID != "" {
			if !hasUser || user.Role == "super_admin" {
				ctx = store.WithTenantContext(ctx, store.TenantContext{TenantID: tenantID})
			}
		} else if hasUser && user.Role == "super_admin" {
			ctx = store.WithTenantContext(ctx, store.TenantContext{
				TenantID:       user.TenantID,
				TenantName:     user.TenantName,
				FinanceEnabled: user.FinanceEnabled,
			})
		}
		item, err := repo.OrganizationUnits(ctx, c.Query("kind"))
		respond(c, item, err)
	})
	api.POST("/organization-units", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.OrganizationUnitInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			ctx, ok := organizationUnitRequestContext(c, repo, actor)
			if !ok {
				return
			}
			item, err := repo.SaveOrganizationUnit(ctx, "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/organization-units/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.OrganizationUnitInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			ctx, ok := organizationUnitRequestContext(c, repo, actor)
			if !ok {
				return
			}
			item, err := repo.SaveOrganizationUnit(ctx, c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.DELETE("/organization-units/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		ctx, ok := organizationUnitRequestContext(c, repo, actor)
		if !ok {
			return
		}
		item, err := repo.DeleteOrganizationUnit(ctx, c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.GET("/notifications", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		ctx := c.Request.Context()
		if isAdmin(actor) {
			var contextOK bool
			ctx, contextOK = tenantAdminRequestContext(c, repo, actor)
			if !contextOK {
				return
			}
		}
		item, err := repo.Notifications(ctx, actor)
		respond(c, item, err)
	})
	api.PATCH("/notifications/read-all", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		count, err := repo.MarkAllNotificationsRead(c.Request.Context(), actor)
		respond(c, gin.H{"count": count}, err)
	})
	api.GET("/ledger", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		item, err := repo.Ledger(ctx, actor)
		if err == nil {
			item = filterLedgerForActor(actor, item)
		}
		respond(c, item, err)
	})
	api.GET("/ledger/export.csv", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		item, err := repo.Ledger(ctx, actor)
		if err != nil {
			respond(c, nil, err)
			return
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-ledger.csv")
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write([]string{"id", "user_id", "user_name", "reservation_id", "owner", "description", "entry_type", "amount", "reference_id", "created_at"})
		for _, entry := range item {
			_ = writer.Write([]string{
				entry.ID,
				entry.UserID,
				entry.UserName,
				entry.ReservationID,
				entry.GroupName,
				entry.Description,
				entry.EntryType,
				fmt.Sprintf("%.2f", entry.Amount),
				entry.ReferenceID,
				entry.CreatedAt.Format(time.RFC3339),
			})
		}
		writer.Flush()
	})
	api.GET("/financial-accounts", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		item, err := repo.FinancialAccounts(ctx, actor)
		if err == nil {
			item = filterFinancialAccountsForActor(actor, item)
		}
		respond(c, item, err)
	})
	api.POST("/financial-accounts", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, financeAdminRoles...)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.FinancialAccountInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveFinancialAccount(ctx, "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/financial-accounts/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, financeAdminRoles...)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.FinancialAccountInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveFinancialAccount(ctx, c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/materials", get(caller(repo.Materials)))
	api.GET("/materials/analytics", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles...); !ok {
			return
		}
		item, err := repo.MaterialAnalytics(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/materials/categories", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles...); !ok {
			return
		}
		item, err := repo.MaterialCategories(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/materials/alert-actions", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles...); !ok {
			return
		}
		item, err := repo.MaterialAlertActions(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/materials/import-template.csv", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles...); !ok {
			return
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-materials-import-template.csv")
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write(materialImportTemplateHeader())
		_ = writer.Write([]string{"示例铅工作液", "标准品", "工作液", "金属元素", "100ug/L 50mL", "瓶", "80", "1", "1", "国家标准物质中心", "NIM", "WORK-PB-001", "GBW(E)080129-D", "7439-92-1", "CRM", "100ug/L", "", "1:10", "铅标准溶液稀释配制", "2-8°C 避光", "标准品库", "防爆冰箱", "二层", "A09", "2026-标准品采购合同", "TC-2026-066", "", "/files/certs/pb-standard.pdf", "", "WORK-PB-001", "2026-08-31", "2026-05-15", "30", "0", "5", "是", "30", "正常"})
		writer.Flush()
	})
	api.GET("/materials/export.csv", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		items, err := repo.Materials(c.Request.Context())
		if err != nil {
			respond(c, nil, err)
			return
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-materials.csv")
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write([]string{"资源名称", "资源类型", "一级目录", "二级目录", "CAS号", "规格", "单位", "库存", "低库存线", "损毁数", "供应商", "生产商", "批号", "货号", "母液/来源", "稀释倍数", "配制方法", "库位", "有效期", "开封日期", "开封到期", "冻融次数", "审批", "二维码", "状态"})
		for _, item := range items {
			if !canManageMaterials(actor) {
				continue
			}
			_ = writer.Write([]string{
				item.Name,
				materialProductTypeLabel(item.ProductType),
				item.Category,
				item.Subcategory,
				item.CASNo,
				item.Spec,
				item.Unit,
				strconv.Itoa(item.Stock),
				strconv.Itoa(item.WarningLine),
				strconv.Itoa(item.DamagedQuantity),
				item.Supplier,
				item.Manufacturer,
				item.BatchNo,
				item.CatalogNo,
				item.ParentMaterialName,
				item.DilutionFactor,
				item.PreparationMethod,
				materialLocation(item),
				item.ExpiresAt,
				item.OpenedAt,
				item.OpenExpiresAt,
				fmt.Sprintf("%d/%d", item.FreezeThawCount, item.FreezeThawLimit),
				boolLabel(item.ApprovalRequired),
				item.QRCode,
				materialStatusLabel(item.Status),
			})
		}
		writer.Flush()
	})
	api.POST("/materials/import.csv", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20))
		if err != nil {
			respond(c, nil, err)
			return
		}
		item, err := repo.ImportMaterialsCSV(c.Request.Context(), string(body), actor.Name)
		respond(c, item, err)
	})
	api.GET("/material-requests/export.csv", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		items, err := repo.MaterialRequests(c.Request.Context())
		if err != nil {
			respond(c, nil, err)
			return
		}
		items = filterMaterialRequestsForActor(actor, items)
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-material-requests.csv")
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write([]string{"产品", "申请人", "课题组", "数量", "用途", "状态", "创建时间", "模板类型"})
		for _, item := range items {
			templateType := "试剂领用记录表"
			if strings.Contains(item.MaterialName, "标准") {
				templateType = "标准品领用记录表"
			}
			_ = writer.Write([]string{item.MaterialName, item.Requester, item.GroupName, strconv.Itoa(item.Quantity), item.Purpose, materialRequestStatusLabel(item.Status), item.CreatedAt.Format(time.RFC3339), templateType})
		}
		writer.Flush()
	})
	api.GET("/material-damages/export.csv", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		items, err := repo.MaterialDamages(c.Request.Context())
		if err != nil {
			respond(c, nil, err)
			return
		}
		items = filterMaterialDamagesForActor(actor, items)
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-material-damages.csv")
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write([]string{"资源", "登记人", "课题组", "唯一编号", "批次", "数量", "原因", "照片", "附件", "状态", "审核人", "审核备注", "创建时间"})
		for _, item := range items {
			_ = writer.Write([]string{item.MaterialName, item.Reporter, item.GroupName, item.UnitCode, item.BatchNo, strconv.Itoa(item.Quantity), item.Reason, item.PhotoURL, item.AttachmentURL, materialDamageStatusLabel(item.Status), item.Reviewer, item.ReviewComment, item.CreatedAt.Format(time.RFC3339)})
		}
		writer.Flush()
	})
	api.GET("/training/courses", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.TrainingCourses(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/training/courses", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.TrainingCourseInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingCourse(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/courses/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.TrainingCourseInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingCourse(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/training/authorizations", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.TrainingAuthorizations(c.Request.Context())
		if err == nil && !canManageTraining(actor) {
			filtered := make([]store.TrainingAuthorization, 0, len(item))
			for _, record := range item {
				if record.UserID == actor.UserID || record.UserName == actor.Name {
					filtered = append(filtered, record)
				}
			}
			item = filtered
		}
		respond(c, item, err)
	})
	api.POST("/training/authorizations", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.TrainingAuthorizationInput
		if bindJSON(c, &input) {
			if !isAdmin(actor) {
				input.UserID = actor.UserID
				input.UserName = actor.Name
				input.Status = "pending"
			}
			if input.UserID == "" {
				input.UserID = actor.UserID
			}
			if input.UserName == "" {
				input.UserName = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.SaveTrainingAuthorization(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/authorizations/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.TrainingAuthorizationInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingAuthorization(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/training/questions", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.TrainingQuestions(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/training/questions", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.TrainingQuestionInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingQuestion(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/questions/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.TrainingQuestionInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingQuestion(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/training/exams", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.TrainingExams(c.Request.Context())
		if err == nil && !isAdmin(actor) {
			filtered := make([]store.TrainingExam, 0, len(item))
			for _, record := range item {
				if record.UserID == actor.UserID || record.UserName == actor.Name {
					filtered = append(filtered, record)
				}
			}
			item = filtered
		}
		respond(c, item, err)
	})
	api.POST("/training/exams", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.TrainingExamInput
		if bindJSON(c, &input) {
			if !isAdmin(actor) {
				input.UserID = actor.UserID
				input.UserName = actor.Name
			}
			if input.UserName == "" {
				input.UserName = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.SaveTrainingExam(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/exams/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.TrainingExamInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingExam(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/training/practicals", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles...); !ok {
			return
		}
		item, err := repo.TrainingPracticals(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/training/practicals", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.TrainingPracticalInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingPractical(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/practicals/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.TrainingPracticalInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingPractical(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/training/rules", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles...); !ok {
			return
		}
		item, err := repo.TrainingRules(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/training/rules", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.TrainingRuleInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingRule(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/rules/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.TrainingRuleInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingRule(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/workflows/:kind", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles...); !ok {
			return
		}
		item, err := repo.BusinessConfigs(c.Request.Context(), "workflow", c.Param("kind"))
		respond(c, item, err)
	})
	api.POST("/workflows/:kind", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.BusinessConfigInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveBusinessConfig(c.Request.Context(), "workflow", c.Param("kind"), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/workflows/:kind/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.BusinessConfigInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveBusinessConfig(c.Request.Context(), "workflow", c.Param("kind"), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/billing/:kind", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, financeAdminRoles...)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		item, err := repo.BusinessConfigs(ctx, "billing", c.Param("kind"))
		respond(c, item, err)
	})
	api.POST("/billing/:kind", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, financeAdminRoles...)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.BusinessConfigInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveBusinessConfig(ctx, "billing", c.Param("kind"), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/billing/:kind/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, financeAdminRoles...)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.BusinessConfigInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveBusinessConfig(ctx, "billing", c.Param("kind"), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/spaces", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.Spaces(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/spaces", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.SpaceInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveSpace(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/spaces/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.SpaceInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveSpace(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/space-reservations", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.SpaceReservations(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/space-reservations", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.SpaceReservationInput
		if bindJSON(c, &input) {
			input.RequesterID = actor.UserID
			input.Requester = actor.Name
			input.Actor = actor.Name
			item, err := repo.CreateSpaceReservation(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/samples", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.Samples(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/samples", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.SampleInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveSample(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/samples/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.SampleInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveSample(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/sample-movements", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.SampleMovements(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/sample-movements", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.SampleMovementInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.CreateSampleMovement(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/lims/tasks", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.LimsTasks(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/lims/tasks", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.LimsTaskInput
		if bindJSON(c, &input) {
			input.RequesterID = actor.UserID
			if input.RequesterName == "" {
				input.RequesterName = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.SaveLimsTask(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/lims/tasks/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.LimsTaskInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveLimsTask(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/eln/records", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.ElnRecords(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/eln/records", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.ElnRecordInput
		if bindJSON(c, &input) {
			input.AuthorID = actor.UserID
			if input.AuthorName == "" {
				input.AuthorName = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.SaveElnRecord(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/eln/records/:id", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.ElnRecordInput
		if bindJSON(c, &input) {
			input.AuthorID = actor.UserID
			if input.AuthorName == "" {
				input.AuthorName = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.SaveElnRecord(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/iot/devices", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.IotDevices(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/iot/devices", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.IotDeviceInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveIotDevice(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/iot/devices/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.IotDeviceInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveIotDevice(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/ai-assistant", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.AssistantQueries(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/ai-assistant", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.AssistantQueryInput
		if bindJSON(c, &input) {
			input.RequesterID = actor.UserID
			if input.Requester == "" {
				input.Requester = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.AskAssistant(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/inventory-ledger", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles...); !ok {
			return
		}
		item, err := repo.InventoryLedger(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/materials/scan/:code", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.MaterialByQRCode(c.Request.Context(), c.Param("code"))
		respond(c, item, err)
	})
	api.GET("/material-requests", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.MaterialRequests(c.Request.Context())
		item = filterMaterialRequestsForActor(actor, item)
		respond(c, item, err)
	})
	api.GET("/material-purchases", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.MaterialPurchases(c.Request.Context())
		item = filterMaterialPurchasesForActor(actor, item)
		respond(c, item, err)
	})
	api.GET("/material-damages", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.MaterialDamages(c.Request.Context())
		item = filterMaterialDamagesForActor(actor, item)
		respond(c, item, err)
	})
	api.GET("/maintenance", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles...); !ok {
			return
		}
		item, err := repo.MaintenanceOrders(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/audit-events", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles...); !ok {
			return
		}
		item, err := repo.AuditEvents(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/operations", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles...); !ok {
			return
		}
		item, err := repo.Operations(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/operations/export.csv", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles...); !ok {
			return
		}
		item, err := repo.Operations(c.Request.Context())
		if err != nil {
			respond(c, nil, err)
			return
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-operations.csv")
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write([]string{"section", "name", "value"})
		_ = writer.Write([]string{"dashboard", "today_reservations", strconv.Itoa(item.Dashboard.TodayReservations)})
		_ = writer.Write([]string{"dashboard", "pending_approvals", strconv.Itoa(item.Dashboard.PendingApprovals)})
		_ = writer.Write([]string{"dashboard", "active_instruments", strconv.Itoa(item.Dashboard.ActiveInstruments)})
		_ = writer.Write([]string{"dashboard", "monthly_revenue", fmt.Sprintf("%.2f", item.Dashboard.MonthlyRevenue)})
		_ = writer.Write([]string{"operations", "in_use_instruments", strconv.Itoa(item.InUseInstruments)})
		_ = writer.Write([]string{"operations", "alert_count", strconv.Itoa(item.AlertCount)})
		for _, point := range item.ReservationTrend {
			_ = writer.Write([]string{"reservation_trend", point.Hour, strconv.Itoa(point.Count)})
		}
		for _, load := range item.InstrumentLoads {
			_ = writer.Write([]string{"instrument_load", load.InstrumentName, fmt.Sprintf("%.2f", load.Hours)})
		}
		for _, metric := range item.ApprovalEfficiency {
			_ = writer.Write([]string{"approval_efficiency", metric.Label, fmt.Sprintf("%.2f", metric.Hours)})
		}
		for _, alert := range item.Alerts {
			_ = writer.Write([]string{"alert", alert.Source, alert.Body})
		}
		writer.Flush()
	})

	api.POST("/email-verification-codes", func(c *gin.Context) {
		var input store.EmailVerificationCodeInput
		if bindJSON(c, &input) {
			item, err := repo.RequestEmailVerificationCode(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.POST("/register", func(c *gin.Context) {
		var input store.RegisterInput
		if bindJSON(c, &input) {
			item, err := repo.Register(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.POST("/verify-email", func(c *gin.Context) {
		var input struct {
			Token string `json:"token"`
		}
		if bindJSON(c, &input) {
			item, err := repo.VerifyEmail(c.Request.Context(), input.Token)
			respond(c, item, err)
		}
	})
	api.POST("/login", func(c *gin.Context) {
		var input store.LoginInput
		if bindJSON(c, &input) {
			item, err := repo.Login(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.POST("/dingtalk/quick-login", func(c *gin.Context) {
		var input store.DingTalkQuickLoginInput
		if bindJSON(c, &input) {
			item, err := repo.DingTalkQuickLogin(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/me", func(c *gin.Context) {
		token, ok := bearerToken(c)
		if !ok {
			return
		}
		item, err := repo.CurrentUser(c.Request.Context(), token)
		respond(c, item, err)
	})
	api.GET("/me/dingtalk-binding", func(c *gin.Context) {
		actor, ok := requireAuthenticated(c, repo)
		if !ok {
			return
		}
		item, err := repo.CurrentUserDingTalkBinding(c.Request.Context(), actor.UserID)
		respond(c, item, err)
	})
	api.POST("/me/dingtalk-binding", func(c *gin.Context) {
		actor, ok := requireAuthenticated(c, repo)
		if !ok {
			return
		}
		var input store.DingTalkBindingInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.BindCurrentUserDingTalk(c.Request.Context(), actor.UserID, input)
			respond(c, item, err)
		}
	})
	api.DELETE("/me/dingtalk-binding", func(c *gin.Context) {
		actor, ok := requireAuthenticated(c, repo)
		if !ok {
			return
		}
		item, err := repo.UnbindCurrentUserDingTalk(c.Request.Context(), actor.UserID, actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/me/profile", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.UserProfileInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.UpdateCurrentUserProfile(c.Request.Context(), actor.UserID, input)
			respond(c, item, err)
		}
	})
	api.PATCH("/me/password", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.PasswordChangeInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			if err := repo.ChangePassword(c.Request.Context(), actor.UserID, input); err != nil {
				respond(c, gin.H{"ok": false}, err)
				return
			}
			respond(c, gin.H{"ok": true}, nil)
		}
	})
	api.POST("/logout", func(c *gin.Context) {
		token, ok := bearerToken(c)
		if !ok {
			return
		}
		respond(c, gin.H{"ok": true}, repo.Logout(c.Request.Context(), token))
	})
	api.POST("/logout-all", func(c *gin.Context) {
		actor, ok := requireAuthenticated(c, repo)
		if !ok {
			return
		}
		respond(c, gin.H{"ok": true}, repo.LogoutAll(c.Request.Context(), actor.UserID))
	})
	api.POST("/reservations", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.ReservationInput
		if bindJSON(c, &input) {
			input.UserID = actor.UserID
			input.UserName = actor.Name
			item, err := repo.CreateReservation(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.POST("/reservations/batch", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.ReservationBatchInput
		if bindJSON(c, &input) {
			input.UserID = actor.UserID
			input.UserName = actor.Name
			items, err := repo.CreateReservationBatch(c.Request.Context(), input)
			respond(c, items, err)
		}
	})
	api.PATCH("/reservations/:id/approve", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "group_leader", "tenant_admin", "lab_admin", "super_admin")
		if !ok {
			return
		}
		if !authorizeReservationReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		_ = c.ShouldBindJSON(&input)
		item, err := repo.ApproveReservation(c.Request.Context(), c.Param("id"), true, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/reservations/:id/reject", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "group_leader", "tenant_admin", "lab_admin", "super_admin")
		if !ok {
			return
		}
		if !authorizeReservationReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		_ = c.ShouldBindJSON(&input)
		item, err := repo.ApproveReservation(c.Request.Context(), c.Param("id"), false, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/reservations/:id/check-in", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		if !authorizeReservationOwnerOrAdmin(c, repo, actor, c.Param("id")) {
			return
		}
		item, err := repo.CheckInReservation(c.Request.Context(), c.Param("id"))
		respond(c, item, err)
	})
	api.PATCH("/reservations/:id/complete", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		if !authorizeReservationOwnerOrAdmin(c, repo, actor, c.Param("id")) {
			return
		}
		item, err := repo.CompleteReservation(c.Request.Context(), c.Param("id"))
		respond(c, item, err)
	})
	api.PATCH("/reservations/:id/check-out", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		if !authorizeReservationOwnerOrAdmin(c, repo, actor, c.Param("id")) {
			return
		}
		item, err := repo.CompleteReservation(c.Request.Context(), c.Param("id"))
		respond(c, item, err)
	})
	api.PATCH("/reservations/:id/cancel", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		reservation, ok := authorizeReservationCancel(c, repo, actor, c.Param("id"))
		if !ok {
			return
		}
		var input struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&input)
		bypassCutoff := isAdmin(actor) || canReviewGroup(actor, reservation.GroupName)
		item, err := repo.CancelReservation(c.Request.Context(), c.Param("id"), input.Reason, bypassCutoff)
		respond(c, item, err)
	})
	api.PATCH("/users/:id/review", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.UserReviewInput
		if bindJSON(c, &input) {
			if actor.Role != "super_admin" && strings.TrimSpace(input.TenantID) != "" && strings.TrimSpace(input.TenantID) != actor.TenantID {
				c.JSON(http.StatusForbidden, gin.H{"error": "only system super admin can change tenant"})
				return
			}
			input.Actor = actor.Name
			input.ActorRole = actor.Role
			item, err := repo.ReviewUser(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.POST("/users/:id/memberships", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "super_admin")
		if !ok {
			return
		}
		var input store.UserMembershipInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveUserMembership(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.DELETE("/users/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		if c.Param("id") == actor.UserID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete current account"})
			return
		}
		users, err := repo.Users(c.Request.Context())
		if err != nil {
			respond(c, nil, err)
			return
		}
		for _, user := range users {
			if user.ID != c.Param("id") {
				continue
			}
			if actor.Role != "super_admin" && (user.Role == "tenant_admin" || user.Role == "lab_admin" || user.Role == "super_admin") {
				c.JSON(http.StatusForbidden, gin.H{"error": "only system super admin can manage administrator roles"})
				return
			}
			break
		}
		item, err := repo.DeleteUser(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/notifications/:id/read", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.MarkNotificationRead(c.Request.Context(), c.Param("id"), actor)
		respond(c, item, err)
	})
	api.DELETE("/notifications/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		ctx, ok := tenantAdminRequestContext(c, repo, actor)
		if !ok {
			return
		}
		item, err := repo.DeleteNotification(ctx, c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.POST("/notifications", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		ctx, ok := tenantAdminRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.AnnouncementInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.Announce(ctx, input)
			respond(c, item, err)
		}
	})
	api.PATCH("/notifications/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		ctx, ok := tenantAdminRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.AnnouncementInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.UpdateNotification(ctx, c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.POST("/ledger/adjustments", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, financeAdminRoles...)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.LedgerAdjustmentInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.AdjustLedger(ctx, input)
			respond(c, item, err)
		}
	})
	api.POST("/materials", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.MaterialInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveMaterial(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.POST("/materials/categories", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.MaterialCategoryInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveMaterialCategory(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/materials/categories/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.MaterialCategoryInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveMaterialCategory(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.DELETE("/materials/categories/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		item, err := repo.DeleteMaterialCategory(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/materials/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.MaterialInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveMaterial(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.DELETE("/materials/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		item, err := repo.DeleteMaterial(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.POST("/materials/:id/stock-adjustments", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.StockAdjustmentInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.AdjustMaterialStock(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.POST("/materials/:id/alert-actions", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		var input store.MaterialAlertActionInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.CreateMaterialAlertAction(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.POST("/material-requests", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.MaterialRequestInput
		if bindJSON(c, &input) {
			input.RequesterID = actor.UserID
			input.Requester = actor.Name
			item, err := repo.CreateMaterialRequest(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.PATCH("/material-requests/:id/approve", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "group_leader", "material_admin", "tenant_admin", "lab_admin", "super_admin")
		if !ok {
			return
		}
		if !authorizeMaterialRequestReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		_ = c.ShouldBindJSON(&input)
		item, err := repo.ApproveMaterialRequest(c.Request.Context(), c.Param("id"), true, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-requests/:id/reject", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "group_leader", "material_admin", "tenant_admin", "lab_admin", "super_admin")
		if !ok {
			return
		}
		if !authorizeMaterialRequestReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		_ = c.ShouldBindJSON(&input)
		item, err := repo.ApproveMaterialRequest(c.Request.Context(), c.Param("id"), false, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-requests/:id/outbound", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		item, err := repo.OutboundMaterialRequest(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/material-requests/:id/cancel", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		if !authorizeMaterialRequestOwnerOrAdmin(c, repo, actor, c.Param("id")) {
			return
		}
		item, err := repo.CancelMaterialRequest(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.POST("/material-purchases", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.MaterialPurchaseInput
		if bindJSON(c, &input) {
			input.RequesterID = actor.UserID
			input.Requester = actor.Name
			item, err := repo.CreateMaterialPurchase(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.PATCH("/material-purchases/:id/approve", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "group_leader", "material_admin", "tenant_admin", "lab_admin", "super_admin")
		if !ok {
			return
		}
		if !authorizeMaterialPurchaseReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		_ = c.ShouldBindJSON(&input)
		item, err := repo.ApproveMaterialPurchase(c.Request.Context(), c.Param("id"), true, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-purchases/:id/reject", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "group_leader", "material_admin", "tenant_admin", "lab_admin", "super_admin")
		if !ok {
			return
		}
		if !authorizeMaterialPurchaseReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		_ = c.ShouldBindJSON(&input)
		item, err := repo.ApproveMaterialPurchase(c.Request.Context(), c.Param("id"), false, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-purchases/:id/order", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		item, err := repo.MarkMaterialPurchaseOrdered(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/material-purchases/:id/receive", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		item, err := repo.ReceiveMaterialPurchase(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/material-purchases/:id/cancel", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		if !authorizeMaterialPurchaseOwnerOrAdmin(c, repo, actor, c.Param("id")) {
			return
		}
		item, err := repo.CancelMaterialPurchase(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.POST("/material-damages", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.MaterialDamageInput
		if bindJSON(c, &input) {
			input.ReporterID = actor.UserID
			input.Reporter = actor.Name
			item, err := repo.CreateMaterialDamage(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.PATCH("/material-damages/:id/approve", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "group_leader", "material_admin", "tenant_admin", "lab_admin", "super_admin")
		if !ok {
			return
		}
		if !authorizeMaterialDamageReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		_ = c.ShouldBindJSON(&input)
		item, err := repo.ApproveMaterialDamage(c.Request.Context(), c.Param("id"), true, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-damages/:id/reject", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "group_leader", "material_admin", "tenant_admin", "lab_admin", "super_admin")
		if !ok {
			return
		}
		if !authorizeMaterialDamageReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		_ = c.ShouldBindJSON(&input)
		item, err := repo.ApproveMaterialDamage(c.Request.Context(), c.Param("id"), false, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-damages/:id/process", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles...)
		if !ok {
			return
		}
		item, err := repo.ProcessMaterialDamage(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/material-damages/:id/cancel", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		if !authorizeMaterialDamageOwnerOrAdmin(c, repo, actor, c.Param("id")) {
			return
		}
		item, err := repo.CancelMaterialDamage(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.POST("/maintenance", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input store.MaintenanceInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.CreateMaintenanceOrder(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.PATCH("/maintenance/:id/start", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		item, err := repo.StartMaintenanceOrder(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/maintenance/:id/cancel", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&input)
		item, err := repo.CancelMaintenanceOrder(c.Request.Context(), c.Param("id"), input.Reason, actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/maintenance/:id/complete", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		var input struct {
			Result string `json:"result"`
		}
		_ = c.ShouldBindJSON(&input)
		item, err := repo.CompleteMaintenanceOrder(c.Request.Context(), c.Param("id"), input.Result, actor.Name)
		respond(c, item, err)
	})
}

type callerFunc[T any] func(context.Context) (T, error)

func caller[T any](fn func(context.Context) (T, error)) callerFunc[T] {
	return fn
}

func get[T any](fn callerFunc[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := fn(c.Request.Context())
		respond(c, item, err)
	}
}

func bindJSON(c *gin.Context, input any) bool {
	if err := c.ShouldBindJSON(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json payload"})
		return false
	}
	return true
}

func intQuery(c *gin.Context, key string, fallback int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func bearerToken(c *gin.Context) (string, bool) {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return "", false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return "", false
	}
	return token, true
}

func tenantContextMiddleware(repo authRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenant := store.TenantContext{TenantID: "00000000-0000-0000-0000-000000000001"}
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(header, "Bearer ") {
			token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			if token != "" {
				if user, err := repo.CurrentUser(c.Request.Context(), token); err == nil {
					tenant = store.TenantContext{
						TenantID:       user.TenantID,
						TenantName:     user.TenantName,
						FinanceEnabled: user.FinanceEnabled,
						AllTenants:     user.Role == "super_admin",
						Actor: store.Actor{
							UserID:         user.ID,
							TenantID:       user.TenantID,
							TenantName:     user.TenantName,
							Name:           user.Name,
							Email:          user.Email,
							Department:     user.Department,
							Role:           user.Role,
							Status:         user.Status,
							GroupName:      user.GroupName,
							EmailVerified:  user.EmailVerified,
							FinanceEnabled: user.FinanceEnabled,
							AuthEpoch:      user.AuthEpoch,
						},
					}
				}
			}
		}
		c.Request = c.Request.WithContext(store.WithTenantContext(c.Request.Context(), tenant))
		c.Next()
	}
}

func optionalCurrentUser(c *gin.Context, repo authRepository) (store.User, bool) {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return store.User{}, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return store.User{}, false
	}
	user, err := repo.CurrentUser(c.Request.Context(), token)
	if err != nil {
		return store.User{}, false
	}
	return user, true
}

func requireAuthenticated(c *gin.Context, repo authRepository) (store.Actor, bool) {
	token, ok := bearerToken(c)
	if !ok {
		return store.Actor{}, false
	}
	user, err := repo.CurrentUser(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
		return store.Actor{}, false
	}
	return store.Actor{
		UserID:         user.ID,
		TenantID:       user.TenantID,
		TenantName:     user.TenantName,
		Name:           user.Name,
		Email:          user.Email,
		Department:     user.Department,
		Role:           user.Role,
		Status:         user.Status,
		GroupName:      user.GroupName,
		EmailVerified:  user.EmailVerified,
		FinanceEnabled: user.FinanceEnabled,
		AuthEpoch:      user.AuthEpoch,
	}, true
}

func requireActiveUser(c *gin.Context, repo authRepository) (store.Actor, bool) {
	actor, ok := requireAuthenticated(c, repo)
	if !ok {
		return store.Actor{}, false
	}
	if actor.Status != "active" || actor.Role == "unassigned" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is not approved"})
		return store.Actor{}, false
	}
	if !actor.EmailVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "email is not verified"})
		return store.Actor{}, false
	}
	return actor, true
}

func requireAnyRole(c *gin.Context, repo authRepository, roles ...string) (store.Actor, bool) {
	actor, ok := requireActiveUser(c, repo)
	if !ok {
		return store.Actor{}, false
	}
	for _, allowed := range roles {
		if actor.Role == allowed {
			return actor, true
		}
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return store.Actor{}, false
}

func requireFinanceEnabled(c *gin.Context, actor store.Actor) bool {
	if actor.Role == "super_admin" || actor.FinanceEnabled {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "finance module is disabled for this tenant"})
	return false
}

func financeRequestContext(c *gin.Context, repo repository, actor store.Actor) (context.Context, bool) {
	if actor.Role != "super_admin" {
		if !requireFinanceEnabled(c, actor) {
			return nil, false
		}
		return c.Request.Context(), true
	}

	tenantID := strings.TrimSpace(c.Query("tenantId"))
	if tenantID == "" || tenantID == actor.TenantID {
		if !actor.FinanceEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "finance module is disabled for this tenant"})
			return nil, false
		}
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       actor.TenantID,
			TenantName:     actor.TenantName,
			FinanceEnabled: actor.FinanceEnabled,
		}), true
	}

	tenants, err := repo.Tenants(c.Request.Context())
	if err != nil {
		respond(c, nil, err)
		return nil, false
	}
	for _, tenant := range tenants {
		if tenant.ID != tenantID {
			continue
		}
		if tenant.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant is disabled"})
			return nil, false
		}
		if !tenant.FinanceEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "finance module is disabled for this tenant"})
			return nil, false
		}
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       tenant.ID,
			TenantName:     tenant.Name,
			FinanceEnabled: tenant.FinanceEnabled,
		}), true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "tenant not found"})
	return nil, false
}

func organizationUnitRequestContext(c *gin.Context, repo repository, actor store.Actor) (context.Context, bool) {
	tenantID := strings.TrimSpace(c.Query("tenantId"))
	if actor.Role != "super_admin" || tenantID == "" || tenantID == actor.TenantID {
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       actor.TenantID,
			TenantName:     actor.TenantName,
			FinanceEnabled: actor.FinanceEnabled,
		}), true
	}

	tenants, err := repo.Tenants(c.Request.Context())
	if err != nil {
		respond(c, nil, err)
		return nil, false
	}
	for _, tenant := range tenants {
		if tenant.ID != tenantID {
			continue
		}
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       tenant.ID,
			TenantName:     tenant.Name,
			FinanceEnabled: tenant.FinanceEnabled,
		}), true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "tenant not found"})
	return nil, false
}

func tenantAdminRequestContext(c *gin.Context, repo repository, actor store.Actor) (context.Context, bool) {
	tenantID := strings.TrimSpace(c.Query("tenantId"))
	if actor.Role != "super_admin" || tenantID == "" || tenantID == actor.TenantID {
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       actor.TenantID,
			TenantName:     actor.TenantName,
			FinanceEnabled: actor.FinanceEnabled,
			AllTenants:     false,
			Actor:          actor,
		}), true
	}

	tenants, err := repo.Tenants(c.Request.Context())
	if err != nil {
		respond(c, nil, err)
		return nil, false
	}
	for _, tenant := range tenants {
		if tenant.ID != tenantID {
			continue
		}
		if tenant.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant is disabled"})
			return nil, false
		}
		return store.WithTenantContext(c.Request.Context(), store.TenantContext{
			TenantID:       tenant.ID,
			TenantName:     tenant.Name,
			FinanceEnabled: tenant.FinanceEnabled,
			AllTenants:     false,
			Actor:          actor,
		}), true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "tenant not found"})
	return nil, false
}

func isAdmin(actor store.Actor) bool {
	return actor.Role == "tenant_admin" || actor.Role == "lab_admin" || actor.Role == "super_admin"
}

func canManageMaterials(actor store.Actor) bool {
	return actor.Role == "material_admin" || isAdmin(actor)
}

func canManageFinance(actor store.Actor) bool {
	return actor.Role == "finance_admin" || isAdmin(actor)
}

func canManageTraining(actor store.Actor) bool {
	return actor.Role == "material_admin" || isAdmin(actor)
}

func canAccessReservation(actor store.Actor, item store.Reservation) bool {
	if isAdmin(actor) {
		return true
	}
	if actor.Role == "group_leader" && actor.GroupName != "" && actor.GroupName == item.GroupName {
		return true
	}
	return item.UserID != "" && item.UserID == actor.UserID
}

func canReviewGroup(actor store.Actor, groupName string) bool {
	if isAdmin(actor) {
		return true
	}
	return actor.Role == "group_leader" && actor.GroupName != "" && actor.GroupName == groupName
}

func canReviewMaterialGroup(actor store.Actor, groupName string) bool {
	if canManageMaterials(actor) {
		return true
	}
	return actor.Role == "group_leader" && actor.GroupName != "" && actor.GroupName == groupName
}

func filterReservationsForActor(actor store.Actor, items []store.Reservation) []store.Reservation {
	filtered := make([]store.Reservation, 0, len(items))
	for _, item := range items {
		if canAccessReservation(actor, item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterLedgerForActor(actor store.Actor, items []store.LedgerEntry) []store.LedgerEntry {
	if canManageFinance(actor) {
		return items
	}
	filtered := make([]store.LedgerEntry, 0, len(items))
	for _, item := range items {
		if item.UserID == actor.UserID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterFinancialAccountsForActor(actor store.Actor, items []store.FinancialAccount) []store.FinancialAccount {
	if canManageFinance(actor) {
		return items
	}
	filtered := make([]store.FinancialAccount, 0, 1)
	for _, item := range items {
		if item.UserID == actor.UserID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterMaterialRequestsForActor(actor store.Actor, items []store.MaterialRequest) []store.MaterialRequest {
	if canManageMaterials(actor) {
		return items
	}
	filtered := make([]store.MaterialRequest, 0, len(items))
	for _, item := range items {
		if actor.Role == "group_leader" && item.GroupName == actor.GroupName {
			filtered = append(filtered, item)
			continue
		}
		if item.RequesterID == actor.UserID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterMaterialPurchasesForActor(actor store.Actor, items []store.MaterialPurchase) []store.MaterialPurchase {
	if canManageMaterials(actor) {
		return items
	}
	filtered := make([]store.MaterialPurchase, 0, len(items))
	for _, item := range items {
		if actor.Role == "group_leader" && item.GroupName == actor.GroupName {
			filtered = append(filtered, item)
			continue
		}
		if item.RequesterID == actor.UserID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterMaterialDamagesForActor(actor store.Actor, items []store.MaterialDamage) []store.MaterialDamage {
	if canManageMaterials(actor) {
		return items
	}
	filtered := make([]store.MaterialDamage, 0, len(items))
	for _, item := range items {
		if actor.Role == "group_leader" && item.GroupName == actor.GroupName {
			filtered = append(filtered, item)
			continue
		}
		if item.ReporterID == actor.UserID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func canAccessNotification(actor store.Actor, item store.Notification) bool {
	switch item.TargetScope {
	case "", "global":
		return true
	case "personal":
		return item.UserID != "" && item.UserID == actor.UserID
	case "group":
		return item.GroupName != "" && item.GroupName == actor.GroupName
	case "department":
		return item.Department != "" && item.Department == actor.Department
	default:
		return isAdmin(actor)
	}
}

func filterNotificationsForActor(actor store.Actor, items []store.Notification) []store.Notification {
	filtered := make([]store.Notification, 0, len(items))
	for _, item := range items {
		if canAccessNotification(actor, item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func authorizeReservationReview(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findReservation(c, repo, id)
	if !ok {
		return false
	}
	if canReviewGroup(actor, item.GroupName) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeReservationOwnerOrAdmin(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findReservation(c, repo, id)
	if !ok {
		return false
	}
	if isAdmin(actor) || item.UserID == actor.UserID {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeReservationCancel(c *gin.Context, repo repository, actor store.Actor, id string) (store.Reservation, bool) {
	item, ok := findReservation(c, repo, id)
	if !ok {
		return store.Reservation{}, false
	}
	if isAdmin(actor) || item.UserID == actor.UserID || canReviewGroup(actor, item.GroupName) {
		return item, true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return store.Reservation{}, false
}

func authorizeMaterialRequestReview(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialRequest(c, repo, id)
	if !ok {
		return false
	}
	if canReviewMaterialGroup(actor, item.GroupName) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeMaterialRequestOwnerOrAdmin(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialRequest(c, repo, id)
	if !ok {
		return false
	}
	if canManageMaterials(actor) || item.RequesterID == actor.UserID {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeMaterialPurchaseReview(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialPurchase(c, repo, id)
	if !ok {
		return false
	}
	if canReviewMaterialGroup(actor, item.GroupName) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeMaterialPurchaseOwnerOrAdmin(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialPurchase(c, repo, id)
	if !ok {
		return false
	}
	if canManageMaterials(actor) || item.RequesterID == actor.UserID {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeMaterialDamageReview(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialDamage(c, repo, id)
	if !ok {
		return false
	}
	if canReviewMaterialGroup(actor, item.GroupName) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeMaterialDamageOwnerOrAdmin(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	item, ok := findMaterialDamage(c, repo, id)
	if !ok {
		return false
	}
	if canManageMaterials(actor) || item.ReporterID == actor.UserID {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	return false
}

func authorizeNotificationAccess(c *gin.Context, repo repository, actor store.Actor, id string) bool {
	items, err := repo.Notifications(c.Request.Context(), actor)
	if err != nil {
		respond(c, nil, err)
		return false
	}
	for _, item := range items {
		if item.ID == id {
			if canAccessNotification(actor, item) {
				return true
			}
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return false
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
	return false
}

func findReservation(c *gin.Context, repo repository, id string) (store.Reservation, bool) {
	item, err := repo.Reservation(c.Request.Context(), id)
	if err != nil {
		respond(c, nil, err)
		return store.Reservation{}, false
	}
	return item, true
}

func findMaterialRequest(c *gin.Context, repo repository, id string) (store.MaterialRequest, bool) {
	item, err := repo.MaterialRequest(c.Request.Context(), id)
	if err != nil {
		respond(c, nil, err)
		return store.MaterialRequest{}, false
	}
	return item, true
}

func findMaterialPurchase(c *gin.Context, repo repository, id string) (store.MaterialPurchase, bool) {
	item, err := repo.MaterialPurchase(c.Request.Context(), id)
	if err != nil {
		respond(c, nil, err)
		return store.MaterialPurchase{}, false
	}
	return item, true
}

func findMaterialDamage(c *gin.Context, repo repository, id string) (store.MaterialDamage, bool) {
	item, err := repo.MaterialDamage(c.Request.Context(), id)
	if err != nil {
		respond(c, nil, err)
		return store.MaterialDamage{}, false
	}
	return item, true
}

func materialProductTypeLabel(productType string) string {
	labels := map[string]string{
		"consumable": "耗材",
		"reagent":    "试剂",
		"standard":   "标准品",
	}
	if label, ok := labels[productType]; ok {
		return label
	}
	return productType
}

func materialStatusLabel(status string) string {
	labels := map[string]string{
		"normal":               "正常",
		"near_expiry":          "临期",
		"low":                  "低库存",
		"expired":              "过期",
		"open_expired":         "开封超期",
		"freeze_thaw_exceeded": "冻融超限",
		"damaged":              "损毁",
		"disabled":             "停用",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}

func materialRequestStatusLabel(status string) string {
	labels := map[string]string{
		"pending":   "待审批",
		"approved":  "已通过",
		"rejected":  "已拒绝",
		"outbound":  "已出库",
		"cancelled": "已取消",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}

func materialDamageStatusLabel(status string) string {
	labels := map[string]string{
		"pending":   "待审核",
		"approved":  "已通过",
		"rejected":  "已拒绝",
		"processed": "已处理",
		"cancelled": "已取消",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}

func materialLocation(item store.Material) string {
	parts := []string{item.StorageRoom, item.StorageCabinet, item.StorageLayer, item.StorageSlot}
	visible := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			visible = append(visible, part)
		}
	}
	return strings.Join(visible, " / ")
}

func materialImportTemplateHeader() []string {
	return []string{
		"资源名称",
		"资源类型",
		"一级目录",
		"二级目录",
		"规格",
		"单位",
		"单价",
		"库存",
		"低库存线",
		"供应商",
		"生产商",
		"批号",
		"货号",
		"CAS号",
		"级别",
		"浓度",
		"母液ID",
		"稀释倍数",
		"配制方法",
		"保存条件",
		"库房/冰箱",
		"柜/架",
		"层/盒",
		"孔位",
		"招标合同",
		"合同序号",
		"资源证书地址",
		"标准证书地址",
		"附件地址",
		"二维码编码",
		"有效期",
		"开封日期",
		"开封有效天数",
		"冻融次数",
		"冻融上限",
		"是否需要审批",
		"临期预警天数",
		"状态",
	}
}

func boolLabel(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func respond(c *gin.Context, payload any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, payload)
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	if message, ok := clientSafeError(err); ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	slog.Error("api request failed", "method", c.Request.Method, "path", c.FullPath(), "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func clientSafeError(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return "", false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "", false
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "", false
	}
	prefixes := []string{
		"invalid ",
		"missing ",
		"minimum ",
		"reservation ",
		"instrument ",
		"user ",
		"email ",
		"account ",
		"tenant ",
		"current ",
		"personal ",
		"group ",
		"department ",
		"organization ",
		"config ",
		"material ",
		"maintenance ",
		"insufficient ",
		"dingtalk ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(message, prefix) {
			return message, true
		}
	}
	return "", false
}
