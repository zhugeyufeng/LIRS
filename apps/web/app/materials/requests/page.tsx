import { Search } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { MaterialRequestActions } from "@/components/material-request-form";
import { MaterialsNav } from "@/components/materials-nav";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, MaterialRequest } from "@/lib/api";
import { isMaterialAdminRole } from "@/lib/permissions";

export default async function MaterialRequestsPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string; status?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const [requests, currentUser] = await Promise.all([api.materialRequests(), api.me()]);
  const isAdmin = isMaterialAdminRole(currentUser.role);
  const canReview = isAdmin || currentUser.role === "group_leader";
  const visibleRequests = requests.filter((item) => {
    const matchesSearch =
      query === "" ||
      [item.materialName, item.requester, item.groupName, item.purpose, item.batchNo ?? "", item.unitCode ?? ""].some((value) => value.toLowerCase().includes(query));
    const matchesStatus = !params.status || item.status === params.status;
    return matchesSearch && matchesStatus;
  });

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">申领管理</h1>
        <p className="mt-1 text-sm text-muted-foreground">展示资源申领记录、审批状态和出库状态；新申领请在标准品/标准物质、试剂或耗材页面选择具体资源。</p>
      </div>

      <MaterialsNav active="requests" />

      <div className="grid gap-6">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle>申领单</CardTitle>
          </CardHeader>
          <CardContent>
            <form action="/materials/requests" className="mb-4 grid gap-3 md:grid-cols-[minmax(0,1fr)_180px_auto]">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <input className="h-10 w-full rounded-md border bg-white pl-10 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索产品、申请人、编号、批次、用途" />
              </div>
              <select className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={params.status ?? ""} name="status">
                <option value="">全部状态</option>
                <option value="pending">待审批</option>
                <option value="approved">已通过</option>
                <option value="rejected">已拒绝</option>
                <option value="outbound">已出库</option>
                <option value="cancelled">已取消</option>
              </select>
              <button className="inline-flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white md:w-auto" type="submit">
                筛选
              </button>
            </form>

            <div className="grid gap-3 xl:hidden">
              {visibleRequests.map((item) => (
                <MaterialRequestCard currentUserId={currentUser.id} isAdmin={isAdmin} item={item} key={item.id} canReview={canReview} canOutbound={isAdmin} />
              ))}
            </div>

            <div className="hidden overflow-x-auto rounded-lg border xl:block">
              <table className="w-full table-fixed text-left text-sm">
                <thead className="bg-slate-50 text-slate-500">
                  <tr>
                    <th className="p-3">产品</th>
                    <th className="p-3">申请人</th>
                    <th className="p-3">用途</th>
                    <th className="w-28 p-3">状态</th>
                    <th className="w-[22rem] p-3">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {visibleRequests.map((item) => (
                    <tr key={item.id}>
                      <td className="break-words p-3 align-top">
                        <p className="font-bold">{item.materialName}</p>
                        <p className="mt-1 text-xs text-slate-500">数量 {item.quantity}</p>
                        {item.unitCode ? <p className="mt-1 text-xs text-slate-500">编号 {item.unitCode}</p> : null}
                        {item.batchNo ? <p className="mt-1 text-xs text-slate-500">批次 {item.batchNo}</p> : null}
                      </td>
                      <td className="break-words p-3 align-top">
                        <p className="font-medium">{item.requester}</p>
                        <p className="mt-1 text-xs text-slate-500">{item.groupName}</p>
                      </td>
                      <td className="break-words p-3 align-top">{item.purpose}</td>
                      <td className="p-3 align-top">
                        <span className="rounded bg-slate-100 px-2 py-1 text-xs font-bold">{requestStatusLabel(item.status)}</span>
                      </td>
                      <td className="p-3 align-top">
                        <MaterialRequestActions
                          canCancel={isAdmin || item.requesterId === currentUser.id}
                          canOutbound={isAdmin}
                          canReview={canReview}
                          id={item.id}
                          status={item.status}
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {visibleRequests.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">当前筛选下暂无申领单。</p> : null}
          </CardContent>
        </Card>

      </div>
    </AppShell>
  );
}

function MaterialRequestCard({
  item,
  canReview,
  canOutbound,
  currentUserId,
  isAdmin,
}: {
  item: MaterialRequest;
  canReview: boolean;
  canOutbound: boolean;
  currentUserId: string;
  isAdmin: boolean;
}) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
        <div className="min-w-0">
          <p className="break-words font-bold text-slate-900">{item.materialName} x{item.quantity}</p>
          {item.unitCode ? <p className="mt-1 break-words text-xs text-slate-500">编号 {item.unitCode}</p> : null}
          {item.batchNo ? <p className="mt-1 break-words text-xs text-slate-500">批次 {item.batchNo}</p> : null}
          <p className="mt-1 break-words text-sm text-slate-500">{item.requester} / {item.groupName}</p>
        </div>
        <span className="w-fit shrink-0 rounded bg-slate-100 px-2 py-1 text-xs font-bold">{requestStatusLabel(item.status)}</span>
      </div>
      <p className="mt-3 break-words text-sm text-slate-600">{item.purpose}</p>
      <div className="mt-3">
        <MaterialRequestActions canCancel={isAdmin || item.requesterId === currentUserId} canOutbound={canOutbound} canReview={canReview} id={item.id} status={item.status} />
      </div>
    </div>
  );
}

function requestStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: "待审批",
    approved: "已通过",
    rejected: "已拒绝",
    outbound: "已出库",
    cancelled: "已取消",
  };
  return labels[status] ?? status;
}
