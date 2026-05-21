import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft, AlertTriangle, HardDrive, ShieldCheck } from "lucide-react";
import { AdminShell, requireAdmin } from "@/components/admin-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type AuditEvent } from "@/lib/api";
import { alertLevelLabel } from "@/lib/status-labels";

type Section = "login-logs" | "operation-logs" | "data-audit" | "permission-audit" | "risks" | "backups" | "compliance";

export default async function AdminSecuritySectionPage({
  params,
  searchParams,
}: {
  params: Promise<{ section: string }>;
  searchParams?: Promise<{ q?: string }>;
}) {
  await requireAdmin();
  const { section: rawSection } = await params;
  const section = rawSection as Section;
  const query = ((await searchParams)?.q ?? "").trim().toLowerCase();
  if (!isSection(section)) {
    notFound();
  }

  const [auditEvents, operations] = await Promise.all([api.auditEvents(), api.operations()]);
  const visibleEvents = filterEvents(section, auditEvents, query);

  return (
    <AdminShell active="security" title={sectionTitle(section)} description={sectionDescription(section)}>
      <Link className="mb-5 inline-flex items-center gap-2 text-sm text-slate-600 hover:text-primary" href="/admin/security">
        <ArrowLeft className="h-4 w-4" />
        返回安全审计中心
      </Link>

      {section === "risks" ? <RiskOverview alerts={operations.alerts} /> : null}
      {section === "backups" ? <BackupOverview /> : null}
      {section === "compliance" ? <ComplianceOverview /> : null}

      {section !== "backups" && section !== "compliance" ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ShieldCheck className="h-5 w-5 text-primary" />
              {sectionTitle(section)}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {visibleEvents.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">暂无记录。</p> : null}
            {visibleEvents.map((event) => (
              <AuditEventRow event={event} key={event.id} />
            ))}
          </CardContent>
        </Card>
      ) : null}

      {section !== "backups" && section !== "compliance" ? (
        <div className="mt-6 flex flex-wrap gap-3">
          <Link className="inline-flex h-10 items-center justify-center rounded-md border px-4 text-sm font-medium text-slate-700 hover:bg-slate-50" href="/admin/security/login-logs">
            登录日志
          </Link>
          <Link className="inline-flex h-10 items-center justify-center rounded-md border px-4 text-sm font-medium text-slate-700 hover:bg-slate-50" href="/admin/security/operation-logs">
            操作日志
          </Link>
        </div>
      ) : null}
    </AdminShell>
  );
}

function isSection(value: string): value is Section {
  return ["login-logs", "operation-logs", "data-audit", "permission-audit", "risks", "backups", "compliance"].includes(value);
}

function filterEvents(section: Section, events: AuditEvent[], query: string) {
  let filtered = events;
  if (section === "login-logs") {
    filtered = events.filter((event) => event.action.startsWith("auth."));
  } else if (section === "operation-logs") {
    filtered = events;
  } else if (section === "data-audit") {
    filtered = events.filter((event) => !event.action.startsWith("auth."));
  } else if (section === "permission-audit") {
    filtered = events.filter((event) => event.action.startsWith("user.") || event.action.includes("tenant.") || event.action.includes("organization_unit."));
  }
  return filtered
    .filter((event) => {
      if (!query) {
        return true;
      }
      return [event.actor, event.action, event.targetType, event.targetId, event.oldValue, event.newValue].some((value) => String(value ?? "").toLowerCase().includes(query));
    })
    .slice(0, 50);
}

function sectionTitle(section: Section) {
  const titles: Record<Section, string> = {
    "login-logs": "登录日志",
    "operation-logs": "操作日志",
    "data-audit": "数据审计",
    "permission-audit": "权限审计",
    risks: "异常访问",
    backups: "数据备份",
    compliance: "合规配置",
  };
  return titles[section];
}

function sectionDescription(section: Section) {
  const descriptions: Record<Section, string> = {
    "login-logs": "查看登录、退出和会话相关留痕。",
    "operation-logs": "查看所有关键业务操作留痕。",
    "data-audit": "查看关键数据变更前后内容。",
    "permission-audit": "查看角色、机构、部门和数据域变更。",
    risks: "查看异常预警和风险提示。",
    backups: "查看备份策略和保留说明。",
    compliance: "查看留存、归档和逻辑删除策略。",
  };
  return descriptions[section];
}

function AuditEventRow({ event }: { event: AuditEvent }) {
  return (
    <div className="rounded-lg border p-4 text-sm">
      <div className="flex flex-col justify-between gap-2 sm:flex-row sm:items-start">
        <div className="min-w-0">
          <p className="font-bold text-slate-900">{event.action}</p>
          <p className="mt-1 break-words text-xs text-slate-500">
            {event.actor} / {event.targetType} / {event.targetId}
          </p>
        </div>
        <span className="shrink-0 text-xs text-slate-500">{formatDateTime(event.createdAt)}</span>
      </div>
      <div className="mt-3 grid gap-3 md:grid-cols-2">
        <Detail label="变更前" value={event.oldValue} />
        <Detail label="变更后" value={event.newValue} />
      </div>
    </div>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md bg-slate-50 p-3">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-2 break-words text-sm leading-6 text-slate-700">{value || "无"}</p>
    </div>
  );
}

function RiskOverview({ alerts }: { alerts: { source: string; level: string; body: string }[] }) {
  return (
    <Card className="mb-6">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <AlertTriangle className="h-5 w-5 text-primary" />
          风险预警
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {alerts.length === 0 ? <p className="text-sm text-slate-500">暂无预警。</p> : null}
        {alerts.map((item, index) => (
          <div className="rounded-lg border bg-amber-50 p-3 text-sm text-amber-800" key={`${item.source}-${index}`}>
            <p className="font-medium">{item.source}</p>
            <p className="mt-1 break-words text-xs leading-5">{item.body}</p>
            <p className="mt-1 text-[10px] opacity-70">{alertLevelLabel(item.level)}</p>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

function BackupOverview() {
  return (
    <Card className="mb-6">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <HardDrive className="h-5 w-5 text-primary" />
          数据备份
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 text-sm leading-6 text-slate-700">
        <p>当前系统使用 Docker Compose 侧车执行 PostgreSQL 备份，保留最近 14 天的每日备份文件。</p>
        <p>备份策略和恢复流程建议仅由系统超级管理员维护，避免误删和误覆盖。</p>
      </CardContent>
    </Card>
  );
}

function ComplianceOverview() {
  return (
    <Card className="mb-6">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ShieldCheck className="h-5 w-5 text-primary" />
          合规配置
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 text-sm leading-6 text-slate-700">
        <p>关键业务记录采用逻辑删除或作废方式保留留痕，预约、审批、库存、财务和审计数据默认不做物理删除。</p>
        <p>导出和高风险操作建议纳入审计，并根据机构要求设置数据保留周期和归档规则。</p>
      </CardContent>
    </Card>
  );
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Shanghai",
  });
}
