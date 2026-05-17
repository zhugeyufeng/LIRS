import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { RegisterForm } from "@/components/register-form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function RegisterPage() {
  const tenants = await api.tenants().catch(() => []);
  const initialTenantId = tenants[0]?.id ?? "";
  const departments = initialTenantId ? await api.organizationUnits("department", initialTenantId).then((items) => items.map((item) => item.name)).catch(() => []) : [];

  return (
    <AppShell>
      <div className="mx-auto max-w-xl">
        <Link className="mb-4 inline-flex items-center gap-2 text-sm text-slate-600 hover:text-primary" href="/">
          <ArrowLeft className="h-4 w-4" />
          返回系统首页
        </Link>
        <Card>
          <CardHeader>
            <CardTitle>普通用户注册</CardTitle>
          </CardHeader>
          <CardContent>
            <RegisterForm departments={departments} tenants={tenants} />
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}
