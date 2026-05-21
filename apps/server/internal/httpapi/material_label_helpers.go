package httpapi

import (
	"strings"

	"lirs/apps/server/internal/store"
)

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
