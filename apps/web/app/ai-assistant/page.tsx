import { Bot, MessageSquareText, Search, Sparkles } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { AssistantQueryDeleteButton, AssistantQueryForm } from "@/components/extension-forms";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { isTenantAdminRole } from "@/lib/permissions";

export default async function AiAssistantPage({
  searchParams,
}: {
  searchParams?: Promise<{ q?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const query = (params.q ?? "").trim().toLowerCase();
  const currentUser = await api.me();
  const queries = await api.assistantQueries().catch(() => []);
  const visibleQueries = queries.filter((item) => [item.question, item.answer, item.context].some((value) => String(value ?? "").toLowerCase().includes(query)));
  const recentCount = queries.filter((item) => Date.now() - new Date(item.createdAt).getTime() <= 7 * 24 * 60 * 60 * 1000).length;
  const canManageQueries = isTenantAdminRole(currentUser.role);

  return (
    <AppShell currentUser={currentUser}>
      <div className="mb-6 flex flex-col justify-between gap-4 xl:flex-row xl:items-end">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold">AI 助手</h1>
          <p className="mt-1 text-sm text-muted-foreground">基于当前机构数据回答预约、培训、样本和物联网设备相关问题。</p>
        </div>
        <form action="/ai-assistant" className="flex w-full max-w-xl gap-2 xl:w-auto">
          <div className="relative min-w-0 flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <input className="h-10 w-full rounded-md border bg-white pl-9 pr-3 text-sm" defaultValue={params.q ?? ""} name="q" placeholder="搜索问题、回答或背景..." />
          </div>
          <button className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-bold text-white" type="submit">
            筛选
          </button>
        </form>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="历史问答" value={queries.length} />
        <Metric label="近 7 天提问" value={recentCount} />
        <Metric label="当前租户" value={currentUser.tenantName} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MessageSquareText className="h-5 w-5 text-primary" />
              最近问答
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {visibleQueries.map((item) => (
              <div className="rounded-lg border p-4" key={item.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <p className="break-words font-semibold text-slate-900">{item.question}</p>
                    <p className="mt-1 break-words text-xs text-slate-500">{formatDateTime(item.createdAt)}</p>
                  </div>
                  <span className="w-fit rounded bg-primary/10 px-2 py-1 text-xs font-bold text-primary">AI 回复</span>
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{item.answer}</p>
                {item.context ? (
                  <p className="mt-3 rounded-md bg-slate-50 p-3 text-xs leading-6 text-slate-500">背景：{item.context}</p>
                ) : null}
                {canManageQueries ? <AssistantQueryDeleteButton item={item} /> : null}
              </div>
            ))}
            {visibleQueries.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无问答记录。</p> : null}
          </CardContent>
        </Card>

        <aside className="min-w-0 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Sparkles className="h-5 w-5 text-primary" />
                发起问题
              </CardTitle>
            </CardHeader>
            <CardContent>
              <AssistantQueryForm actorName={currentUser.name} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Bot className="h-5 w-5 text-primary" />
                使用提示
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
              <p>可以直接问“今天有哪些待审批预约”或“某台仪器当前状态怎样”。</p>
              <p>问题和回答都会写入数据库，方便后续检索和审计。</p>
            </CardContent>
          </Card>
        </aside>
      </div>
    </AppShell>
  );
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 break-words text-2xl font-bold">{value}</p>
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
