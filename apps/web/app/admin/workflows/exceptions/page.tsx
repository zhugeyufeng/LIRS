import { AlertTriangle } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { WorkflowTabs } from "@/components/workflow-tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

const timeoutMs = 24 * 60 * 60 * 1000;

export default async function WorkflowExceptionsPage() {
  await requireAdminSection("workflows");
  const [reservations, materialRequests, materialPurchases] = await Promise.all([
    api.reservations().catch(() => []),
    api.materialRequests().catch(() => []),
    api.materialPurchases().catch(() => []),
  ]);
  const now = Date.now();
  const rows = [
    ...reservations
      .filter((item) => item.status === "pending" && now - new Date(item.startTime).getTime() > timeoutMs)
      .map((item) => ({ id: item.id, type: "仪器预约", title: item.instrumentName, requester: item.userName, createdAt: item.startTime })),
    ...materialRequests
      .filter((item) => item.status === "pending" && now - new Date(item.createdAt).getTime() > timeoutMs)
      .map((item) => ({ id: item.id, type: "资源申领", title: item.materialName, requester: item.requester, createdAt: item.createdAt })),
    ...materialPurchases
      .filter((item) => item.status === "pending" && now - new Date(item.createdAt).getTime() > timeoutMs)
      .map((item) => ({ id: item.id, type: "资源申购", title: item.materialName, requester: item.requester, createdAt: item.createdAt })),
  ];

  return (
    <AdminShell active="workflows" title="超时审批" description="查看超过 24 小时仍待处理的审批申请，便于转交或管理员干预。">
      <WorkflowTabs active="exceptions" />
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-amber-600" aria-hidden="true" />
            超时待处理申请
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {rows.map((row) => (
            <article className="rounded-lg border bg-white p-4" key={`${row.type}-${row.id}`}>
              <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                <div className="min-w-0">
                  <p className="text-xs font-bold text-amber-700">{row.type}</p>
                  <h2 className="mt-1 break-words font-bold text-slate-900">{row.title}</h2>
                  <p className="mt-1 break-words text-xs text-slate-500">{row.id}</p>
                </div>
                <span className="w-fit rounded-full bg-amber-50 px-2 py-1 text-xs font-bold text-amber-700">超时待处理</span>
              </div>
              <p className="mt-3 text-sm text-slate-600">
                {row.requester} / 提交或预约时间：{formatDateTime(row.createdAt)}
              </p>
            </article>
          ))}
          {rows.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无超时审批。</p> : null}
        </CardContent>
      </Card>
    </AdminShell>
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
