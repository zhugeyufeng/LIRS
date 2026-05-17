import Link from "next/link";
import { ShieldCheck, FileClock, FileSearch, FolderLock, HardDrive, AlertTriangle, LogIn, type LucideIcon } from "lucide-react";
import { AdminShell, requireAdmin } from "@/components/admin-shell";
import { api } from "@/lib/api";

export default async function AdminSecurityPage() {
  await requireAdmin();
  const [auditEvents, operations] = await Promise.all([api.auditEvents(), api.operations()]);
  const loginCount = auditEvents.filter((event) => event.action.startsWith("auth.")).length;
  const permissionCount = auditEvents.filter((event) => event.action.startsWith("user.") || event.action.includes("tenant.") || event.action.includes("organization_unit.")).length;
  const dataCount = auditEvents.filter((event) => !event.action.startsWith("auth.")).length;

  return (
    <AdminShell active="security" title="安全审计与合规中心" description="集中查看登录、操作、数据、权限、异常访问和备份合规相关信息。">
      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="登录相关" value={loginCount} />
        <Metric label="权限变更" value={permissionCount} />
        <Metric label="数据审计" value={dataCount} />
        <Metric label="风险预警" value={operations.alerts.length} />
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <SecurityCard href="/admin/security/login-logs" icon={LogIn} title="登录日志" description="查看登录、退出和会话清理记录。" />
        <SecurityCard href="/admin/security/operation-logs" icon={FileClock} title="操作日志" description="查看所有关键业务操作留痕。" />
        <SecurityCard href="/admin/security/data-audit" icon={FileSearch} title="数据审计" description="查看关键数据变更前后内容。" />
        <SecurityCard href="/admin/security/permission-audit" icon={FolderLock} title="权限审计" description="查看角色、机构、部门和数据域变更。" />
        <SecurityCard href="/admin/security/risks" icon={AlertTriangle} title="异常访问" description="查看异常预警、风险提示和高频问题。" />
        <SecurityCard href="/admin/security/backups" icon={HardDrive} title="数据备份" description="查看备份策略与保留说明。" />
        <SecurityCard href="/admin/security/compliance" icon={ShieldCheck} title="合规配置" description="查看留存、归档和逻辑删除策略。" />
      </div>

      <div className="mt-6 rounded-lg border bg-white p-4 text-sm leading-6 text-slate-600">
        安全审计记录通过数据库审计表写入，备份由 Docker Compose sidecar 定时执行，保留周期和合规策略在这里集中展示。
      </div>
    </AdminShell>
  );
}

function SecurityCard({ description, href, icon: Icon, title }: { description: string; href: string; icon: LucideIcon; title: string }) {
  return (
    <Link className="rounded-lg border bg-white p-4 transition-colors hover:border-primary/40 hover:bg-primary/5" href={href}>
      <div className="flex items-center gap-3">
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
          <Icon className="h-5 w-5" aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <h2 className="font-bold text-slate-900">{title}</h2>
          <p className="mt-1 text-xs text-slate-500">{description}</p>
        </div>
      </div>
    </Link>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  );
}
