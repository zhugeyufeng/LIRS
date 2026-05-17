import { Settings2 } from "lucide-react";
import { BusinessConfigForm } from "@/components/business-config-form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { BusinessConfig } from "@/lib/api";

export function BusinessConfigList({
  createTitle,
  description,
  items,
  path,
  title,
}: {
  createTitle: string;
  description: string;
  items: BusinessConfig[];
  path: string;
  title: string;
}) {
  const activeCount = items.filter((item) => item.status === "active").length;
  return (
    <>
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Metric label="配置总数" value={items.length} />
        <Metric label="启用配置" value={activeCount} />
        <Metric label="最近更新" value={items[0] ? formatDateTime(items[0].updatedAt) : "暂无"} />
      </div>
      <Card>
        <CardHeader className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle className="flex items-center gap-2">
            <Settings2 className="h-5 w-5 text-primary" aria-hidden="true" />
            {title}
          </CardTitle>
          <BusinessConfigForm path={path} title={createTitle} />
        </CardHeader>
        <CardContent>
          <p className="mb-4 text-sm leading-6 text-muted-foreground">{description}</p>
          <div className="grid gap-3">
            {items.map((item) => (
              <article className="rounded-lg border bg-white p-4" key={item.id}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-start">
                  <div className="min-w-0">
                    <h2 className="break-words font-bold text-slate-900">{item.title}</h2>
                    <p className="mt-1 break-words text-xs text-slate-500">
                      {item.category || "未分类"} / {item.scope || "全机构"} / 更新人：{item.updatedBy || "system"}
                    </p>
                  </div>
                  <StatusBadge status={item.status} />
                </div>
                <p className="mt-3 break-words text-sm leading-6 text-slate-600">{item.description || "暂无说明。"}</p>
                <details className="mt-3 rounded-md bg-slate-50 p-3 text-xs">
                  <summary className="cursor-pointer font-bold text-slate-700">配置 JSON</summary>
                  <pre className="mt-2 overflow-x-auto whitespace-pre-wrap break-words font-mono text-slate-600">{formatConfigJson(item.configJson)}</pre>
                </details>
                <div className="mt-4 flex justify-end">
                  <BusinessConfigForm item={item} path={path} title={createTitle} />
                </div>
              </article>
            ))}
            {items.length === 0 ? <p className="rounded-lg border border-dashed p-4 text-sm text-slate-500">暂无配置，请新增第一条。</p> : null}
          </div>
        </CardContent>
      </Card>
    </>
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

function StatusBadge({ status }: { status: string }) {
  const classes: Record<string, string> = {
    active: "bg-emerald-50 text-emerald-700",
    draft: "bg-amber-50 text-amber-700",
    disabled: "bg-slate-100 text-slate-600",
    archived: "bg-blue-50 text-blue-700",
  };
  const labels: Record<string, string> = {
    active: "启用",
    draft: "草稿",
    disabled: "停用",
    archived: "归档",
  };
  return <span className={`inline-flex w-fit rounded-full px-2 py-1 text-xs font-bold ${classes[status] ?? "bg-slate-100 text-slate-600"}`}>{labels[status] ?? status}</span>;
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

function formatConfigJson(value: string) {
  try {
    return JSON.stringify(JSON.parse(value || "{}"), null, 2);
  } catch {
    return value || "{}";
  }
}
