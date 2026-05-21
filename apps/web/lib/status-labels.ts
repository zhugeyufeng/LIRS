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

export function auditActionLabel(action: string | null | undefined) {
  return labelFrom(action, {
    "access_control_settings.update": "更新门禁设置",
    "auth.login": "用户登录",
    "auth.logout_all": "清理全部登录会话",
    "billing.config.create": "创建计费配置",
    "billing.config.update": "更新计费配置",
    "financial_account.save": "保存财务账户",
    "instrument.create": "新增仪器",
    "instrument.delete": "删除仪器",
    "instrument.update": "更新仪器",
    "iot.device.create": "创建设备",
    "iot.device.delete": "删除设备",
    "iot.device.update": "更新设备",
    "assistant.query.delete": "删除 AI 问答",
    "ledger.adjust": "调整财务流水",
    "maintenance.cancel": "取消维护单",
    "maintenance.complete": "完成维护单",
    "maintenance.create": "创建维护单",
    "maintenance.start": "开始维护单",
    "material.approved": "通过资源申领",
    "material.cancel": "取消资源申领",
    "material.create": "新增资源",
    "material.delete": "删除资源",
    "material.outbound": "资源出库",
    "material.rejected": "拒绝资源申领",
    "material.request": "提交资源申领",
    "material.stock_adjust": "调整资源库存",
    "material.update": "更新资源",
    "material_alert.handled": "处理资源预警",
    "material_alert.ignored": "忽略资源预警",
    "material_category.disable": "停用资源目录",
    "material_category.save": "保存资源目录",
    "material_category.update": "更新资源目录",
    "material_damage.approved": "通过损毁记录",
    "material_damage.cancel": "取消损毁记录",
    "material_damage.create": "新增损毁记录",
    "material_damage.process": "处理损毁记录",
    "material_damage.rejected": "拒绝损毁记录",
    "material_purchase.approved": "通过资源申购",
    "material_purchase.cancel": "取消资源申购",
    "material_purchase.create": "登记资源申购",
    "material_purchase.month_confirm": "确认月度申购汇总",
    "material_purchase.order": "资源申购下单",
    "material_purchase.receive": "资源申购入库",
    "material_purchase.rejected": "拒绝资源申购",
    "material_purchase.resubmit": "重新提交资源申购",
    "material_purchase.returned": "退回资源申购",
    "notification.announce": "发布公告",
    "notification.delete": "删除通知",
    "notification.dingtalk_settings": "更新钉钉通知设置",
    "notification.dingtalk_test": "测试钉钉通知",
    "notification.graph_mail_settings": "更新邮件通知设置",
    "notification.graph_mail_test": "测试邮件通知",
    "notification.update": "更新通知",
    "notification.wechat_settings": "更新微信通知设置",
    "organization_unit.create": "新增组织单位",
    "organization_unit.delete": "删除组织单位",
    "organization_unit.update": "更新组织单位",
    "procurement_project.delete": "删除采购项目",
    "procurement_project.save": "保存采购项目",
    "purchasable_material.delete": "删除可采购物资",
    "purchasable_material.import": "导入可采购物资",
    "purchasable_material.save": "保存可采购物资",
    "purchasable_material.update": "更新可采购物资",
    "reservation.approve": "通过预约",
    "reservation.auto_cancel": "自动取消预约",
    "reservation.cancel": "取消预约",
    "reservation.checkin": "预约签到",
    "reservation.checkout": "预约签退",
    "reservation.create": "提交预约",
    "reservation.reject": "拒绝预约",
    "sample.create": "新增样本",
    "sample.update": "更新样本",
    "sample_movement.create": "新增样本流转",
    "site_settings.update": "更新站点设置",
    "space.create": "新增空间",
    "space.update": "更新空间",
    "space_reservation.create": "提交空间预约",
    "tenant.create": "新增机构",
    "tenant.update": "更新机构",
    "training.authorization.create": "新增培训授权",
    "training.authorization.update": "更新培训授权",
    "training.course.create": "新增培训课程",
    "training.course.update": "更新培训课程",
    "training.exam.create": "新增考试记录",
    "training.exam.update": "更新考试记录",
    "training.practical.create": "新增实操考核",
    "training.practical.update": "更新实操考核",
    "training.question.create": "新增题库题目",
    "training.question.update": "更新题库题目",
    "training.rule.create": "新增准入规则",
    "training.rule.update": "更新准入规则",
    "user.create": "新增用户",
    "user.delete": "删除用户",
    "user.dingtalk_bind": "绑定钉钉账号",
    "user.dingtalk_login_bind": "扫码登录绑定",
    "user.dingtalk_unbind": "解绑钉钉账号",
    "user.email_verified": "验证邮箱",
    "user.membership.save": "保存用户机构关系",
    "user.password_change": "修改密码",
    "user.register": "用户注册",
    "user.review": "审核用户",
    "workflow.config.create": "创建流程配置",
    "workflow.config.update": "更新流程配置",
  }, "未知操作");
}

export function auditTargetTypeLabel(targetType: string | null | undefined) {
  return labelFrom(targetType, {
    business_config: "业务配置",
    financial_account: "财务账户",
    assistant_query: "AI 问答记录",
    instrument: "仪器",
    iot_device: "物联网设备",
    ledger_entry: "财务流水",
    maintenance_order: "维护单",
    material: "资源",
    material_category: "资源目录",
    material_damage: "资源损毁记录",
    material_purchase: "资源申购",
    material_purchase_month: "月度申购汇总",
    material_request: "资源申领",
    notification: "通知",
    organization_unit: "组织单位",
    procurement_project: "采购项目",
    purchasable_material: "可采购物资",
    reservation: "预约",
    sample: "样本",
    sample_movement: "样本流转",
    site_setting: "站点设置",
    space: "空间",
    space_reservation: "空间预约",
    tenant: "机构",
    training_authorization: "培训授权",
    training_course: "培训课程",
    training_exam: "考试记录",
    training_practical_assessment: "实操考核",
    training_question: "题库题目",
    training_rule: "准入规则",
    user: "用户",
  }, "未知对象");
}

export function auditValueLabel(value: string | null | undefined) {
  const normalizedValue = normalizeStatus(value);
  if (!normalizedValue) {
    return "无";
  }
  if (normalizedValue.includes("=")) {
    return normalizedValue
      .replace(/\bcreated=/g, "新增 ")
      .replace(/\bupdated=/g, "更新 ");
  }
  if (normalizedValue.includes(":")) {
    const [kind, ...rest] = normalizedValue.split(":");
    const name = rest.join(":");
    const kindLabels: LabelMap = {
      department: "部门",
      group: "团队",
      laboratory: "实验室",
    };
    if (kindLabels[kind]) {
      return `${kindLabels[kind]}：${name || "未命名"}`;
    }
  }
  return normalizedValue
    .split("/")
    .map(auditSegmentLabel)
    .join(" / ");
}

function auditSegmentLabel(value: string) {
  const segment = normalizeStatus(value);
  const exactLabels: LabelMap = {
    "false": "否",
    "true": "是",
    deleted: "已删除",
    password_updated: "密码已更新",
    revoked: "已撤销",
    web: "网页端",
    responsive_check: "响应式检查",
  };
  return exactLabels[segment] ?? commonStatusLabels[segment] ?? segment;
}
