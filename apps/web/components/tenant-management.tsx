"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { Building2, Pencil, Plus, Save } from "lucide-react";
import { browserPatch, browserPost, Tenant, TenantPayload, User } from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function TenantManagement({ currentUser, tenants }: { currentUser: User; tenants: Tenant[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState("");
  const isSuperAdmin = currentUser.role === "super_admin";

  async function saveTenant(event: FormEvent<HTMLFormElement>, tenant?: Tenant, close?: () => void) {
    event.preventDefault();
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: TenantPayload = {
      name: String(form.get("name") ?? ""),
      financeEnabled: form.get("financeEnabled") === "on",
      status: String(form.get("status") ?? "active"),
    };
    const key = tenant?.id ?? "new";
    setPending(key);
    setMessage("");
    try {
      if (tenant) {
        await browserPatch<Tenant>(`/api/tenants/${tenant.id}`, payload);
        setMessage("单位/机构信息已更新。");
      } else {
        await browserPost<Tenant>("/api/tenants", payload);
        formElement.reset();
        setMessage("单位/机构已创建。");
      }
      close?.();
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending("");
    }
  }

  return (
    <div className="space-y-5">
      {isSuperAdmin ? (
        <div className="flex justify-end">
          <AdminDialog
            description="新增单位/机构后，系统会自动生成唯一机构编码，用户注册可通过机构 ID 或机构编码归属到该单位/机构。"
            title="新增单位/机构"
            trigger={
              <Button>
                <Plus className="h-4 w-4" />
                新增单位/机构
              </Button>
            }
          >
            {(close) => (
              <form className="space-y-4" onSubmit={(event) => saveTenant(event, undefined, close)}>
                <TenantFields />
                <div className="flex justify-end">
                  <Button disabled={pending === "new"} type="submit">
                    <Save className="h-4 w-4" />
                    {pending === "new" ? "保存中" : "创建单位/机构"}
                  </Button>
                </div>
              </form>
            )}
          </AdminDialog>
        </div>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-2">
        {tenants.map((tenant) => (
          <div className="rounded-lg border bg-white p-4" key={tenant.id}>
            <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <Building2 className="h-5 w-5 shrink-0 text-primary" aria-hidden="true" />
                  <h2 className="break-words font-bold text-slate-900">{tenant.name}</h2>
                </div>
                <p className="mt-2 break-all text-xs text-slate-500">单位/机构编码：{tenant.code}</p>
                <p className="mt-1 break-all text-xs text-slate-500">单位/机构 ID：{tenant.id}</p>
                <p className="mt-2 text-xs text-slate-500">财务模块：{tenant.financeEnabled ? "启用" : "停用"}</p>
              </div>
              <div className="flex shrink-0 flex-col items-start gap-3 sm:items-end">
                <span className={tenant.status === "active" ? "w-fit rounded bg-emerald-50 px-2 py-1 text-xs font-bold text-emerald-700" : "w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-600"}>
                  {tenant.status === "active" ? "启用" : "停用"}
                </span>
                <AdminDialog
                  description="修改后会立即影响该单位/机构下的注册、财务模块展示和机构状态。"
                  title={`修改单位/机构：${tenant.name}`}
                  trigger={
                    <Button size="sm" variant="outline">
                      <Pencil className="h-4 w-4" />
                      修改
                    </Button>
                  }
                >
                  {(close) => (
                    <form className="space-y-4" onSubmit={(event) => saveTenant(event, tenant, close)}>
                      <TenantFields tenant={tenant} lockStatus={!isSuperAdmin} />
                      <div className="flex justify-end">
                        <Button disabled={pending === tenant.id} type="submit">
                          <Save className="h-4 w-4" />
                          {pending === tenant.id ? "保存中" : "保存设置"}
                        </Button>
                      </div>
                    </form>
                  )}
                </AdminDialog>
              </div>
            </div>
          </div>
        ))}
      </div>
      {tenants.length === 0 ? <p className="rounded-lg border bg-white p-4 text-sm text-slate-500">暂无机构。</p> : null}
      {message ? <p className="rounded-md bg-slate-100 p-3 text-sm text-slate-700">{message}</p> : null}
    </div>
  );
}

function TenantFields({ tenant, lockStatus = false }: { tenant?: Tenant; lockStatus?: boolean }) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <label className="block space-y-2">
        <span className="text-sm font-medium">单位名称</span>
        <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={tenant?.name ?? ""} name="name" placeholder="填写单位名称" required />
        {tenant ? <FieldHint value={`当前：${tenant.name}`} /> : null}
      </label>
      <label className="block space-y-2">
        <span className="text-sm font-medium">单位/机构编码</span>
        <div className="flex min-h-10 items-center rounded-md border bg-slate-50 px-3 text-sm text-slate-700">
          {tenant?.code ?? "创建后自动生成"}
        </div>
        <FieldHint value={tenant ? "单位/机构编码由系统生成，编辑单位名称时不会变更。" : "提交创建后自动生成唯一编码。"} />
      </label>
      <label className="block space-y-2">
        <span className="text-sm font-medium">状态</span>
        <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={tenant?.status ?? "active"} disabled={lockStatus} name="status">
          <option value="active">启用</option>
          <option value="disabled">停用</option>
        </select>
        {lockStatus ? <input name="status" type="hidden" value={tenant?.status ?? "active"} /> : null}
        {tenant ? <FieldHint value={`当前：${tenant.status === "active" ? "启用" : "停用"}`} /> : null}
      </label>
      <label className="block self-end space-y-2">
        <span className="flex h-10 items-center gap-3 rounded-md border bg-white px-3 text-sm">
          <input className="h-4 w-4" defaultChecked={tenant?.financeEnabled ?? true} name="financeEnabled" type="checkbox" />
          启用财务模块
        </span>
        {tenant ? <FieldHint value={`当前：${tenant.financeEnabled ? "启用" : "停用"}`} /> : null}
      </label>
    </div>
  );
}

function FieldHint({ value }: { value: string }) {
  return <span className="block break-words text-xs text-slate-500">{value}</span>;
}
