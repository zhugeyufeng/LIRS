"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { Edit3, PlusCircle, Save } from "lucide-react";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";
import { browserPatch, browserPost, type BusinessConfig, type BusinessConfigPayload } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";

export function BusinessConfigForm({
  item,
  path,
  title,
}: {
  item?: BusinessConfig;
  path: string;
  title: string;
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (item && !confirmTwice(`确定修改${title}“${item.title}”吗？`, "请再次确认。配置修改后相关业务规则会立即按新配置读取。")) {
      return;
    }
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const configJson = String(form.get("configJson") ?? "{}").trim() || "{}";
    try {
      JSON.parse(configJson);
    } catch {
      setPending(false);
      setMessage("配置 JSON 格式不正确");
      return;
    }
    const payload: BusinessConfigPayload = {
      title: String(form.get("title") ?? "").trim(),
      category: String(form.get("category") ?? "").trim(),
      scope: String(form.get("scope") ?? "").trim(),
      status: String(form.get("status") ?? "active"),
      description: String(form.get("description") ?? "").trim(),
      configJson,
    };
    try {
      if (item) {
        await browserPatch<BusinessConfig>(`${path}/${item.id}`, payload);
      } else {
        await browserPost<BusinessConfig>(path, payload);
        formElement.reset();
      }
      setMessage("配置已保存");
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <AdminDialog
      description={item ? `正在修改：${item.title}` : "新增配置会写入当前机构数据库，并按租户隔离。"}
      title={item ? `修改${title}` : `新增${title}`}
      trigger={
        <Button className={item ? "w-full sm:w-auto" : ""} size={item ? "sm" : "default"} variant={item ? "outline" : "default"}>
          {item ? <Edit3 className="h-4 w-4" aria-hidden="true" /> : <PlusCircle className="h-4 w-4" aria-hidden="true" />}
          {item ? "修改" : "新增"}
        </Button>
      }
    >
      <form className="space-y-4" onSubmit={submit}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field defaultValue={item?.title ?? ""} label="配置名称" name="title" placeholder="例如：预约审批默认流程" required />
          <Field defaultValue={item?.category ?? ""} label="分类" name="category" placeholder="例如：预约 / 申领 / 仪器计费" />
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <Field defaultValue={item?.scope ?? ""} label="适用范围" name="scope" placeholder="例如：全机构 / 指定仪器 / 高值耗材" />
          <label className="grid gap-1 text-sm">
            <span className="font-medium text-slate-700">状态</span>
            <select className="h-10 rounded-md border bg-white px-3 text-sm" defaultValue={item?.status ?? "active"} name="status">
              <option value="active">启用</option>
              <option value="draft">草稿</option>
              <option value="disabled">停用</option>
              <option value="archived">归档</option>
            </select>
          </label>
        </div>
        <label className="grid gap-1 text-sm">
          <span className="font-medium text-slate-700">说明</span>
          <textarea
            className="min-h-24 rounded-md border bg-white px-3 py-2 text-sm leading-6"
            defaultValue={item?.description ?? ""}
            name="description"
            placeholder="说明这条配置影响的业务、审批节点或计费口径"
          />
        </label>
        <label className="grid gap-1 text-sm">
          <span className="font-medium text-slate-700">配置 JSON</span>
          <textarea
            className="min-h-32 rounded-md border bg-white px-3 py-2 font-mono text-xs leading-6"
            defaultValue={item?.configJson ?? "{}"}
            name="configJson"
            placeholder='{"steps":["课题组负责人","仪器管理员"],"timeoutHours":24}'
          />
        </label>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-xs text-muted-foreground">{message || "修改会记录到审计日志，历史业务记录不会被前端删除。"}</p>
          <Button disabled={pending} type="submit">
            <Save className="h-4 w-4" aria-hidden="true" />
            {pending ? "保存中..." : "保存"}
          </Button>
        </div>
      </form>
    </AdminDialog>
  );
}

function Field({
  defaultValue,
  label,
  name,
  placeholder,
  required = false,
}: {
  defaultValue: string;
  label: string;
  name: string;
  placeholder: string;
  required?: boolean;
}) {
  return (
    <label className="grid gap-1 text-sm">
      <span className="font-medium text-slate-700">{label}</span>
      <input
        className="h-10 rounded-md border bg-white px-3 text-sm"
        defaultValue={defaultValue}
        name={name}
        placeholder={placeholder}
        required={required}
      />
    </label>
  );
}
