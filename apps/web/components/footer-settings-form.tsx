"use client";

import { type ChangeEvent, type Dispatch, type SetStateAction, FormEvent, startTransition, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Pencil, Plus, Save, Trash2 } from "lucide-react";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";
import {
  browserPatch,
  createDefaultFooterSettings,
  FooterSection,
  FooterSettings,
  FooterSettingsPayload,
} from "@/lib/api";

type SectionDraft = {
  id: string;
  title: string;
  lines: string;
};

export function FooterSettingsForm({ settings }: { settings: FooterSettings }) {
  const router = useRouter();
  const [sections, setSections] = useState<SectionDraft[]>(() => toDrafts(settings.sections));
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    setSections(toDrafts(settings.sections));
  }, [settings.key, settings.sections, settings.updatedAt]);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");

    const form = new FormData(event.currentTarget);
    const payload: FooterSettingsPayload = {
      brandName: String(form.get("brandName") ?? ""),
      brandTagline: String(form.get("brandTagline") ?? ""),
      baseUrl: String(form.get("baseUrl") ?? ""),
      description: String(form.get("description") ?? ""),
      copyright: String(form.get("copyright") ?? ""),
      sections: sections
        .map((section) => ({
          title: section.title.trim(),
          lines: section.lines
            .split("\n")
            .map((line) => line.trim())
            .filter((line) => line !== ""),
        }))
        .filter((section) => section.title !== "" || section.lines.length > 0),
    };

    if (payload.sections.length === 0) {
      payload.sections = createDefaultFooterSettings().sections;
    }

    try {
      const updated = await browserPatch<FooterSettings>("/api/footer-settings", payload);
      setSections(toDrafts(updated.sections));
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
              <p className="text-sm text-slate-500">当前 Footer</p>
              <h2 className="mt-1 break-words text-lg font-bold text-slate-900">{settings.brandName}</h2>
              <p className="mt-1 break-words text-sm text-slate-600">{settings.brandTagline}</p>
            </div>
            <p className="max-w-3xl break-words text-sm leading-6 text-slate-600">{settings.description}</p>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <Summary label="网站域名" value={settings.baseUrl || "未设置"} />
              <Summary label="栏目数" value={`${settings.sections.length || createDefaultFooterSettings().sections.length} 个`} />
              <Summary label="版权信息" value={settings.copyright} />
              <Summary
                label="最后更新"
                value={settings.updatedAt ? `${formatDateTime(settings.updatedAt)}${settings.updatedBy ? ` · ${settings.updatedBy}` : ""}` : "尚未保存自定义内容"}
              />
            </div>
          </div>
          <AdminDialog
            description="修改后会同步到全站底部版权、简介和栏目内容。"
            maxWidth="max-w-4xl"
            title="修改 Footer 页面"
            trigger={
              <Button className="w-full lg:w-auto" variant="outline">
                <Pencil className="h-4 w-4" aria-hidden="true" />
                修改
              </Button>
            }
          >
            {(close) => (
              <form className="space-y-6" onSubmit={(event) => submit(event, close)}>
                <div className="grid gap-4 xl:grid-cols-2">
                  <Field defaultValue={settings.brandName} label="品牌名称" name="brandName" />
                  <Field defaultValue={settings.brandTagline} label="品牌副标题" name="brandTagline" />
                </div>
                <Field
                  defaultValue={settings.baseUrl}
                  hint="用于生成资源详情二维码，例如 https://lirs.example.cn。留空时生成站内相对路径。"
                  label="网站域名"
                  name="baseUrl"
                  placeholder="https://lirs.example.cn"
                  required={false}
                />
                <label className="block space-y-2">
                  <span className="text-sm font-medium">简介文案</span>
                  <textarea
                    className="min-h-28 w-full rounded-md border bg-white px-3 py-2 text-sm outline-none ring-0 transition focus:border-primary/40 focus:ring-2 focus:ring-primary/20"
                    defaultValue={settings.description}
                    name="description"
                    required
                  />
                  <FieldHint value={`当前：${formatCurrent(settings.description)}`} />
                </label>

                <div className="space-y-3">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium">底部栏目</p>
                      <p className="mt-1 text-xs text-muted-foreground">每个栏目支持多行内容，保存后会立即同步到全站 footer。</p>
                    </div>
                    <Button className="w-full sm:w-auto" onClick={() => setSections((current) => [...current, createSectionDraft()])} type="button" variant="outline">
                      <Plus className="h-4 w-4" aria-hidden="true" />
                      新增栏目
                    </Button>
                  </div>

                  <div className="space-y-3">
                    {sections.map((section, index) => (
                      <div className="space-y-3 rounded-lg border bg-slate-50/40 p-4" key={section.id}>
                        <div className="flex items-center justify-between gap-3">
                          <p className="text-sm font-medium">栏目 {index + 1}</p>
                          <Button
                            disabled={sections.length <= 1}
                            onClick={() => setSections((current) => current.filter((item) => item.id !== section.id))}
                            size="icon"
                            type="button"
                            variant="ghost"
                          >
                            <Trash2 className="h-4 w-4" aria-hidden="true" />
                          </Button>
                        </div>
                        <div className="grid gap-3">
                          <Field
                            hint={`当前栏目 ${index + 1} 标题：${formatCurrent(section.title)}`}
                            label="栏目标题"
                            name={`section-title-${section.id}`}
                            onChange={(event) => updateSection(setSections, section.id, { title: event.target.value })}
                            value={section.title}
                          />
                          <label className="block space-y-2">
                            <span className="text-sm font-medium">栏目内容</span>
                            <textarea
                              className="min-h-24 w-full rounded-md border bg-white px-3 py-2 text-sm outline-none ring-0 transition focus:border-primary/40 focus:ring-2 focus:ring-primary/20"
                              onChange={(event) => updateSection(setSections, section.id, { lines: event.target.value })}
                              placeholder="每行一条内容"
                              value={section.lines}
                            />
                            <FieldHint value={`当前栏目 ${index + 1} 内容：${formatCurrent(section.lines)}`} />
                          </label>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="grid gap-4 xl:grid-cols-[1fr_auto]">
                  <Field defaultValue={settings.copyright} label="版权信息" name="copyright" />
                  <div className="flex items-end">
                    <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                      <Save className="h-4 w-4" aria-hidden="true" />
                      {pending ? "保存中..." : "保存 footer"}
                    </Button>
                  </div>
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
  defaultValue,
  hint,
  label,
  name,
  onChange,
  placeholder,
  required = true,
  value,
}: {
  defaultValue?: string;
  hint?: string;
  label: string;
  name: string;
  onChange?: (event: ChangeEvent<HTMLInputElement>) => void;
  placeholder?: string;
  required?: boolean;
  value?: string;
}) {
  const controlled = value !== undefined;
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium">{label}</span>
      <input
        className="h-10 w-full rounded-md border bg-white px-3 text-sm outline-none ring-0 transition focus:border-primary/40 focus:ring-2 focus:ring-primary/20"
        name={name}
        placeholder={placeholder}
        required={required}
        {...(controlled
          ? {
              onChange: onChange ?? (() => undefined),
              value,
            }
          : { defaultValue })}
      />
      <FieldHint value={hint ?? `当前：${formatCurrent(controlled ? value : defaultValue)}`} />
    </label>
  );
}

function FieldHint({ value }: { value: string }) {
  return <span className="block break-words text-xs text-slate-500">{value}</span>;
}

function formatCurrent(value?: string) {
  const text = (value ?? "").trim();
  return text === "" ? "未设置" : text;
}

function createSectionDraft(section?: FooterSection, index?: number): SectionDraft {
  return {
    id: index === undefined ? makeSectionId() : `footer-section-${index}`,
    title: section?.title ?? "",
    lines: section?.lines.join("\n") ?? "",
  };
}

function toDrafts(sections: FooterSection[]) {
  if (sections.length === 0) {
    return [createSectionDraft()];
  }
  return sections.map((section, index) => createSectionDraft(section, index));
}

function updateSection(
  setSections: Dispatch<SetStateAction<SectionDraft[]>>,
  id: string,
  patch: Partial<SectionDraft>,
) {
  setSections((current) => current.map((section) => (section.id === id ? { ...section, ...patch } : section)));
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

function makeSectionId() {
  if (typeof globalThis.crypto !== "undefined" && typeof globalThis.crypto.randomUUID === "function") {
    return globalThis.crypto.randomUUID();
  }
  return `footer-section-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
