"use client";

import { FormEvent, startTransition, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { browserDelete, browserPatch, browserPost, OrganizationUnit, OrganizationUnitPayload, Tenant, User } from "@/lib/api";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function OrganizationUnitManager({
  currentUser,
  tenants,
  selectedTenantId,
  selectedTenantName,
  departments,
  groups,
}: {
  currentUser: User;
  tenants: Tenant[];
  selectedTenantId: string;
  selectedTenantName: string;
  departments: OrganizationUnit[];
  groups: OrganizationUnit[];
}) {
  const isSuperAdmin = currentUser.role === "super_admin";
  return (
    <div className="space-y-5">
      {isSuperAdmin ? (
        <TenantSelector tenants={tenants} selectedTenantId={selectedTenantId} selectedTenantName={selectedTenantName} />
      ) : (
        <div className="rounded-lg border bg-slate-50/60 p-4">
          <p className="text-sm font-medium text-slate-900">当前机构：{selectedTenantName}</p>
          <p className="mt-1 text-xs text-slate-500">以下基础数据仅作用于当前机构。</p>
        </div>
      )}
      <div className="grid gap-6 xl:grid-cols-2">
        <UnitSection departments={departments} kind="department" selectedTenantId={selectedTenantId} title="用户部门" units={departments} />
        <UnitSection departments={departments} kind="group" selectedTenantId={selectedTenantId} title="部门二级团队" units={groups} />
      </div>
    </div>
  );
}

function TenantSelector({
  tenants,
  selectedTenantId,
  selectedTenantName,
}: {
  tenants: Tenant[];
  selectedTenantId: string;
  selectedTenantName: string;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  function changeTenant(value: string) {
    const next = new URLSearchParams(searchParams.toString());
    next.set("tenantId", value);
    router.push(`${pathname}?${next.toString()}`);
  }

  return (
    <div className="rounded-lg border bg-slate-50/60 p-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <label className="block min-w-0 flex-1 space-y-2">
          <span className="text-sm font-medium text-slate-900">选择机构</span>
          <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" onChange={(event) => changeTenant(event.currentTarget.value)} value={selectedTenantId}>
            {tenants.map((tenant) => (
              <option key={tenant.id} value={tenant.id}>
                {tenant.name}
              </option>
            ))}
          </select>
        </label>
        <p className="text-xs text-slate-500 md:max-w-sm md:text-right">当前机构：{selectedTenantName}。切换后，下方部门和团队信息都会读取该机构自己的配置。</p>
      </div>
    </div>
  );
}

function UnitSection({
  departments,
  kind,
  selectedTenantId,
  title,
  units,
}: {
  departments: OrganizationUnit[];
  kind: "department" | "group";
  selectedTenantId: string;
  title: string;
  units: OrganizationUnit[];
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: OrganizationUnitPayload = {
      kind,
      name: String(form.get("name") ?? ""),
      parentName: kind === "group" ? String(form.get("parentName") ?? "") : "",
    };
    try {
      await browserPost<OrganizationUnit>(organizationUnitPath("/api/organization-units", selectedTenantId), payload);
      setMessage(`${title} 已新增。`);
      formElement.reset();
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "新增失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-4 rounded-lg border bg-slate-50/60 p-4">
      <div>
        <p className="text-sm font-bold">{title}</p>
        <p className="mt-1 text-xs text-slate-500">
          {kind === "department"
            ? "部门用于用户属性和仪器一级分类；删除前相关用户、仪器和二级团队必须为 0。"
            : "团队是部门下的二级归属；仪器可直接归属部门，也可归属到该部门下的团队。"}
        </p>
      </div>
      <AdminDialog
        description={kind === "department" ? "新增部门后，可在人员、仪器和注册等页面中选择使用。" : "新增团队时必须选择所属部门，仪器编辑时会按部门过滤可选团队。"}
        title={`新增${title}`}
        trigger={
          <Button className="w-full sm:w-fit">
            <Plus className="h-4 w-4" aria-hidden="true" />
            新增{title}
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <label className="block space-y-2">
              <span className="text-sm font-medium">{title}名称</span>
              <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" name="name" placeholder={`填写${title}名称`} required />
            </label>
            {kind === "group" ? <DepartmentSelect departments={departments} /> : null}
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                {pending ? "添加中..." : "新增"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      <div className="space-y-2">
        {units.length === 0 ? <p className="text-sm text-slate-500">暂无数据。</p> : null}
        {units.map((unit) => (
          <UnitRow departments={departments} key={unit.id} selectedTenantId={selectedTenantId} unit={unit} kind={kind} />
        ))}
      </div>
      {message ? <p className="text-sm text-slate-600">{message}</p> : null}
    </div>
  );
}

function UnitRow({
  departments,
  selectedTenantId,
  unit,
  kind,
}: {
  departments: OrganizationUnit[];
  selectedTenantId: string;
  unit: OrganizationUnit;
  kind: "department" | "group";
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const form = new FormData(event.currentTarget);
    const payload: OrganizationUnitPayload = {
      kind,
      name: String(form.get("name") ?? ""),
      parentName: kind === "group" ? String(form.get("parentName") ?? unit.parentName ?? "") : "",
    };
    try {
      await browserPatch<OrganizationUnit>(organizationUnitPath(`/api/organization-units/${unit.id}`, selectedTenantId), payload);
      setMessage("已保存");
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  async function deleteUnit() {
    if (!confirm(`确定删除“${unit.name}”吗？删除前相关占用必须为 0。`)) {
      return;
    }
    setPending(true);
    setMessage("");
    try {
      await browserDelete<OrganizationUnit>(organizationUnitPath(`/api/organization-units/${unit.id}`, selectedTenantId));
      setMessage("已删除");
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "删除失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded-md border bg-white p-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <p className="break-words text-sm font-medium text-slate-800">{unit.name}</p>
        {kind === "group" ? <p className="mt-1 break-words text-xs text-slate-500">所属部门：{unit.parentName || "未绑定部门"}</p> : null}
      </div>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <AdminDialog
          description={kind === "department" ? "修改部门名称会同步更新用户部门、仪器部门和团队所属部门。" : "修改团队名称会同步更新对应仪器和历史申请中的团队名称。"}
          title={`修改${kind === "department" ? "用户部门" : "部门二级团队"}`}
          trigger={
            <Button className="w-full sm:w-auto" disabled={pending} size="sm" variant="outline">
              <Pencil className="h-4 w-4" aria-hidden="true" />
              修改
            </Button>
          }
        >
          {(close) => (
            <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
              <label className="block space-y-2">
                <span className="text-sm font-medium">{kind === "department" ? "用户部门名称" : "团队名称"}</span>
                <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={unit.name} name="name" placeholder="填写新的名称" required />
                <span className="block break-words text-xs text-slate-500">当前：{unit.name}</span>
              </label>
              {kind === "group" ? <DepartmentSelect currentValue={unit.parentName} departments={departments} /> : null}
              <div className="flex justify-end">
                <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                  {pending ? "保存中..." : "保存"}
                </Button>
              </div>
            </form>
          )}
        </AdminDialog>
        <Button className="w-full sm:w-auto" disabled={pending} onClick={deleteUnit} size="sm" type="button" variant="destructive">
          <Trash2 className="h-4 w-4" aria-hidden="true" />
          删除
        </Button>
      </div>
      {message ? <span className="text-xs text-slate-500 sm:min-w-24">{message}</span> : null}
    </div>
  );
}

function DepartmentSelect({
  currentValue = "",
  departments,
}: {
  currentValue?: string;
  departments: OrganizationUnit[];
}) {
  const options = Array.from(new Set([currentValue, ...departments.map((department) => department.name)].filter(Boolean)));
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium">所属部门</span>
      <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={currentValue} name="parentName" required>
        <option value="">选择团队所属部门</option>
        {options.map((department) => (
          <option key={department} value={department}>
            {department}
          </option>
        ))}
      </select>
      {currentValue ? <span className="block break-words text-xs text-slate-500">当前：{currentValue}</span> : null}
    </label>
  );
}

function organizationUnitPath(path: string, tenantId: string) {
  const query = new URLSearchParams();
  if (tenantId) {
    query.set("tenantId", tenantId);
  }
  const value = query.toString();
  return value ? `${path}?${value}` : path;
}
