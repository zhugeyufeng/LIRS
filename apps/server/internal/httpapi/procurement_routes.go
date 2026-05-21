package httpapi

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"lirs/apps/server/internal/store"
)

func registerProcurementRoutes(api *gin.RouterGroup, repo repository) {
	api.GET("/purchasable-materials", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.PurchasableMaterials(c.Request.Context())
		respond(c, item, err)
	})
	api.GET("/procurement-projects", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.ProcurementProjects(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/procurement-projects", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.ProcurementProjectInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveProcurementProject(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/procurement-projects/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.ProcurementProjectInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveProcurementProject(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.DELETE("/procurement-projects/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		item, err := repo.DeleteProcurementProject(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.POST("/purchasable-materials", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.PurchasableMaterialInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SavePurchasableMaterial(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/purchasable-materials/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.PurchasableMaterialInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SavePurchasableMaterial(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.DELETE("/purchasable-materials/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		item, err := repo.DeletePurchasableMaterial(c.Request.Context(), c.Param("id"), actor.Name)
		respond(c, item, err)
	})
	api.GET("/purchasable-materials/import-template.csv", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
			return
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-purchasable-materials-import-template.csv")
		writeCSVBOM(c)
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write(purchasableMaterialImportHeader())
		_ = writer.Write([]string{"", "", "9-病毒PCR试剂类采购项目 编号：WXCDCQTCG2021-029（满足采购人要求20210929）", "", "", "", "", "", "", "", ""})
		_ = writer.Write([]string{"PCR-0001", "1", "9-病毒PCR试剂类采购项目 编号：WXCDCQTCG2021-029（满足采购人要求20210929）", "病毒核酸检测试剂盒", "示例品牌", "200T/盒", "盒", "1200", "示例备注", "满足采购人要求", "1盒"})
		writer.Flush()
	})
	api.GET("/purchasable-materials/export.csv", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
			return
		}
		items, err := repo.PurchasableMaterials(c.Request.Context())
		if err != nil {
			respond(c, nil, err)
			return
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-purchasable-materials.csv")
		writeCSVBOM(c)
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write(purchasableMaterialImportHeader())
		for _, item := range items {
			_ = writer.Write([]string{item.IDNo, item.SequenceNo, item.ProcurementProject, item.ProjectName, item.Brand, item.Spec, item.Unit, fmt.Sprintf("%.2f", item.PurchasePrice), item.Remark, item.TechnicalRequirement, item.MinSpec})
		}
		writer.Flush()
	})
	api.POST("/purchasable-materials/import", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 8<<20))
		if err != nil {
			respond(c, nil, err)
			return
		}
		if len(body) == 8<<20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "material purchasable import failed: 文件超过 8MB 限制，请拆分后导入"})
			return
		}
		input := store.PurchasableMaterialImportInput{
			Filename: firstNonEmptyString(c.Query("filename"), c.GetHeader("X-Filename")),
			Content:  body,
			Actor:    actor.Name,
		}
		item, err := repo.ImportPurchasableMaterials(c.Request.Context(), input)
		respond(c, item, err)
	})
	api.GET("/material-purchases/monthly-export.csv", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		items, err := repo.MaterialPurchases(c.Request.Context())
		if err != nil {
			respond(c, nil, err)
			return
		}
		month := strings.TrimSpace(c.Query("month"))
		if month == "" {
			month = time.Now().Format("2006-01")
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=lirs-material-purchases-"+month+".csv")
		writeCSVBOM(c)
		writer := csv.NewWriter(c.Writer)
		_ = writer.Write([]string{"申购流水号", "ID号", "序号", "采购项目名称及编号", "项目名称", "品牌", "规格", "单位", "采购价（元）", "备注", "技术要求", "最小规格", "申购人"})
		for _, item := range filterMaterialPurchasesForActor(actor, items) {
			if item.CreatedAt.Format("2006-01") != month {
				continue
			}
			_ = writer.Write([]string{
				item.PurchaseSerialNo,
				item.PurchaseIDNo,
				item.PurchaseSequenceNo,
				item.PurchaseProjectName,
				firstNonEmptyString(item.PurchaseItemName, item.MaterialName),
				item.PurchaseBrand,
				item.PurchaseSpec,
				item.PurchaseUnit,
				fmt.Sprintf("%.2f", item.EstimatedUnitPrice),
				item.PurchaseRemark,
				item.PurchaseTechnicalRequirement,
				item.PurchaseMinSpec,
				item.Requester,
			})
		}
		writer.Flush()
	})
}

func purchasableMaterialImportHeader() []string {
	return []string{"ID号", "序号", "采购项目名称及编号", "项目名称", "品牌", "规格", "单位", "采购价（元）", "备注", "技术要求", "最小规格"}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
