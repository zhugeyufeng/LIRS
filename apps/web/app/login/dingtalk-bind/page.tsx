import Link from "next/link";
import { redirect } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { DingTalkLoginBindForm } from "@/components/dingtalk-login-bind-form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

type SearchParams = {
  bindToken?: string;
  tenantId?: string;
  tenantCode?: string;
  dingTalkName?: string;
  next?: string;
};

export default async function DingTalkLoginBindPage({ searchParams }: { searchParams?: Promise<SearchParams> }) {
  const params = (await searchParams) ?? {};
  const bindingToken = params.bindToken?.trim() ?? "";
  if (!bindingToken) {
    redirect("/login");
  }
  const tenants = await api.tenants().catch(() => []);
  const tenant = resolveTenant(tenants, params.tenantId ?? "", params.tenantCode ?? "");
  const nextPath = safeNextPath(params.next ?? "") ?? "/dashboard";

  return (
    <AppShell>
      <div className="mx-auto max-w-md">
        <Link className="mb-4 inline-flex items-center gap-2 text-sm text-slate-600 hover:text-primary" href="/login">
          <ArrowLeft className="h-4 w-4" />
          返回登录
        </Link>
        <Card>
          <CardHeader>
            <CardTitle>绑定现有账户</CardTitle>
          </CardHeader>
          <CardContent>
            <DingTalkLoginBindForm
              bindingToken={bindingToken}
              dingTalkName={params.dingTalkName ?? ""}
              nextPath={nextPath}
              tenantName={tenant?.name ?? params.tenantCode ?? ""}
            />
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}

function resolveTenant(tenants: { id: string; code: string; name: string }[], tenantId: string, tenantCode: string) {
  const normalizedCode = tenantCode.trim().toLowerCase();
  return tenants.find((tenant) => (tenantId && tenant.id === tenantId) || (normalizedCode && tenant.code.toLowerCase() === normalizedCode));
}

function safeNextPath(next: string) {
  if (!next || !next.startsWith("/") || next.startsWith("//")) {
    return null;
  }
  return next;
}
