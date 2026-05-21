package httpapi

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/xuri/excelize/v2"

	"lirs/apps/server/internal/store"
)

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
	registerUploadRoutes(router, api, repo)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
	registerGraphMailRoutes(api, repo)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
	registerDingTalkNotificationRoutes(api, repo)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		item, err := repo.DeleteInstrument(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/footer-settings", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		if err != nil {
			respond(c, nil, err)
			return
		}
		item = filterReservationsForActor(actor, item)
		respond(c, item, nil)
	})
	api.GET("/users", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, userReaderRoles()...); !ok {
			return
		}
		item, err := repo.Users(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/users", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.UserCreateInput
		if bindJSON(c, &input) {
			if actor.Role != "super_admin" && strings.TrimSpace(input.TenantID) != "" && strings.TrimSpace(input.TenantID) != actor.TenantID {
				c.JSON(http.StatusForbidden, gin.H{"error": "only system super admin can create users for another tenant"})
				return
			}
			input.Actor = actor.Name
			input.ActorRole = actor.Role
			item, err := repo.CreateUser(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/organization-units", func(c *gin.Context) {
		ctx := c.Request.Context()
		user, hasUser := optionalCurrentUser(c, repo)
		if tenantID := strings.TrimSpace(c.Query("tenantId")); tenantID != "" {
			if !hasUser || user.Role == "super_admin" || tenantID == user.TenantID {
				ctx = store.WithTenantContext(ctx, store.TenantContext{TenantID: tenantID})
			}
		} else if hasUser {
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		if source := strings.TrimSpace(c.Query("source")); source != "" {
			ctx = store.WithNotificationSourceContext(ctx, source)
		}
		if isAdmin(actor) {
			var contextOK bool
			ctx, contextOK = tenantAdminRequestContext(c, repo, actor)
			if !contextOK {
				return
			}
			if source := strings.TrimSpace(c.Query("source")); source != "" {
				ctx = store.WithNotificationSourceContext(ctx, source)
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
		ctx := c.Request.Context()
		if source := strings.TrimSpace(c.Query("source")); source != "" {
			ctx = store.WithNotificationSourceContext(ctx, source)
		}
		count, err := repo.MarkAllNotificationsRead(ctx, actor)
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
		if err != nil {
			respond(c, nil, err)
			return
		}
		item = filterLedgerForActor(actor, item)
		respond(c, item, nil)
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
		writeCSVBOM(c)
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
		if err != nil {
			respond(c, nil, err)
			return
		}
		item = filterFinancialAccountsForActor(actor, item)
		respond(c, item, nil)
	})
	api.POST("/financial-accounts", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, financeAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, financeAdminRoles()...)
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
	api.GET("/materials", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.Materials(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/materials/analytics", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
			return
		}
		item, err := repo.MaterialAnalytics(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/materials/categories", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
			return
		}
		item, err := repo.MaterialCategories(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/materials/alert-actions", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
			return
		}
		item, err := repo.MaterialAlertActions(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/materials/import-template.csv", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
			return
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-materials-import-template.csv")
		writeCSVBOM(c)
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write(materialImportTemplateHeader())
		_ = writer.Write([]string{"PCR-0001", "示例铅标准物质", "标准品", "单元素标准品", "金属元素", "100ug/L 50mL", "瓶", "80", "1", "1", "国家标准物质中心", "NIM", "WORK-PB-001", "GBW(E)080129-D", "7439-92-1", "CRM", "100ug/L", "2-8°C 避光", "标准品库", "防爆冰箱", "二层", "A09", "2026-标准品采购项目 编号：STD-2026-066", "首次入库备注", "", "/files/certs/pb-standard.pdf", "", "WORK-PB-001", "2026-08-31", "2026-05-15", "30", "0", "5", "是", "30", "正常"})
		writer.Flush()
	})
	api.GET("/materials/export.csv", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		writeCSVBOM(c)
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write([]string{"资源名称", "资源类型", "一级目录", "二级目录", "CAS号", "规格", "单位", "库存", "低库存线", "损毁数", "供应商", "生产商", "批号", "货号", "库位", "采购项目名称及编号", "备注", "有效期", "开封日期", "开封到期", "冻融次数", "审批", "二维码", "状态"})
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
				materialLocation(item),
				firstNonEmptyString(item.TenderContract, item.ContractNo),
				item.Remark,
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
	api.POST("/materials/import", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 8<<20))
		if err != nil {
			respond(c, nil, err)
			return
		}
		item, err := repo.ImportMaterials(c.Request.Context(), store.MaterialImportInput{
			Filename: c.Query("filename"),
			Content:  body,
			Actor:    actor.Name,
		})
		respond(c, item, err)
	})
	registerProcurementRoutes(api, repo)
	api.GET("/material-requests/export.csv", func(c *gin.Context) {
		materialRequestsMonthlyExport(c, repo, false)
	})
	api.GET("/material-requests/monthly-export.xlsx", func(c *gin.Context) {
		materialRequestsMonthlyExport(c, repo, true)
	})
	api.GET("/material-requests/export.xlsx", func(c *gin.Context) {
		materialRequestsMonthlyExport(c, repo, true)
	})
	api.GET("/material-damages/export.csv", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		writeCSVBOM(c)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
			return
		}
		item, err := repo.TrainingPracticals(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/training/practicals", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
			return
		}
		item, err := repo.TrainingRules(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/training/rules", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles()...); !ok {
			return
		}
		item, err := repo.BusinessConfigs(c.Request.Context(), "workflow", c.Param("kind"))
		respond(c, item, err)
	})
	api.POST("/workflows/:kind", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, financeAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, financeAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, financeAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
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
		if err != nil {
			respond(c, nil, err)
			return
		}
		item = filterMaterialRequestsForActor(actor, item)
		respond(c, item, nil)
	})
	api.GET("/material-purchases", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.MaterialPurchases(c.Request.Context())
		if err != nil {
			respond(c, nil, err)
			return
		}
		item = filterMaterialPurchasesForActor(actor, item)
		respond(c, item, nil)
	})
	api.GET("/material-damages", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.MaterialDamages(c.Request.Context())
		if err != nil {
			respond(c, nil, err)
			return
		}
		item = filterMaterialDamagesForActor(actor, item)
		respond(c, item, nil)
	})
	api.GET("/maintenance", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles()...); !ok {
			return
		}
		item, err := repo.MaintenanceOrders(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/audit-events", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles()...); !ok {
			return
		}
		item, err := repo.AuditEvents(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/operations", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles()...); !ok {
			return
		}
		item, err := repo.Operations(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/operations/export.csv", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles()...); !ok {
			return
		}
		item, err := repo.Operations(c.Request.Context())
		if err != nil {
			respond(c, nil, err)
			return
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-operations.csv")
		writeCSVBOM(c)
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
	registerDingTalkLoginRoutes(api, repo)
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
		if !bindOptionalJSON(c, &input) {
			return
		}
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
		if !bindOptionalJSON(c, &input) {
			return
		}
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
		if !bindOptionalJSON(c, &input) {
			return
		}
		bypassCutoff := isAdmin(actor) || canReviewGroup(actor, reservation.GroupName)
		item, err := repo.CancelReservation(c.Request.Context(), c.Param("id"), input.Reason, bypassCutoff)
		respond(c, item, err)
	})
	api.PATCH("/users/:id/review", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, financeAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		item, err := repo.DeleteMaterialCategory(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/materials/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		item, err := repo.DeleteMaterial(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.POST("/materials/:id/stock-adjustments", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		if !bindOptionalJSON(c, &input) {
			return
		}
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
		if !bindOptionalJSON(c, &input) {
			return
		}
		item, err := repo.ApproveMaterialRequest(c.Request.Context(), c.Param("id"), false, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-requests/:id/outbound", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
	api.PATCH("/material-purchases/:id", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		ctx := c.Request.Context()
		if canManageMaterials(actor) {
			requestCtx, contextOK := materialWriteRequestContext(c, repo, actor)
			if !contextOK {
				return
			}
			ctx = requestCtx
		}
		if !authorizeMaterialPurchaseOwnerOrAdmin(c, repo, actor, c.Param("id")) {
			return
		}
		var input store.MaterialPurchaseUpdateInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.UpdateMaterialPurchase(ctx, c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.PATCH("/material-purchases/:id/approve", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		ctx, ok := materialWriteRequestContext(c, repo, actor)
		if !ok {
			return
		}
		if !authorizeMaterialPurchaseReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		if !bindOptionalJSON(c, &input) {
			return
		}
		item, err := repo.ApproveMaterialPurchase(ctx, c.Param("id"), true, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-purchases/:id/reject", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		ctx, ok := materialWriteRequestContext(c, repo, actor)
		if !ok {
			return
		}
		if !authorizeMaterialPurchaseReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		if !bindOptionalJSON(c, &input) {
			return
		}
		item, err := repo.ApproveMaterialPurchase(ctx, c.Param("id"), false, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-purchases/:id/return", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		ctx, ok := materialWriteRequestContext(c, repo, actor)
		if !ok {
			return
		}
		if !authorizeMaterialPurchaseReview(c, repo, actor, c.Param("id")) {
			return
		}
		var input struct {
			Comment string `json:"comment"`
		}
		if !bindOptionalJSON(c, &input) {
			return
		}
		item, err := repo.ReturnMaterialPurchase(ctx, c.Param("id"), actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-purchases/:id/order", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		ctx, ok := materialWriteRequestContext(c, repo, actor)
		if !ok {
			return
		}
		item, err := repo.MarkMaterialPurchaseOrdered(ctx, c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/material-purchases/:id/receive", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		ctx, ok := materialWriteRequestContext(c, repo, actor)
		if !ok {
			return
		}
		item, err := repo.ReceiveMaterialPurchase(ctx, c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/material-purchases/:id/cancel", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		ctx := c.Request.Context()
		if canManageMaterials(actor) {
			requestCtx, contextOK := materialWriteRequestContext(c, repo, actor)
			if !contextOK {
				return
			}
			ctx = requestCtx
		}
		if !authorizeMaterialPurchaseOwnerOrAdmin(c, repo, actor, c.Param("id")) {
			return
		}
		item, err := repo.CancelMaterialPurchase(ctx, c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.POST("/material-purchases/monthly-confirmations", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input struct {
			Month string `json:"month"`
		}
		if bindJSON(c, &input) {
			item, err := repo.ConfirmMaterialPurchaseMonth(c.Request.Context(), input.Month, actor.Name)
			respond(c, item, err)
		}
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
		if !bindOptionalJSON(c, &input) {
			return
		}
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
		if !bindOptionalJSON(c, &input) {
			return
		}
		item, err := repo.ApproveMaterialDamage(c.Request.Context(), c.Param("id"), false, actor.Name, input.Comment)
		respond(c, item, err)
	})
	api.PATCH("/material-damages/:id/process", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
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
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		item, err := repo.StartMaintenanceOrder(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/maintenance/:id/cancel", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input struct {
			Reason string `json:"reason"`
		}
		if !bindOptionalJSON(c, &input) {
			return
		}
		item, err := repo.CancelMaintenanceOrder(c.Request.Context(), c.Param("id"), input.Reason, actor.Name)
		respond(c, item, err)
	})
	api.PATCH("/maintenance/:id/complete", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input struct {
			Result string `json:"result"`
		}
		if !bindOptionalJSON(c, &input) {
			return
		}
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

func bindOptionalJSON(c *gin.Context, input any) bool {
	if c.Request == nil || c.Request.Body == nil || c.Request.ContentLength == 0 {
		return true
	}
	if err := c.ShouldBindJSON(input); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
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

func writeCSVBOM(c *gin.Context) {
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
}

func materialRequestsMonthlyExport(c *gin.Context, repo repository, xlsx bool) {
	actor, ok := requireActiveUser(c, repo)
	if !ok {
		return
	}
	month := strings.TrimSpace(c.Query("month"))
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	if _, err := time.Parse("2006-01", month); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "月份格式无效，请使用 YYYY-MM"})
		return
	}
	rows, err := repo.MaterialRequestsForMonth(c.Request.Context(), month)
	if err != nil {
		respond(c, nil, err)
		return
	}
	rows = filterMaterialRequestExportRowsForActor(actor, rows)
	if xlsx {
		writeMaterialRequestExportXLSX(c, month, rows)
		return
	}
	writeMaterialRequestExportCSV(c, month, rows)
}

func writeMaterialRequestExportCSV(c *gin.Context, month string, rows []store.MaterialRequestExportRow) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=lirs-material-requests-"+month+".csv")
	writeCSVBOM(c)
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"产品", "申请人", "课题组", "编号", "批次", "库位", "数量", "单位", "用途", "状态", "创建时间", "模板类型"})
	for _, item := range rows {
		templateType := "试剂领用记录表"
		if strings.Contains(item.MaterialName, "标准") {
			templateType = "标准品领用记录表"
		}
		_ = writer.Write([]string{item.MaterialName, item.Requester, item.GroupName, item.UnitCode, item.BatchNo, item.Location, strconv.Itoa(item.Quantity), item.Unit, item.Purpose, materialRequestStatusLabel(item.Status), item.CreatedAt.Format(time.RFC3339), templateType})
	}
	writer.Flush()
}

func writeMaterialRequestExportXLSX(c *gin.Context, month string, rows []store.MaterialRequestExportRow) {
	file, err := materialRequestExportWorkbook(month, rows)
	if err != nil {
		respond(c, nil, err)
		return
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		respond(c, nil, err)
		return
	}
	filename := month + "标准物质领用记录表.xlsx"
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="lirs-material-requests-`+month+`.xlsx"; filename*=UTF-8''`+url.PathEscape(filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer.Bytes())
}

func materialRequestExportWorkbook(month string, rows []store.MaterialRequestExportRow) (*excelize.File, error) {
	const sheet = "领用历史记录"
	file := excelize.NewFile()
	defaultSheet := file.GetSheetName(0)
	if err := file.SetSheetName(defaultSheet, sheet); err != nil {
		return nil, err
	}
	if err := file.SetDocProps(&excelize.DocProperties{
		Title:       month + "标准物质领用记录表",
		Subject:     "标准物质领用记录表",
		Creator:     "LIRS",
		Description: "按月导出的标准物质领用记录表",
		Language:    "zh-CN",
	}); err != nil {
		return nil, err
	}
	if err := file.MergeCell(sheet, "A1", "L1"); err != nil {
		return nil, err
	}
	if err := file.MergeCell(sheet, "A2", "L2"); err != nil {
		return nil, err
	}
	if err := file.SetColWidth(sheet, "A", "A", 14.625); err != nil {
		return nil, err
	}
	widths := map[string]float64{"B": 15, "C": 14.125, "D": 12.5, "E": 9.25, "F": 8, "G": 7, "H": 10, "I": 11.125, "J": 12.625, "K": 11.875, "L": 13.5}
	for col, width := range widths {
		if err := file.SetColWidth(sheet, col, col, width); err != nil {
			return nil, err
		}
	}
	_ = file.SetRowHeight(sheet, 1, 70.5)
	_ = file.SetRowHeight(sheet, 2, 63)
	_ = file.SetRowHeight(sheet, 3, 37.5)
	border := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
	}
	titleStyle, err := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "宋体", Size: 18, Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}
	metaStyle, err := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "宋体", Size: 11},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}
	headerStyle, err := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "宋体", Size: 11, Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}
	bodyStyle, err := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "宋体", Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    border,
	})
	if err != nil {
		return nil, err
	}
	dateFormat := "yyyy/m/d"
	dateStyle, err := file.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Family: "宋体", Size: 10},
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:       border,
		CustomNumFmt: &dateFormat,
	})
	if err != nil {
		return nil, err
	}

	_ = file.SetCellStyle(sheet, "A1", "L1", titleStyle)
	_ = file.SetCellStyle(sheet, "A2", "L2", metaStyle)
	_ = file.SetCellStyle(sheet, "A3", "L3", headerStyle)
	_ = file.SetCellValue(sheet, "A1", "无锡市疾病预防控制中心记录表格\n标准物质领用记录表")
	_ = file.SetCellValue(sheet, "A2", "文件编号：WXCDC-490-308         \n版次：第6版 第0次修订 \n 发布日期：2023年1月6日 \n")
	headers := []string{"品名", "标准号", "品牌", "规格", "存放地", "领用数量", "领用单位", "领用人", "领用时间", "批号", "有效期", "审批信息"}
	for index, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(index+1, 3)
		_ = file.SetCellValue(sheet, cell, header)
	}
	for index, item := range rows {
		row := index + 4
		_ = file.SetRowHeight(sheet, row, 30)
		values := []any{
			item.MaterialName,
			item.StandardNo,
			item.Brand,
			item.Spec,
			item.Location,
			float64(item.Quantity),
			item.Unit,
			item.Requester,
			item.CreatedAt,
			firstNonEmptyString(item.BatchNo, item.UnitCode),
			item.ExpiresAt,
			firstNonEmptyString(item.ApprovalInfo, materialRequestStatusLabel(item.Status)),
		}
		for col, value := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = file.SetCellValue(sheet, cell, value)
		}
		lastCell, _ := excelize.CoordinatesToCellName(len(headers), row)
		_ = file.SetCellStyle(sheet, "A"+strconv.Itoa(row), lastCell, bodyStyle)
		dateCell, _ := excelize.CoordinatesToCellName(9, row)
		_ = file.SetCellStyle(sheet, dateCell, dateCell, dateStyle)
	}
	if len(rows) == 0 {
		_ = file.SetRowHeight(sheet, 4, 30)
		_ = file.SetCellStyle(sheet, "A4", "L4", bodyStyle)
	}
	_ = file.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      3,
		TopLeftCell: "A4",
		ActivePane:  "bottomLeft",
		Selection:   []excelize.Selection{{SQRef: "A4:L4", ActiveCell: "A4", Pane: "bottomLeft"}},
	})
	return file, nil
}

func materialProductTypeLabel(productType string) string {
	labels := map[string]string{
		"consumable": "耗材",
		"reagent":    "试剂",
		"standard":   "标准品/标准物质",
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
		"可采购物资ID号",
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
		"保存条件",
		"库房/冰箱",
		"柜/架",
		"层/盒",
		"孔位",
		"采购项目名称及编号",
		"备注",
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
	if status, message, ok := postgresClientError(err); ok {
		c.JSON(status, gin.H{"error": message})
		return
	}
	if message, ok := clientSafeError(err); ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	slog.Error("api request failed", "method", c.Request.Method, "path", c.FullPath(), "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func postgresClientError(err error) (int, string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return 0, "", false
	}
	switch pgErr.Code {
	case "23505":
		return http.StatusConflict, "resource already exists", true
	case "23503":
		return http.StatusBadRequest, "referenced resource not found", true
	default:
		return 0, "", false
	}
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
		"mail ",
		"personal ",
		"group ",
		"department ",
		"organization ",
		"config ",
		"material ",
		"maintenance ",
		"insufficient ",
		"dingtalk ",
		"graph ",
		"标准品证书",
		"无法",
		"不支持",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(message, prefix) {
			return message, true
		}
	}
	return "", false
}
