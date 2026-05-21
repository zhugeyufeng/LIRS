type LabelMap = Record<string, string>;

const commonStatusLabels: LabelMap = {
  active: "启用",
  disabled: "停用",
  pending: "待处理",
  pending_approval: "待审核",
  approved: "已通过",
  rejected: "已拒绝",
  in_use: "使用中",
  completed: "已完成",
  cancelled: "已取消",
  draft: "草稿",
  archived: "已归档",
  submitted: "已提交",
  graded: "已评分",
  available: "可用",
  busy: "占用",
  maintenance: "维护中",
  expired: "已过期",
  revoked: "已撤销",
  stored: "已入库",
  testing: "检测中",
  checked_out: "外借",
  disposed: "已处置",
  assigned: "已分配",
  running: "进行中",
  signed: "已签名",
  online: "在线",
  offline: "离线",
  warning: "预警",
  normal: "普通",
  high: "高",
  urgent: "紧急",
  pass: "通过",
  fail: "未通过",
  failed: "未通过",
  returned: "已退回",
  ordered: "已下单",
  received: "已到货",
  registered: "已登记",
  outbound: "已出库",
  reported: "已上报",
  in_progress: "处理中",
  processed: "已处理",
  depleted: "已用尽",
  reserved: "已预留",
  used: "已领用",
  debit: "预约扣费",
  adjustment: "调整",
  account_init: "账户初始化",
  info: "普通",
  success: "成功",
};

function normalizeStatus(value: string | null | undefined) {
  return String(value ?? "").trim();
}

function labelFrom(value: string | null | undefined, labels: LabelMap, unknownLabel = "未知状态") {
  const status = normalizeStatus(value);
  if (!status) {
    return "未设置";
  }
  return labels[status] ?? commonStatusLabels[status] ?? unknownLabel;
}

export function statusLabel(status: string | null | undefined) {
  return labelFrom(status, commonStatusLabels);
}

export function enablementStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    active: "启用",
    disabled: "停用",
    draft: "草稿",
    archived: "已归档",
  });
}

export function trainingCourseStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    draft: "草稿",
    active: "启用",
    archived: "已归档",
  });
}

export function trainingQuestionStatusLabel(status: string | null | undefined) {
  return trainingCourseStatusLabel(status);
}

export function trainingQuestionTypeLabel(value: string | null | undefined) {
  return labelFrom(value, {
    single: "单选",
    multiple: "多选",
    judge: "判断",
    short: "简答",
  }, "未知题型");
}

export function trainingRuleStatusLabel(status: string | null | undefined) {
  return enablementStatusLabel(status);
}

export function trainingExamStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    draft: "草稿",
    submitted: "已提交",
    graded: "已评分",
    archived: "已归档",
  });
}

export function trainingPracticalResultLabel(result: string | null | undefined) {
  return labelFrom(result, {
    pending: "待确认",
    pass: "通过",
    fail: "未通过",
  }, "未知结果");
}

export function trainingAuthorizationStatusLabel(status: string | null | undefined, activeLabel = "已授权") {
  return labelFrom(status, {
    pending: "待审核",
    active: activeLabel,
    expired: "已过期",
    revoked: "已撤销",
  });
}

export function trainingDeliveryModeLabel(value: string | null | undefined) {
  return labelFrom(value, {
    online: "线上",
    offline: "线下",
    blended: "混合",
  }, "未知方式");
}

export function reservationStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    pending: "待审批",
    approved: "已通过",
    rejected: "已拒绝",
    in_use: "使用中",
    completed: "已完成",
    cancelled: "已取消",
  });
}

export function instrumentStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    available: "可用",
    busy: "繁忙",
    maintenance: "维护中",
    disabled: "停用",
  });
}

export function slotStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    available: "可预约",
    occupied: "已占用",
    maintenance: "维护中",
    disabled: "已停用",
    unavailable: "不可预约",
  });
}

export function spaceStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    available: "可用",
    busy: "占用",
    maintenance: "维护中",
    disabled: "停用",
  });
}

export function sampleStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    stored: "已入库",
    testing: "检测中",
    checked_out: "外借",
    archived: "已归档",
    disposed: "已处置",
  });
}

export function hazardLevelLabel(level: string | null | undefined) {
  return labelFrom(level, {
    normal: "普通",
    warning: "警示",
    danger: "高危",
  }, "未知等级");
}

export function alertLevelLabel(level: string | null | undefined) {
  return labelFrom(level, {
    info: "普通",
    normal: "普通",
    warning: "预警",
    high: "高风险",
    danger: "高危",
    urgent: "紧急",
    critical: "严重",
  }, "未知等级");
}

export function limsTaskStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    pending: "待分配",
    assigned: "已分配",
    running: "进行中",
    completed: "已完成",
    cancelled: "已取消",
  });
}

export function elnRecordStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    draft: "草稿",
    submitted: "已提交",
    signed: "已签名",
    archived: "已归档",
  });
}

export function iotDeviceStatusLabel(status: string | null | undefined, online = false) {
  const normalizedStatus = normalizeStatus(status);
  if (online && normalizedStatus !== "disabled") {
    return "在线";
  }
  return labelFrom(normalizedStatus, {
    online: "在线",
    offline: "离线",
    warning: "预警",
    disabled: "停用",
  });
}

export function spaceReservationStatusLabel(status: string | null | undefined) {
  return reservationStatusLabel(status);
}

export function userStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    pending_approval: "待审核",
    active: "启用",
    disabled: "停用",
  });
}

export function materialRequestStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    pending: "待审批",
    approved: "已通过",
    rejected: "已拒绝",
    outbound: "已出库",
    cancelled: "已取消",
  });
}

export function materialPurchaseStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    registered: "已登记",
    approved: "已通过",
    rejected: "已拒绝",
    returned: "退回修改",
    ordered: "已下单",
    received: "已入库",
    cancelled: "已取消",
  });
}

export function materialStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    normal: "正常",
    near_expiry: "临期",
    low: "低库存",
    expired: "过期",
    open_expired: "开封超期",
    freeze_thaw_exceeded: "冻融超限",
    damaged: "损毁",
    disabled: "停用",
  });
}

export function materialProductTypeLabel(productType: string | null | undefined) {
  return labelFrom(productType, {
    consumable: "耗材",
    reagent: "试剂",
    standard: "标准品/标准物质",
  }, "未知资源类型");
}

export function materialDamageStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    pending: "待审核",
    approved: "已通过",
    rejected: "已拒绝",
    processed: "已处理",
    cancelled: "已取消",
  });
}

export function materialBatchStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    active: "可用",
    depleted: "已用尽",
    disabled: "停用",
  });
}

export function materialUnitStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    available: "可领用",
    reserved: "已预留",
    used: "已领用",
    damaged: "已损毁",
    disabled: "停用",
  });
}

export function maintenanceStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    reported: "已上报",
    assigned: "已派工",
    in_progress: "处理中",
    completed: "已完成",
    cancelled: "已取消",
  });
}

export function maintenanceTypeLabel(type: string | null | undefined) {
  return labelFrom(type, {
    routine: "例行维护",
    fault: "故障维护",
    emergency: "紧急维护",
  }, "未知类型");
}

export function priorityLabel(priority: string | null | undefined) {
  return labelFrom(priority, {
    normal: "普通",
    high: "高优先级",
    urgent: "紧急",
  }, "未知优先级");
}

export function workflowStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    pending: "待处理",
    approved: "已通过",
    rejected: "已拒绝",
    registered: "已登记",
    returned: "退回修改",
    ordered: "采购中",
    received: "已到货",
    outbound: "已出库",
    in_use: "使用中",
    completed: "已完成",
    cancelled: "已取消",
  });
}

export function notificationLevelLabel(level: string | null | undefined) {
  return labelFrom(level, {
    info: "普通",
    warning: "提醒",
    success: "成功",
  }, "未知级别");
}

export function financeEntryTypeLabel(type: string | null | undefined) {
  return labelFrom(type, {
    debit: "预约扣费",
    adjustment: "调整",
    account_init: "账户初始化",
  }, "未知类型");
}

export function materialFeeStatusLabel(status: string | null | undefined) {
  return labelFrom(status, {
    pending: "待审批",
    approved: "已通过",
    rejected: "已拒绝",
    outbound: "已出库",
    registered: "已登记",
    returned: "退回修改",
    ordered: "采购中",
    received: "已到货",
    cancelled: "已取消",
  });
}
