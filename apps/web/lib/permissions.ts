export type AdminSection =
  | "overview"
  | "users"
  | "instruments"
  | "materials"
  | "approvals"
  | "training"
  | "trainingQuestions"
  | "trainingPractical"
  | "trainingAuthorizations"
  | "trainingRules"
  | "notifications"
  | "maintenance"
  | "finance"
  | "workflows"
  | "operations"
  | "analytics"
  | "security"
  | "settings";

export function isTenantAdminRole(role: string | undefined) {
  return role === "tenant_admin" || role === "lab_admin" || role === "super_admin";
}

export function isMaterialAdminRole(role: string | undefined) {
  return role === "material_admin" || isTenantAdminRole(role);
}

export function isFinanceAdminRole(role: string | undefined) {
  return role === "finance_admin" || isTenantAdminRole(role);
}

export function isAnyAdminRole(role: string | undefined) {
  return role === "material_admin" || role === "finance_admin" || isTenantAdminRole(role);
}

export function canAccessAdminSection(role: string | undefined, section: AdminSection, financeEnabled = false) {
  if (section === "approvals" && role === "group_leader") {
    return true;
  }
  if (section === "training" || section === "trainingQuestions" || section === "trainingPractical" || section === "trainingAuthorizations" || section === "trainingRules") {
    return isMaterialAdminRole(role) || isTenantAdminRole(role);
  }
  if (!isAnyAdminRole(role)) {
    return false;
  }
  if (section === "overview") {
    return true;
  }
  if (section === "materials") {
    return isMaterialAdminRole(role);
  }
  if (section === "finance") {
    return isFinanceAdminRole(role) && (role === "super_admin" || financeEnabled);
  }
  if (section === "workflows") {
    return isTenantAdminRole(role);
  }
  if (isTenantAdminRole(role)) {
    return true;
  }
  return false;
}

export function roleLabel(role: string) {
  const labels: Record<string, string> = {
    unassigned: "待分配",
    student: "学生",
    teacher: "教师",
    researcher: "研究员",
    group_leader: "负责人",
    material_admin: "试剂管理员",
    finance_admin: "财务管理员",
    tenant_admin: "机构管理员",
    lab_admin: "实验室管理员",
    super_admin: "系统超级管理员",
  };
  return labels[role] ?? role;
}
