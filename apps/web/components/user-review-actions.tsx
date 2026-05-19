"use client";

import { FormEvent, startTransition, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Pencil, Plus, Save, Trash2 } from "lucide-react";
import { browserDelete, browserPatch, browserPost, OrganizationUnit, Tenant, User, UserCreatePayload, UserMembershipPayload, UserReviewPayload } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";
import { isTenantAdminRole, roleLabel } from "@/lib/permissions";

const baseRoles = [
  ["unassigned", "待分配"],
  ["student", "学生"],
  ["teacher", "教师"],
  ["researcher", "研究员"],
  ["group_leader", "负责人"],
];

const scopedAdminRoles = [
  ["material_admin", "试剂管理员"],
  ["finance_admin", "财务管理员"],
];

const systemAdminRoles = [
  ["tenant_admin", "机构管理员"],
  ["lab_admin", "实验室管理员"],
  ["super_admin", "系统超级管理员"],
];

const membershipRoles = [...baseRoles, ...scopedAdminRoles, ["tenant_admin", "机构管理员"], ["lab_admin", "实验室管理员"]];

export function UserCreateForm({ currentUser, departments, tenants }: { currentUser: User; departments: string[]; tenants: Tenant[] }) {
  const router = useRouter();
  const isSuperAdmin = currentUser.role === "super_admin";
  const tenantOptions = isSuperAdmin ? tenants : [];
  const [selectedTenantId, setSelectedTenantId] = useState(currentUser.tenantId);
  const [departmentOptions, setDepartmentOptions] = useState(() => mergeOptions(departments, currentUser.department));
  const [selectedDepartment, setSelectedDepartment] = useState(currentUser.department || departments[0] || "");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const roles = isSuperAdmin ? [...baseRoles, ...scopedAdminRoles, ...systemAdminRoles] : [...baseRoles, ...scopedAdminRoles];

  useEffect(() => {
    if (!isSuperAdmin || selectedTenantId === currentUser.tenantId) {
      const nextOptions = mergeOptions(departments, selectedDepartment);
      setDepartmentOptions(nextOptions);
      if (!nextOptions.includes(selectedDepartment)) {
        setSelectedDepartment(nextOptions[0] ?? "");
      }
      return;
    }
    const controller = new AbortController();
    fetch(`/api/organization-units?kind=department&tenantId=${encodeURIComponent(selectedTenantId)}`, {
      signal: controller.signal,
    })
      .then((response) => (response.ok ? response.json() : []))
      .then((items: OrganizationUnit[]) => {
        const nextOptions = mergeOptions(items.map((item) => item.name), "");
        setDepartmentOptions(nextOptions);
        setSelectedDepartment((current) => (nextOptions.includes(current) ? current : nextOptions[0] ?? ""));
      })
      .catch((error) => {
        if ((error as Error).name !== "AbortError") {
          setDepartmentOptions([]);
          setSelectedDepartment("");
        }
      });
    return () => controller.abort();
  }, [currentUser.tenantId, departments, isSuperAdmin, selectedDepartment, selectedTenantId]);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const form = new FormData(event.currentTarget);
    const payload: UserCreatePayload = {
      tenantId: isSuperAdmin ? String(form.get("tenantId") ?? selectedTenantId) : currentUser.tenantId,
      name: String(form.get("name") ?? ""),
      phone: String(form.get("phone") ?? ""),
      email: String(form.get("email") ?? ""),
      password: String(form.get("password") ?? ""),
      department: String(form.get("department") ?? selectedDepartment),
      groupName: "",
      role: String(form.get("role") ?? "unassigned"),
      status: String(form.get("status") ?? "active"),
    };
    try {
      const created = await browserPost<User>("/api/users", payload);
      setMessage(`已新增：${created.name}`);
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
    <div className="space-y-2">
      <AdminDialog
        description="后台新增人员会直接创建账号，并按所选机构、部门、角色和状态写入人员列表。"
        maxWidth="max-w-3xl"
        title="手动添加人员"
        trigger={
          <Button className="w-full sm:w-auto" size="sm">
            <Plus className="h-4 w-4" aria-hidden="true" />
            添加人员
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <div className="grid gap-4 md:grid-cols-2">
              {isSuperAdmin ? (
                <label className="block min-w-0 space-y-2">
                  <span className="text-sm font-medium">所属机构</span>
                  <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" name="tenantId" onChange={(event) => setSelectedTenantId(event.currentTarget.value)} value={selectedTenantId}>
                    {mergeTenants(tenantOptions, currentUser).map((tenant) => (
                      <option key={tenant.id} value={tenant.id}>
                        {tenant.name}
                      </option>
                    ))}
                  </select>
                </label>
              ) : (
                <input name="tenantId" type="hidden" value={currentUser.tenantId} />
              )}
              <Field label="姓名" name="name" placeholder="填写姓名" required />
              <Field label="邮箱" name="email" placeholder="填写邮箱" required type="email" />
              <Field label="手机号" name="phone" placeholder="填写手机号" required type="tel" />
              <Field label="初始密码" name="password" placeholder="至少 8 位" required type="password" />
              <label className="block min-w-0 space-y-2">
                <span className="text-sm font-medium">部门</span>
                <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" name="department" onChange={(event) => setSelectedDepartment(event.currentTarget.value)} required value={selectedDepartment}>
                  {departmentOptions.map((department) => (
                    <option key={department} value={department}>
                      {department}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block min-w-0 space-y-2">
                <span className="text-sm font-medium">角色</span>
                <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue="unassigned" name="role">
                  {roles.map(([value, label]) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block min-w-0 space-y-2">
                <span className="text-sm font-medium">账号状态</span>
                <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue="active" name="status">
                  <option value="pending_approval">待审核</option>
                  <option value="active">启用</option>
                  <option value="disabled">停用</option>
                </select>
              </label>
            </div>
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                <Save className="h-4 w-4" aria-hidden="true" />
                {pending ? "新增中..." : "新增人员"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

export function UserReviewActions({
  currentUser,
  user,
  departments,
  memberships = [],
  tenants,
}: {
  currentUser: User;
  user: User;
  departments: string[];
  memberships?: string[];
  tenants: Tenant[];
}) {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<"basic" | "membership">("basic");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const isSuperAdmin = currentUser.role === "super_admin";
  const canEditTarget = isSuperAdmin || !isTenantAdminRole(user.role);
  const canDeleteTarget = canEditTarget && currentUser.id !== user.id && user.status !== "disabled";
  const [selectedTenantId, setSelectedTenantId] = useState(user.tenantId);
  const [departmentOptions, setDepartmentOptions] = useState(() => mergeOptions(departments, user.department));
  const [selectedDepartment, setSelectedDepartment] = useState(user.department);
  const roles = isSuperAdmin ? [...baseRoles, ...scopedAdminRoles, ...systemAdminRoles] : [...baseRoles, ...scopedAdminRoles];

  useEffect(() => {
    if (!isSuperAdmin || selectedTenantId === currentUser.tenantId) {
      const nextOptions = mergeOptions(departments, selectedTenantId === user.tenantId ? user.department : "");
      setDepartmentOptions(nextOptions);
      if (!nextOptions.includes(selectedDepartment)) {
        setSelectedDepartment(nextOptions[0] ?? "");
      }
      return;
    }
    const controller = new AbortController();
    fetch(`/api/organization-units?kind=department&tenantId=${encodeURIComponent(selectedTenantId)}`, {
      signal: controller.signal,
    })
      .then((response) => (response.ok ? response.json() : []))
      .then((items: OrganizationUnit[]) => {
        const nextOptions = mergeOptions(items.map((item) => item.name), selectedTenantId === user.tenantId ? user.department : "");
        setDepartmentOptions(nextOptions);
        if (!nextOptions.includes(selectedDepartment)) {
          setSelectedDepartment(nextOptions[0] ?? "");
        }
      })
      .catch((error) => {
        if ((error as Error).name !== "AbortError") {
          const nextOptions = mergeOptions([], selectedTenantId === user.tenantId ? user.department : "");
          setDepartmentOptions(nextOptions);
          if (!nextOptions.includes(selectedDepartment)) {
            setSelectedDepartment(nextOptions[0] ?? "");
          }
        }
      });
    return () => controller.abort();
  }, [currentUser.tenantId, departments, isSuperAdmin, selectedDepartment, selectedTenantId, user.department, user.tenantId]);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    if (!confirmTwice(`确定修改人员“${user.name}”吗？`, "请再次确认。人员基础信息和权限状态会立即更新。")) {
      return;
    }
    setPending(true);
    setMessage("");
    const formData = new FormData(event.currentTarget);
    const payload: UserReviewPayload = {
      tenantId: isSuperAdmin ? String(formData.get("tenantId") ?? user.tenantId) : user.tenantId,
      role: String(formData.get("role") ?? user.role),
      groupName: "",
      department: String(formData.get("department") ?? selectedDepartment),
      email: String(formData.get("email") ?? user.email),
      phone: String(formData.get("phone") ?? user.phone),
      status: String(formData.get("status") ?? user.status),
      actor: currentUser.name,
    };
    try {
      const updated = await browserPatch<User>(`/api/users/${user.id}/review`, payload);
      setMessage(`已更新：${updated.status} / ${updated.role}`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "更新失败");
    } finally {
      setPending(false);
    }
  }

  async function deleteUser(close?: () => void) {
    if (!confirmTwice(`确定删除人员账号“${user.name}”吗？`, "请再次确认删除该人员账号。账号会被停用并从后台人员列表移除。")) {
      return;
    }
    setPending(true);
    setMessage("");
    try {
      const deleted = await browserDelete<User>(`/api/users/${user.id}`);
      setMessage(`已删除：${deleted.name}`);
      close?.();
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
    <div className="flex w-full min-w-0 flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
      <AdminDialog
        description={canEditTarget ? "先核对完整用户资料，再在基础信息和机构权限选项卡中维护账号。" : "机构管理员不能修改机构管理员、实验室管理员或系统超级管理员账号。"}
        maxWidth="max-w-5xl"
        title={`人员详情：${user.name}`}
        trigger={
          <Button
            className="w-full min-w-0 px-2 sm:w-auto"
            disabled={!canEditTarget}
            onClick={() => {
              setActiveTab("basic");
              setMessage("");
            }}
            size="sm"
            variant="outline"
          >
            <Pencil className="h-4 w-4" aria-hidden="true" />
            详情/修改
          </Button>
        }
      >
        {(close) => (
          <div className="space-y-4">
            <UserDetailSummary memberships={memberships} user={user} />
            <TabBar active={activeTab} tabs={userReviewTabs(isSuperAdmin)} onChange={(tab) => setActiveTab(tab)} />
            <section hidden={activeTab !== "basic"}>
              <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
                <div className="grid gap-4 md:grid-cols-2">
                  {isSuperAdmin ? (
                    <label className="block min-w-0 space-y-2">
                      <span className="text-sm font-medium">所属机构</span>
                      <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={user.tenantId} name="tenantId" onChange={(event) => setSelectedTenantId(event.currentTarget.value)}>
                        {mergeTenants(tenants, user).map((tenant) => (
                          <option key={tenant.id} value={tenant.id}>
                            {tenant.name}
                          </option>
                        ))}
                      </select>
                      <FieldHint value={`当前：${user.tenantName}`} />
                    </label>
                  ) : (
                    <input name="tenantId" type="hidden" value={user.tenantId} />
                  )}
                  <Field defaultValue={user.email} label="邮箱" name="email" type="email" />
                  <Field defaultValue={user.phone} label="手机号" name="phone" type="tel" />
                  <label className="block min-w-0 space-y-2">
                    <span className="text-sm font-medium">角色</span>
                    <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={user.role} name="role">
                      {ensureRoleOption(roles, user.role).map(([value, label]) => (
                        <option key={value} value={value}>
                          {label}
                        </option>
                      ))}
                    </select>
                    <FieldHint value={`当前：${roleLabel(user.role)}`} />
                  </label>
                  <label className="block min-w-0 space-y-2">
                    <span className="text-sm font-medium">部门</span>
                    <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" name="department" onChange={(event) => setSelectedDepartment(event.currentTarget.value)} value={selectedDepartment}>
                      {departmentOptions.map((department) => (
                        <option key={department} value={department}>
                          {department}
                        </option>
                      ))}
                    </select>
                    <FieldHint value={`当前：${formatCurrent(user.department)}`} />
                  </label>
                  <label className="block min-w-0 space-y-2">
                    <span className="text-sm font-medium">账号状态</span>
                    <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={user.status} name="status">
                      <option value="pending_approval">待审核</option>
                      <option value="active">启用</option>
                      <option value="disabled">停用</option>
                    </select>
                    <FieldHint value={`当前：${userStatusLabel(user.status)}`} />
                  </label>
                </div>
                <div className="flex justify-end">
                  <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                    <Save className="h-4 w-4" aria-hidden="true" />
                    {pending ? "保存中..." : "保存"}
                  </Button>
                </div>
              </form>
            </section>
            {isSuperAdmin ? (
              <section hidden={activeTab !== "membership"}>
                <UserMembershipPanel user={user} departments={departments} tenants={tenants} close={close} />
              </section>
            ) : null}
          </div>
        )}
      </AdminDialog>
      <AdminDialog
        description="删除后该账号将被停用，当前登录会话会失效；关联的预约、申领、审计记录仍会保留。"
        title={`确认删除：${user.name}`}
        trigger={
          <Button className="w-full min-w-0 px-2 sm:w-auto" disabled={!canDeleteTarget || pending} size="sm" variant="destructive">
            <Trash2 className="h-4 w-4" aria-hidden="true" />
            删除
          </Button>
        }
      >
        {(close) => (
          <div className="space-y-4">
            <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-900">
              <p className="font-bold">即将删除以下人员账号</p>
              <p className="mt-2 break-words">{user.name} / {user.email}</p>
              <p className="mt-1 break-words">{user.tenantName} / {roleLabel(user.role)} / {userStatusLabel(user.status)}</p>
            </div>
            <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} onClick={close} type="button" variant="outline">
                取消
              </Button>
              <Button className="w-full sm:w-auto" disabled={pending} onClick={() => deleteUser(close)} type="button" variant="destructive">
                <Trash2 className="h-4 w-4" aria-hidden="true" />
                {pending ? "删除中..." : "确认删除"}
              </Button>
            </div>
          </div>
        )}
      </AdminDialog>
      {message ? <span className="text-xs text-slate-500">{message}</span> : null}
    </div>
  );
}

function UserDetailSummary({ memberships, user }: { memberships: string[]; user: User }) {
  return (
    <section className="min-w-0 rounded-lg border bg-slate-50 p-4">
      <div className="grid gap-3 text-sm md:grid-cols-2 xl:grid-cols-3">
        <DetailItem label="姓名" value={user.name} />
        <DetailItem label="邮箱" value={user.email} />
        <DetailItem label="手机号" value={user.phone} />
        <DetailItem label="当前机构" value={user.tenantName} />
        <DetailItem label="部门" value={user.department} />
        <DetailItem label="角色" value={roleLabel(user.role)} />
        <DetailItem label="账号状态" value={userStatusLabel(user.status)} />
        <DetailItem label="邮箱验证" value={user.emailVerified ? "已验证" : "未验证"} />
        <DetailItem label="财务模块" value={user.financeEnabled ? "已启用" : "未启用"} />
      </div>
      <div className="mt-4 border-t pt-3">
        <p className="text-xs text-slate-500">同账号机构权限</p>
        {memberships.length > 0 ? (
          <div className="mt-2 flex flex-wrap gap-2">
            {memberships.map((membership) => (
              <span className="max-w-full break-words rounded bg-white px-2 py-1 text-xs font-medium text-slate-700 ring-1 ring-slate-200" key={membership}>
                {membership}
              </span>
            ))}
          </div>
        ) : (
          <p className="mt-1 text-sm font-medium text-slate-800">未设置</p>
        )}
      </div>
    </section>
  );
}

function DetailItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{formatCurrent(value)}</p>
    </div>
  );
}

function UserMembershipPanel({
  close,
  departments,
  tenants,
  user,
}: {
  close?: () => void;
  departments: string[];
  tenants: Tenant[];
  user: User;
}) {
  const router = useRouter();
  const tenantOptions = mergeTenants(tenants, user);
  const defaultTenantId = tenantOptions.find((tenant) => tenant.id !== user.tenantId)?.id ?? user.tenantId;
  const defaultRole = user.role === "super_admin" ? "tenant_admin" : user.role;
  const [selectedTenantId, setSelectedTenantId] = useState(defaultTenantId);
  const [selectedDepartment, setSelectedDepartment] = useState(user.department);
  const [departmentOptions, setDepartmentOptions] = useState(() => mergeOptions(departments, user.department));
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    if (!selectedTenantId) {
      setDepartmentOptions([]);
      setSelectedDepartment("");
      return;
    }
    const controller = new AbortController();
    fetch(`/api/organization-units?kind=department&tenantId=${encodeURIComponent(selectedTenantId)}`, {
      signal: controller.signal,
    })
      .then((response) => (response.ok ? response.json() : []))
      .then((items: OrganizationUnit[]) => {
        const fallbackDepartment = selectedTenantId === user.tenantId ? user.department : "";
        const nextOptions = mergeOptions(items.map((item) => item.name), fallbackDepartment);
        setDepartmentOptions(nextOptions);
        setSelectedDepartment((current) => (nextOptions.includes(current) ? current : nextOptions[0] ?? ""));
      })
      .catch((error) => {
        if ((error as Error).name !== "AbortError") {
          const fallbackDepartment = selectedTenantId === user.tenantId ? user.department : "";
          const nextOptions = mergeOptions([], fallbackDepartment);
          setDepartmentOptions(nextOptions);
          setSelectedDepartment((current) => (nextOptions.includes(current) ? current : nextOptions[0] ?? ""));
        }
      });
    return () => controller.abort();
  }, [selectedTenantId, user.department, user.tenantId]);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formData = new FormData(event.currentTarget);
    const payload: UserMembershipPayload = {
      tenantId: String(formData.get("tenantId") ?? selectedTenantId),
      role: String(formData.get("role") ?? defaultRole),
      groupName: "",
      department: String(formData.get("department") ?? selectedDepartment),
      status: String(formData.get("status") ?? "active"),
    };
    try {
      const updated = await browserPost<User>(`/api/users/${user.id}/memberships`, payload);
      setMessage(`已设置：${updated.tenantName} / ${roleLabel(updated.role)}`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "设置失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-3 rounded-lg border bg-slate-50 p-4">
      <div>
        <p className="text-sm font-medium text-slate-900">机构权限</p>
        <p className="mt-1 text-xs leading-5 text-slate-500">当前机构信息在“基础信息”页签维护，这里为同一用户添加或更新其他机构的角色与状态。</p>
      </div>
      <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
        <div className="grid gap-4 md:grid-cols-2">
          <label className="block min-w-0 space-y-2">
            <span className="text-sm font-medium">机构</span>
            <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" name="tenantId" onChange={(event) => setSelectedTenantId(event.currentTarget.value)} value={selectedTenantId}>
              {tenantOptions.map((tenant) => (
                <option key={tenant.id} value={tenant.id}>
                  {tenant.id === user.tenantId ? `${tenant.name}（当前机构）` : tenant.name}
                </option>
              ))}
            </select>
            <FieldHint value={`当前账号：${user.email}`} />
          </label>
          <label className="block min-w-0 space-y-2">
            <span className="text-sm font-medium">角色</span>
            <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={defaultRole} name="role">
              {ensureRoleOption(membershipRoles, defaultRole)
                .filter(([value]) => value !== "super_admin")
                .map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
            </select>
            <FieldHint value={`默认沿用：${roleLabel(defaultRole)}`} />
          </label>
          <label className="block min-w-0 space-y-2">
            <span className="text-sm font-medium">部门</span>
            <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" name="department" onChange={(event) => setSelectedDepartment(event.currentTarget.value)} value={selectedDepartment}>
              {departmentOptions.map((department) => (
                <option key={department} value={department}>
                  {department}
                </option>
              ))}
            </select>
            <FieldHint value={selectedTenantId === user.tenantId ? `当前：${formatCurrent(user.department)}` : "目标机构下单独设置"} />
          </label>
          <label className="block min-w-0 space-y-2">
            <span className="text-sm font-medium">账号状态</span>
            <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue="active" name="status">
              <option value="pending_approval">待审核</option>
              <option value="active">启用</option>
              <option value="disabled">停用</option>
            </select>
            <FieldHint value="目标机构权限状态：默认启用" />
          </label>
        </div>
        <div className="flex justify-end">
          <Button className="w-full sm:w-auto" disabled={pending} type="submit">
            <Save className="h-4 w-4" aria-hidden="true" />
            {pending ? "保存中..." : "保存机构权限"}
          </Button>
        </div>
      </form>
      {message ? <p className="text-xs text-slate-500">{message}</p> : null}
    </div>
  );
}

function Field({
  defaultValue,
  label,
  name,
  placeholder,
  required = true,
  type = "text",
}: {
  defaultValue?: string;
  label: string;
  name: string;
  placeholder?: string;
  required?: boolean;
  type?: string;
}) {
  return (
    <label className="block min-w-0 space-y-2">
      <span className="text-sm font-medium">{label}</span>
      <input className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={defaultValue ?? ""} name={name} placeholder={placeholder} required={required} type={type} />
      {defaultValue !== undefined ? <FieldHint value={`当前：${formatCurrent(defaultValue)}`} /> : null}
    </label>
  );
}

function FieldHint({ value }: { value: string }) {
  return <span className="block break-words text-xs text-slate-500">{value}</span>;
}

function TabBar({
  active,
  tabs,
  onChange,
}: {
  active: "basic" | "membership";
  tabs: Array<{ key: "basic" | "membership"; label: string }>;
  onChange: (tab: "basic" | "membership") => void;
}) {
  return (
    <div className="inline-flex w-full flex-wrap gap-2 rounded-lg border bg-slate-50 p-1" role="tablist">
      {tabs.map((tab) => (
        <button
          aria-selected={active === tab.key}
          className={
            active === tab.key
              ? "inline-flex h-10 flex-1 items-center justify-center rounded-md bg-white px-3 text-sm font-semibold text-slate-900 shadow-sm"
              : "inline-flex h-10 flex-1 items-center justify-center rounded-md px-3 text-sm font-medium text-slate-600 hover:bg-white/70"
          }
          key={tab.key}
          onClick={() => onChange(tab.key)}
          role="tab"
          type="button"
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}

function userReviewTabs(isSuperAdmin: boolean): Array<{ key: "basic" | "membership"; label: string }> {
  return [
    { key: "basic", label: "基础信息" },
    ...(isSuperAdmin ? [{ key: "membership" as const, label: "机构权限" }] : []),
  ];
}

function formatCurrent(value: string) {
  const text = value.trim();
  return text === "" ? "未设置" : text;
}

function userStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending_approval: "待审核",
    active: "启用",
    disabled: "停用",
  };
  return labels[status] ?? status;
}

function mergeOptions(options: string[], current: string) {
  return Array.from(new Set([current, ...options].filter(Boolean)));
}

function ensureRoleOption(roles: string[][], currentRole: string) {
  if (roles.some(([value]) => value === currentRole)) {
    return roles;
  }
  return [[currentRole, roleLabel(currentRole)], ...roles];
}

function mergeTenants(tenants: Tenant[], user: User) {
  if (tenants.some((tenant) => tenant.id === user.tenantId)) {
    return tenants;
  }
  return [
    {
      id: user.tenantId,
      name: user.tenantName,
      code: user.tenantCode || user.tenantId,
      financeEnabled: user.financeEnabled,
      status: "active",
      createdAt: "",
      updatedAt: "",
    },
    ...tenants,
  ];
}
