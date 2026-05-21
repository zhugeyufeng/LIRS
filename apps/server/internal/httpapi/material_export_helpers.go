package httpapi

import (
	"encoding/csv"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"lirs/apps/server/internal/store"
)

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
