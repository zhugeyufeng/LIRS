import { CreditCard, ReceiptText, Wallet } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { AppShell } from "@/components/app-shell";
import { FinancialAccountForm } from "@/components/financial-account-form";
import { FinanceTabs } from "@/components/finance-tabs";
import { LedgerAdjustmentForm } from "@/components/ledger-adjustment-form";
import { LedgerExportButton } from "@/components/ledger-export-button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type User } from "@/lib/api";
import { isFinanceAdminRole } from "@/lib/permissions";

export default async function FinancePage({
  searchParams,
}: {
  searchParams?: Promise<{ userId?: string; type?: string; accountId?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const currentUser = await api.me();
  if (currentUser.role !== "super_admin" && !currentUser.financeEnabled) {
    return (
      <AppShell currentUser={currentUser}>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wallet className="h-5 w-5 text-primary" />
              财务模块未启用
            </CardTitle>
          </CardHeader>
          <CardContent className="text-sm leading-6 text-slate-600">当前机构未启用财务模块，相关菜单和接口已关闭。</CardContent>
        </Card>
      </AppShell>
    );
  }
  const [ledger, accounts] = await Promise.all([api.ledger(), api.financialAccounts()]);
  const isAdmin = isFinanceAdminRole(currentUser.role);
  let users: User[] = [];
  if (isAdmin) {
    users = await api.users();
  }
  const scopedUsers = users.filter((user) => user.tenantId === currentUser.tenantId);
  const accountUsers = scopedUsers.filter((user) => user.status === "active");
  const visibleLedger = ledger.filter((item) => {
    const matchesUser = !params.userId || item.userId === params.userId;
    const matchesType = !params.type || item.entryType === params.type;
    return matchesUser && matchesType;
  });
  const total = visibleLedger.reduce((sum, item) => sum + (item.amount ?? 0), 0);
  const balance = accounts.reduce((sum, item) => sum + (item.balance ?? 0), 0);
  const credit = accounts.reduce((sum, item) => sum + (item.creditLimit ?? 0), 0);

  const content = (
    <>
      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <Metric label="流水条数" value={visibleLedger.length} />
        <Metric label="费用/调整合计" value={`¥${total.toFixed(2)}`} />
        <Metric label="账户余额合计" value={`¥${balance.toFixed(2)}`} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CreditCard className="h-5 w-5 text-primary" />
                个人账户
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3 md:grid-cols-2">
              {accounts.map((account) => {
                const available = (account.balance ?? 0) + (account.creditLimit ?? 0);
                return (
                  <div className="rounded-lg border p-4" key={account.id || account.userId}>
                    <div className="flex flex-col items-start justify-between gap-3 sm:flex-row">
                      <div className="min-w-0">
                        <p className="break-words font-bold text-primary">{account.userName}</p>
                        <p className="mt-1 text-xs text-slate-500">{account.department} · 更新：{formatDateTime(account.updatedAt)}</p>
                      </div>
                      <span className={available < 0 ? "shrink-0 rounded bg-red-50 px-2 py-1 text-xs font-bold text-red-700" : "shrink-0 rounded bg-emerald-50 px-2 py-1 text-xs font-bold text-emerald-700"}>
                        可用 ¥{available.toFixed(2)}
                      </span>
                    </div>
                    <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                      <div className="rounded bg-slate-50 p-3">
                        <p className="text-xs text-slate-500">余额</p>
                        <p className="mt-1 font-bold">¥{(account.balance ?? 0).toFixed(2)}</p>
                      </div>
                      <div className="rounded bg-slate-50 p-3">
                        <p className="text-xs text-slate-500">授信</p>
                        <p className="mt-1 font-bold">¥{(account.creditLimit ?? 0).toFixed(2)}</p>
                      </div>
                    </div>
                    {isAdmin ? (
                      <div className="mt-4">
                        <FinancialAccountForm account={account} users={accountUsers} />
                      </div>
                    ) : null}
                  </div>
                );
              })}
              {accounts.length === 0 ? <p className="text-sm text-slate-500">暂无个人账户。</p> : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ReceiptText className="h-5 w-5 text-primary" />
                费用流水
              </CardTitle>
            </CardHeader>
            <CardContent>
              <form action="/finance" className="mb-4 grid gap-3 xl:grid-cols-[180px_180px_auto] xl:items-center">
                <select className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={params.userId ?? ""} name="userId">
                  <option value="">全部人员</option>
                  {accountUsers.map((user) => (
                    <option key={user.id} value={user.id}>
                      {user.name}
                    </option>
                  ))}
                </select>
                <select className="h-10 min-w-0 rounded-md border bg-white px-3 text-sm" defaultValue={params.type ?? ""} name="type">
                  <option value="">全部类型</option>
                  <option value="debit">预约扣费</option>
                  <option value="adjustment">调整</option>
                  <option value="account_init">账户初始化</option>
                </select>
                <button className="inline-flex h-10 w-full min-w-20 items-center justify-center whitespace-nowrap rounded-md bg-primary px-4 text-sm font-bold text-white xl:w-auto" type="submit">
                  筛选
                </button>
              </form>
              <div className="grid gap-3 xl:hidden">
                {visibleLedger.map((item) => (
                  <div className="rounded-lg border bg-white p-4 text-sm" key={item.id}>
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-medium">{item.description}</p>
                        <p className="mt-1 break-words text-xs text-slate-500">{item.id}</p>
                      </div>
                      <span className="shrink-0 whitespace-nowrap font-bold text-primary">¥{(item.amount ?? 0).toFixed(2)}</span>
                    </div>
                    <div className="mt-3 grid gap-2 text-xs text-slate-500 sm:grid-cols-3">
                      <span>{item.userName || item.groupName}</span>
                      <span>{entryTypeLabel(item.entryType)}</span>
                      <span>{formatDateTime(item.createdAt)}</span>
                    </div>
                  </div>
                ))}
              </div>
              <div className="hidden overflow-x-auto xl:block">
                <table className="w-full text-left text-sm">
                  <thead className="border-b text-slate-500">
                    <tr>
                      <th className="py-3">说明</th>
                      <th className="py-3">人员</th>
                      <th className="py-3">类型</th>
                      <th className="py-3">时间</th>
                      <th className="py-3 text-right">金额</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y">
                    {visibleLedger.map((item) => (
                      <tr key={item.id}>
                        <td className="py-3">
                          <p className="font-medium">{item.description}</p>
                          <p className="text-xs text-slate-500">{item.id}</p>
                        </td>
                        <td className="py-3">{item.userName || item.groupName}</td>
                        <td className="py-3">{entryTypeLabel(item.entryType)}</td>
                        <td className="py-3 text-xs text-slate-500">{formatDateTime(item.createdAt)}</td>
                        <td className="py-3 text-right font-bold">¥{(item.amount ?? 0).toFixed(2)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {visibleLedger.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">当前筛选下暂无流水。</p> : null}
            </CardContent>
          </Card>
        </div>

        {isAdmin ? (
          <aside className="min-w-0 space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <CreditCard className="h-5 w-5 text-primary" />
                  创建个人账户
                </CardTitle>
              </CardHeader>
              <CardContent>
                <FinancialAccountForm users={accountUsers} />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Wallet className="h-5 w-5 text-primary" />
                  账务调整
                </CardTitle>
              </CardHeader>
              <CardContent>
                <LedgerAdjustmentForm users={accountUsers} />
                <div className="mt-4 rounded-lg bg-slate-50 p-3 text-xs leading-5 text-slate-500">
                  历史流水由数据库触发器保护不可更新或删除。错误扣费通过这里生成个人调整流水，并同步更新个人账户余额。当前总授信 ¥{credit.toFixed(2)}。
                </div>
              </CardContent>
            </Card>
          </aside>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Wallet className="h-5 w-5 text-primary" />
                我的财务范围
              </CardTitle>
            </CardHeader>
            <CardContent className="text-sm leading-6 text-slate-600">
              当前账号只能查看授权范围内的账户和流水。费用纠错由实验室管理员生成调整流水。
            </CardContent>
          </Card>
        )}
      </div>
    </>
  );

  if (isAdmin) {
    return (
      <AdminShell active="finance" currentUser={currentUser} title="财务管理中心" description="预约完成自动入账；历史流水不可修改，纠错通过调整流水保留轨迹。">
        <FinanceTabs active="overview" />
        <div className="mb-6 flex justify-start sm:justify-end">
          <LedgerExportButton />
        </div>
        {content}
      </AdminShell>
    );
  }

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-3 md:flex-row md:items-end">
        <div>
          <h1 className="text-2xl font-bold">财务管理中心</h1>
          <p className="mt-1 text-sm text-muted-foreground">预约完成自动入账；历史流水不可修改，纠错通过调整流水保留轨迹。</p>
        </div>
        <LedgerExportButton />
      </div>
      <FinanceTabs active="overview" />
      {content}
    </AppShell>
  );
}

function entryTypeLabel(type: string) {
  const labels: Record<string, string> = {
    debit: "预约扣费",
    adjustment: "调整",
    account_init: "账户初始化",
  };
  return labels[type] ?? type;
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
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
