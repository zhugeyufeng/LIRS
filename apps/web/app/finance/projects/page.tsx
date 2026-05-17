import { FinanceFrame } from "@/components/finance-frame";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function FinanceProjectsPage() {
  const currentUser = await api.me();
  const accounts = currentUser.role === "super_admin" || currentUser.financeEnabled ? await api.financialAccounts().catch(() => []) : [];

  return (
    <FinanceFrame active="projects" currentUser={currentUser} title="项目/个人经费" description="按个人维护经费账户、余额和授信额度；财务系统当前仅针对个人管理。">
      <Card>
        <CardHeader>
          <CardTitle>个人经费账户</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-2">
          {accounts.map((account) => {
            const available = account.balance + account.creditLimit;
            return (
              <article className="rounded-lg border bg-white p-4" key={account.id || account.userId}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <h2 className="break-words font-bold text-slate-900">{account.userName}</h2>
                    <p className="mt-1 break-words text-xs text-slate-500">{account.department} / {account.groupName || "个人账户"}</p>
                  </div>
                  <span className={available < 0 ? "w-fit rounded-full bg-rose-50 px-2 py-1 text-xs font-bold text-rose-700" : "w-fit rounded-full bg-emerald-50 px-2 py-1 text-xs font-bold text-emerald-700"}>
                    可用 ¥{available.toFixed(2)}
                  </span>
                </div>
                <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                  <Info label="余额" value={`¥${account.balance.toFixed(2)}`} />
                  <Info label="授信" value={`¥${account.creditLimit.toFixed(2)}`} />
                </div>
              </article>
            );
          })}
          {accounts.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无经费账户。</p> : null}
        </CardContent>
      </Card>
    </FinanceFrame>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded bg-slate-50 p-3">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 font-bold text-slate-900">{value}</p>
    </div>
  );
}
