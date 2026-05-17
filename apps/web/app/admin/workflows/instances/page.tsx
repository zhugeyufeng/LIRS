import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { WorkflowTabs } from "@/components/workflow-tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function WorkflowInstancesPage() {
  await requireAdminSection("workflows");
  const [reservations, materialRequests, materialPurchases] = await Promise.all([
    api.reservations().catch(() => []),
    api.materialRequests().catch(() => []),
    api.materialPurchases().catch(() => []),
  ]);
  const rows = [
    ...reservations.map((item) => ({
      id: item.id,
      type: "仪器预约",
      title: item.instrumentName,
      requester: item.userName,
      status: item.status,
      createdAt: item.startTime,
    })),
    ...materialRequests.map((item) => ({
      id: item.id,
      type: "资源申领",
      title: item.materialName,
      requester: item.requester,
      status: item.status,
      createdAt: item.createdAt,
    })),
    ...materialPurchases.map((item) => ({
      id: item.id,
      type: "资源申购",
      title: item.materialName,
      requester: item.requester,
      status: item.status,
      createdAt: item.createdAt,
    })),
  ].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());

  return (
    <AdminShell active="workflows" title="流程实例" description="统一查看预约、申领和申购等业务的流程状态，记录不在前端删除。">
      <WorkflowTabs active="instances" />
      <Card>
        <CardHeader>
          <CardTitle>流程实例列表</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 xl:hidden">
            {rows.map((row) => (
              <InstanceCard key={`${row.type}-${row.id}`} row={row} />
            ))}
          </div>
          <div className="hidden overflow-x-auto xl:block">
            <table className="w-full text-left text-sm">
              <thead className="border-b text-slate-500">
                <tr>
                  <th className="py-3 pr-4">类型</th>
                  <th className="py-3 pr-4">业务对象</th>
                  <th className="py-3 pr-4">申请人</th>
                  <th className="py-3 pr-4">状态</th>
                  <th className="py-3">时间</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {rows.map((row) => (
                  <tr key={`${row.type}-${row.id}`}>
                    <td className="py-3 pr-4">{row.type}</td>
                    <td className="py-3 pr-4">
                      <p className="font-medium">{row.title}</p>
                      <p className="text-xs text-slate-500">{row.id}</p>
                    </td>
                    <td className="py-3 pr-4">{row.requester}</td>
                    <td className="py-3 pr-4">{statusLabel(row.status)}</td>
                    <td className="py-3 text-xs text-slate-500">{formatDateTime(row.createdAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {rows.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无流程实例。</p> : null}
        </CardContent>
      </Card>
    </AdminShell>
  );
}

function InstanceCard({ row }: { row: { id: string; type: string; title: string; requester: string; status: string; createdAt: string } }) {
  return (
    <article className="rounded-lg border bg-white p-4 text-sm">
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
        <div className="min-w-0">
          <p className="text-xs font-bold text-primary">{row.type}</p>
          <h2 className="mt-1 break-words font-bold text-slate-900">{row.title}</h2>
          <p className="mt-1 break-words text-xs text-slate-500">{row.id}</p>
        </div>
        <span className="w-fit rounded-full bg-slate-100 px-2 py-1 text-xs font-bold text-slate-600">{statusLabel(row.status)}</span>
      </div>
      <p className="mt-3 text-slate-600">{row.requester} / {formatDateTime(row.createdAt)}</p>
    </article>
  );
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待处理",
    approved: "已通过",
    rejected: "已驳回",
    ordered: "采购中",
    received: "已到货",
    outbound: "已出库",
    in_use: "使用中",
    completed: "已完成",
    cancelled: "已取消",
  };
  return labels[status] ?? status;
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
