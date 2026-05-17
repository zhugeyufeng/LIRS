import type { ReactNode } from "react";
import { Wallet } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { AppShell } from "@/components/app-shell";
import { FinanceTabs } from "@/components/finance-tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { User } from "@/lib/api";
import { isFinanceAdminRole } from "@/lib/permissions";

type FinanceTab = "overview" | "instrument-fees" | "material-fees" | "bills" | "reconciliation" | "projects" | "invoices";

export function FinanceFrame({
  active,
  children,
  currentUser,
  description,
  title,
}: {
  active: FinanceTab;
  children: ReactNode;
  currentUser: User;
  description: string;
  title: string;
}) {
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

  if (isFinanceAdminRole(currentUser.role)) {
    return (
      <AdminShell active="finance" currentUser={currentUser} title={title} description={description}>
        <FinanceTabs active={active} />
        {children}
      </AdminShell>
    );
  }

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6">
        <p className="text-xs font-bold uppercase tracking-widest text-primary">财务与计费中心</p>
        <h1 className="mt-2 text-2xl font-bold text-slate-900 sm:text-3xl">{title}</h1>
        <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{description}</p>
      </div>
      <FinanceTabs active={active} />
      {children}
    </AppShell>
  );
}
