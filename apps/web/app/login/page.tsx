import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { DingTalkQuickLogin } from "@/components/dingtalk-quick-login";
import { DingTalkWebLogin } from "@/components/dingtalk-web-login";
import { LoginForm } from "@/components/login-form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function LoginPage() {
  const tenants = await api.tenants().catch(() => []);
  return (
    <AppShell>
      <div className="mx-auto max-w-md">
        <Link className="mb-4 inline-flex items-center gap-2 text-sm text-slate-600 hover:text-primary" href="/">
          <ArrowLeft className="h-4 w-4" />
          返回系统首页
        </Link>
        <Card>
          <CardHeader>
            <CardTitle>账号登录</CardTitle>
          </CardHeader>
          <CardContent>
            <DingTalkQuickLogin tenants={tenants} />
            <DingTalkWebLogin tenants={tenants} />
            <div className="my-4 flex items-center gap-3">
              <span className="h-px flex-1 bg-slate-200" />
              <span className="text-xs text-slate-500">或使用账号密码</span>
              <span className="h-px flex-1 bg-slate-200" />
            </div>
            <LoginForm tenants={tenants} />
            <p className="mt-4 text-xs text-slate-500">初始管理员登录信息请查看 README.md。</p>
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}
