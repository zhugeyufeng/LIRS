import Link from "next/link";
import { Building2, UsersRound } from "lucide-react";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { UserReviewActions } from "@/components/user-review-actions";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { roleLabel } from "@/lib/permissions";

export default async function AdminUsersPage({
  searchParams,
}: {
  searchParams?: Promise<{ userSearch?: string; userStatus?: string; userPage?: string }>;
}) {
  const currentUser = await requireAdminSection("users");
  const params = (await searchParams) ?? {};
  const [rawUsers, organizationUnits, tenants] = await Promise.all([
    api.users().catch(() => []),
    api.organizationUnits().catch(() => []),
    currentUser.role === "super_admin" ? api.tenants().catch(() => []) : Promise.resolve([]),
  ]);
  const users = rawUsers.map((user) => ({ ...user, groupName: "" }));
  const departments = organizationUnits.filter((item) => item.kind === "department");
  const pendingUsers = users.filter((item) => item.status === "pending_approval").length;
  const activeUsers = users.filter((item) => item.status === "active").length;
  const disabledUsers = users.filter((item) => item.status === "disabled").length;
  const membershipsByEmail = users.reduce((groups, user) => {
    const key = user.email.toLowerCase();
    const next = groups.get(key) ?? [];
    next.push(`${user.tenantName} / ${roleLabel(user.role)} / ${userStatusLabel(user.status)}`);
    groups.set(key, next);
    return groups;
  }, new Map<string, string[]>());
  const userSearch = (params.userSearch ?? "").trim().toLowerCase();
  const userStatus = params.userStatus ?? "";
  const userPageSize = 10;
  const userPage = Math.max(Number(params.userPage ?? 1) || 1, 1);
  const visibleUsers = users.filter((user) => {
    const matchesSearch =
      userSearch === "" ||
      [user.name, user.email, user.phone, user.department, user.tenantName].some((value) => value.toLowerCase().includes(userSearch));
    const matchesStatus = userStatus === "" || user.status === userStatus;
    return matchesSearch && matchesStatus;
  });
  const userTotalPages = Math.max(Math.ceil(visibleUsers.length / userPageSize), 1);
  const currentUserPage = Math.min(userPage, userTotalPages);
  const pagedUsers = visibleUsers.slice((currentUserPage - 1) * userPageSize, currentUserPage * userPageSize);

  return (
    <AdminShell active="users" title="人员管理" description="集中处理用户审核、角色、部门和账号状态。">
      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="总用户" value={users.length} />
        <Metric label="待审核" value={pendingUsers} />
        <Metric label="启用账号" value={activeUsers} />
        <Metric label="停用账号" value={disabledUsers} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(280px,360px)]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex min-w-0 items-center gap-2">
              <UsersRound className="h-5 w-5 text-primary" />
              <span className="min-w-0 break-words">用户审核与角色分配</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <form action="/admin/users" className="mb-4 grid gap-3 xl:grid-cols-[minmax(0,1fr)_180px_auto]">
              <input
                className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm"
                defaultValue={params.userSearch ?? ""}
                name="userSearch"
                placeholder="搜索姓名、邮箱、手机号、部门、机构"
              />
              <select className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={params.userStatus ?? ""} name="userStatus">
                <option value="">全部状态</option>
                <option value="pending_approval">待审核</option>
                <option value="active">启用</option>
                <option value="disabled">停用</option>
              </select>
              <button className="inline-flex h-10 w-full min-w-20 items-center justify-center whitespace-nowrap rounded-md bg-primary px-4 text-sm font-bold text-white xl:w-auto" type="submit">
                筛选
              </button>
            </form>

            {pagedUsers.length > 0 ? (
              <div className="rounded-lg border bg-white" role="table" aria-label="用户审核与角色分配列表">
                <div className="hidden border-b bg-slate-50 px-4 py-3 text-sm font-medium text-slate-500 xl:grid xl:grid-cols-[minmax(180px,1.8fr)_minmax(128px,1.1fr)_minmax(96px,0.8fr)_72px_104px] xl:gap-3" role="row">
                  <div role="columnheader">用户</div>
                  <div role="columnheader">机构</div>
                  <div role="columnheader">角色</div>
                  <div role="columnheader">状态</div>
                  <div className="text-right" role="columnheader">操作</div>
                </div>
                <div className="divide-y" role="rowgroup">
                  {pagedUsers.map((user) => (
                    <article className="grid gap-4 p-4 text-sm xl:grid-cols-[minmax(180px,1.8fr)_minmax(128px,1.1fr)_minmax(96px,0.8fr)_72px_104px] xl:items-start xl:gap-3" key={user.id} role="row">
                      <div className="min-w-0" role="cell">
                        <p className="break-words font-bold text-slate-900 xl:font-medium">{user.name}</p>
                        <p className="mt-1 break-words text-xs text-slate-500">{user.email}</p>
                      </div>
                      <div className="min-w-0" role="cell">
                        <p className="text-xs font-medium text-slate-500 xl:hidden">机构</p>
                        <p className="mt-1 break-words font-medium text-slate-800 xl:mt-0 xl:font-normal">{user.tenantName}</p>
                      </div>
                      <div className="min-w-0" role="cell">
                        <p className="text-xs font-medium text-slate-500 xl:hidden">角色</p>
                        <p className="mt-1 break-words font-medium text-slate-800 xl:mt-0 xl:font-normal">{roleLabel(user.role)}</p>
                      </div>
                      <div className="min-w-0" role="cell">
                        <p className="text-xs font-medium text-slate-500 xl:hidden">状态</p>
                        <span className="mt-1 inline-flex w-fit rounded bg-slate-100 px-2 py-1 text-xs font-bold text-slate-700 xl:mt-0">{userStatusLabel(user.status)}</span>
                      </div>
                      <div className="min-w-0 xl:flex xl:justify-end" role="cell">
                        <p className="text-xs font-medium text-slate-500 xl:hidden">操作</p>
                        <div className="mt-2 min-w-0 max-w-full xl:mt-0 xl:w-full [&_button]:min-w-0 [&_button]:max-w-full [&_button]:px-2">
                          <UserReviewActions
                            currentUser={currentUser}
                            departments={departments.map((item) => item.name)}
                            memberships={membershipsByEmail.get(user.email.toLowerCase()) ?? []}
                            tenants={tenants}
                            user={user}
                          />
                        </div>
                      </div>
                    </article>
                  ))}
                </div>
              </div>
            ) : (
              <p className="rounded-lg border p-4 text-sm text-slate-500">当前筛选下暂无用户。</p>
            )}

            <div className="mt-4 flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
              <p className="break-words text-sm text-muted-foreground">
                共 {visibleUsers.length} 人，当前第 {currentUserPage} / {userTotalPages} 页
              </p>
              <div className="grid grid-cols-2 gap-2 sm:flex">
                <a
                  className="inline-flex h-8 min-w-16 items-center justify-center whitespace-nowrap rounded-md border px-3 text-xs font-medium text-slate-600 hover:bg-slate-50"
                  href={usersHref(params, Math.max(currentUserPage - 1, 1))}
                >
                  上一页
                </a>
                <a
                  className="inline-flex h-8 min-w-16 items-center justify-center whitespace-nowrap rounded-md border px-3 text-xs font-medium text-slate-600 hover:bg-slate-50"
                  href={usersHref(params, Math.min(currentUserPage + 1, userTotalPages))}
                >
                  下一页
                </a>
              </div>
            </div>
          </CardContent>
        </Card>

        <aside className="min-w-0">
          <Card>
            <CardHeader>
              <CardTitle>组织架构管理</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="rounded-lg border bg-slate-50 p-4 text-sm leading-6 text-slate-600">
                用户仅保留部门属性；仪器团队信息在平台配置中心单独维护，用于确定仪器管理边界。
              </div>
              <div className="grid gap-3">
                <Metric label="部门/实验室" value={departments.length} />
              </div>
              <Link className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-primary px-4 text-sm font-bold text-white" href="/admin/settings/organization">
                <Building2 className="h-4 w-4" aria-hidden="true" />
                管理组织数据
              </Link>
            </CardContent>
          </Card>
        </aside>
      </div>
    </AdminShell>
  );
}

function usersHref(params: { userSearch?: string; userStatus?: string; userPage?: string }, page: number) {
  const query = new URLSearchParams();
  if (params.userSearch) {
    query.set("userSearch", params.userSearch);
  }
  if (params.userStatus) {
    query.set("userStatus", params.userStatus);
  }
  query.set("userPage", String(page));
  return `/admin/users?${query.toString()}`;
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  );
}

function userStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending_approval: "待审核",
    active: "启用",
    disabled: "停用",
  };
  return labels[status] ?? status;
}
