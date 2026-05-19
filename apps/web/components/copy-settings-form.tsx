"use client";

import { type ChangeEvent, type Dispatch, type FormEvent, type SetStateAction, startTransition, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Pencil, Plus, RotateCcw, Save, Trash2 } from "lucide-react";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";
import {
  browserPatch,
  createDefaultCopySettings,
  type CopyEntry,
  type CopySettings,
  type CopySettingsPayload,
} from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";

type CopyEntryDraft = CopyEntry & {
  id: string;
};

export function CopySettingsForm({ settings }: { settings: CopySettings }) {
  const router = useRouter();
  const [entries, setEntries] = useState<CopyEntryDraft[]>(() => toDrafts(settings.entries));
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const scopeSummary = useMemo(() => summarizeScopes(entries), [entries]);

  useEffect(() => {
    setEntries(toDrafts(settings.entries));
  }, [settings.key, settings.entries, settings.updatedAt]);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    if (!confirmTwice("确定修改文案中心吗？", "请再次确认。保存后已接入的导航、按钮和标题文案会立即更新。")) {
      return;
    }
    setPending(true);
    setMessage("");

    const payload: CopySettingsPayload = {
      entries: entries
        .map((entry) => ({
          key: entry.key.trim(),
          label: entry.label.trim(),
          value: entry.value.trim(),
          scope: entry.scope.trim() || "custom",
          description: entry.description.trim(),
        }))
        .filter((entry) => entry.key !== ""),
    };

    if (payload.entries.length === 0) {
      payload.entries = createDefaultCopySettings().entries;
    }

    try {
      const updated = await browserPatch<CopySettings>("/api/copy-settings", payload);
      setEntries(toDrafts(updated.entries));
      setMessage(`已保存，更新时间 ${formatDateTime(updated.updatedAt)}`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-5">
      <div className="rounded-lg border bg-white p-4">
        <div className="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div className="min-w-0 space-y-4">
            <div>
              <p className="text-sm text-slate-500">当前文案配置</p>
              <h2 className="mt-1 break-words text-lg font-bold text-slate-900">文案中心</h2>
              <p className="mt-1 break-words text-sm text-slate-600">顶部导航、按钮、标题、占位符和首页入口文字会从这里读取。</p>
            </div>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <Summary label="条目数" value={`${entries.length} 项`} />
              <Summary label="适用范围" value={scopeSummary || "未设置"} />
              <Summary
                label="最后更新"
                value={settings.updatedAt ? `${formatDateTime(settings.updatedAt)}${settings.updatedBy ? ` · ${settings.updatedBy}` : ""}` : "尚未保存自定义内容"}
              />
              <Summary label="默认条目" value={`${createDefaultCopySettings().entries.length} 项`} />
            </div>
          </div>
          <AdminDialog
            description="修改后会同步到前台导航、首页入口和已接入的按钮/标题文案。自定义条目的标识需要与代码读取的文案标识一致。"
            maxWidth="max-w-6xl"
            title="修改文案中心"
            trigger={
              <Button className="w-full lg:w-auto" variant="outline">
                <Pencil className="h-4 w-4" aria-hidden="true" />
                修改
              </Button>
            }
          >
            {(close) => (
              <form className="space-y-5" onSubmit={(event) => submit(event, close)}>
                <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
                  <div className="min-w-0">
                    <p className="text-sm font-medium">文案条目</p>
                    <p className="mt-1 text-xs text-muted-foreground">条目标识用于前端读取，显示文案是实际展示给用户的文字。</p>
                  </div>
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Button className="w-full sm:w-auto" onClick={() => setEntries((current) => [...current, createEntryDraft()])} type="button" variant="outline">
                      <Plus className="h-4 w-4" aria-hidden="true" />
                      新增条目
                    </Button>
                    <Button className="w-full sm:w-auto" onClick={() => setEntries(toDrafts(createDefaultCopySettings().entries))} type="button" variant="outline">
                      <RotateCcw className="h-4 w-4" aria-hidden="true" />
                      恢复默认
                    </Button>
                  </div>
                </div>

                <div className="space-y-3">
                  {entries.map((entry, index) => (
                    <div className="space-y-3 rounded-lg border bg-slate-50/40 p-4" key={entry.id}>
                      <div className="flex flex-wrap items-center justify-between gap-3">
                        <div className="min-w-0">
                          <p className="text-sm font-medium">条目 {index + 1}</p>
                          <p className="mt-1 break-words text-xs text-muted-foreground">正在修改：{formatCurrent(entry.key || entry.label || entry.value)}</p>
                        </div>
                        <Button
                          disabled={entries.length <= 1}
                          onClick={() => {
                            if (confirmTwice(`确定删除文案条目“${entry.key || entry.label || entry.value}”吗？`, "请再次确认。删除后该条目会在保存文案时移除。")) {
                              setEntries((current) => current.filter((item) => item.id !== entry.id));
                            }
                          }}
                          size="icon"
                          type="button"
                          variant="ghost"
                        >
                          <Trash2 className="h-4 w-4" aria-hidden="true" />
                          <span className="sr-only">删除条目</span>
                        </Button>
                      </div>
                      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
                        <Field
                          label="条目标识"
                          onChange={(event) => updateEntry(setEntries, entry.id, { key: event.target.value })}
                          required
                          value={entry.key}
                          hint={`当前修改的条目标识：${formatCurrent(entry.key)}`}
                        />
                        <Field
                          label="后台标签"
                          onChange={(event) => updateEntry(setEntries, entry.id, { label: event.target.value })}
                          value={entry.label}
                          hint={`后台展示名称：${formatCurrent(entry.label)}`}
                        />
                        <Field
                          label="显示文案"
                          onChange={(event) => updateEntry(setEntries, entry.id, { value: event.target.value })}
                          required
                          value={entry.value}
                          hint={`前台当前显示：${formatCurrent(entry.value)}`}
                        />
                        <Field
                          label="适用范围"
                          onChange={(event) => updateEntry(setEntries, entry.id, { scope: event.target.value })}
                          value={entry.scope}
                          hint={`范围示例：nav / button / page / module / placeholder / brand`}
                        />
                        <Field
                          label="说明"
                          onChange={(event) => updateEntry(setEntries, entry.id, { description: event.target.value })}
                          value={entry.description}
                          hint={`说明该条目影响的位置：${formatCurrent(entry.description)}`}
                        />
                      </div>
                    </div>
                  ))}
                </div>

                <div className="flex justify-end">
                  <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                    <Save className="h-4 w-4" aria-hidden="true" />
                    {pending ? "保存中..." : "保存文案"}
                  </Button>
                </div>
              </form>
            )}
          </AdminDialog>
        </div>
      </div>
      {message ? <p className="rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-700">{message}</p> : null}
    </div>
  );
}

function Summary({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-md bg-slate-50 px-3 py-2">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value}</p>
    </div>
  );
}

function Field({
  hint,
  label,
  onChange,
  required,
  value,
}: {
  hint: string;
  label: string;
  onChange: (event: ChangeEvent<HTMLInputElement>) => void;
  required?: boolean;
  value: string;
}) {
  return (
    <label className="block min-w-0 space-y-2">
      <span className="text-sm font-medium">{label}</span>
      <input
        className="h-10 w-full rounded-md border bg-white px-3 text-sm outline-none ring-0 transition focus:border-primary/40 focus:ring-2 focus:ring-primary/20"
        onChange={onChange}
        required={required}
        value={value}
      />
      <span className="block break-words text-xs text-slate-500">{hint}</span>
    </label>
  );
}

function updateEntry(
  setEntries: Dispatch<SetStateAction<CopyEntryDraft[]>>,
  id: string,
  patch: Partial<CopyEntryDraft>,
) {
  setEntries((current) => current.map((entry) => (entry.id === id ? { ...entry, ...patch } : entry)));
}

function toDrafts(entries: CopyEntry[]) {
  const source = entries.length > 0 ? entries : createDefaultCopySettings().entries;
  const seen = new Set<string>();
  const drafts: CopyEntryDraft[] = [];
  source.forEach((entry, index) => {
    const key = entry.key.trim();
    if (!key || seen.has(key)) {
      return;
    }
    seen.add(key);
    drafts.push(createEntryDraft(entry, index));
  });
  return drafts.length > 0 ? drafts : [createEntryDraft()];
}

function createEntryDraft(entry?: CopyEntry, index?: number): CopyEntryDraft {
  return {
    id: index === undefined ? makeEntryId() : `copy-entry-${index}-${entry?.key ?? ""}`,
    key: entry?.key ?? "",
    label: entry?.label ?? "",
    value: entry?.value ?? "",
    scope: entry?.scope ?? "custom",
    description: entry?.description ?? "",
  };
}

function summarizeScopes(entries: CopyEntryDraft[]) {
  const counts = new Map<string, number>();
  entries.forEach((entry) => {
    const scope = entry.scope.trim() || "custom";
    counts.set(scope, (counts.get(scope) ?? 0) + 1);
  });
  return Array.from(counts.entries())
    .sort(([left], [right]) => left.localeCompare(right, "zh-CN"))
    .slice(0, 6)
    .map(([scope, count]) => `${scope} ${count}`)
    .join(" / ");
}

function formatCurrent(value?: string) {
  const text = (value ?? "").trim();
  return text === "" ? "未设置" : text;
}

function formatDateTime(value: string) {
  if (!value) {
    return "未保存";
  }
  return new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function makeEntryId() {
  if (typeof globalThis.crypto !== "undefined" && typeof globalThis.crypto.randomUUID === "function") {
    return globalThis.crypto.randomUUID();
  }
  return `copy-entry-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
