"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { Pencil, Save, Trash2 } from "lucide-react";
import { browserDelete, browserPatch, browserPost, ProcurementProject, ProcurementProjectPayload } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { AdminDialog } from "@/components/admin-dialog";
import { Field, procurementProjectExpired } from "@/components/material-purchase-shared";
import { Button } from "@/components/ui/button";

export function ProcurementProjectManager({ projects }: { projects: ProcurementProject[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void, project?: ProcurementProject) {
    event.preventDefault();
    const pendingKey = project ? `project:${project.id}` : "project:new";
    if (project && !confirmTwice(`确定修改采购项目“${project.name}”吗？`, "请再次确认。有效期和状态会影响后续申购选择。")) {
      return;
    }
    setPending(pendingKey);
    setMessage("");
    const form = new FormData(event.currentTarget);
    const payload: ProcurementProjectPayload = {
      name: String(form.get("name") ?? ""),
      expiresAt: String(form.get("expiresAt") ?? ""),
      status: String(form.get("status") ?? "active"),
    };
    try {
      if (project) {
        await browserPatch<ProcurementProject>(`/api/procurement-projects/${project.id}`, payload);
      } else {
        await browserPost<ProcurementProject>("/api/procurement-projects", payload);
      }
      setMessage(project ? "采购项目已修改。" : "采购项目已新增。");
      close?.();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending("");
    }
  }

  async function remove(id: string) {
    const project = projects.find((item) => item.id === id);
    if (!confirmTwice(`确定停用采购项目“${project?.name ?? id}”吗？`, "请再次确认。停用后关联物资不能继续申购。")) {
      return;
    }
    setPending(`project-delete:${id}`);
    setMessage("");
    try {
      await browserDelete<ProcurementProject>(`/api/procurement-projects/${id}`);
      setMessage("采购项目已停用。");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "停用失败");
    } finally {
      setPending("");
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
        <AdminDialog
          description="有效期为空表示长期有效；超过有效期后，关联物资不能再被申购。"
          maxWidth="max-w-3xl"
          title="新增采购项目"
          trigger={
            <Button className="w-full sm:w-auto" type="button">
              <Save className="h-4 w-4" aria-hidden="true" />
              新增采购项目
            </Button>
          }
        >
          {(close) => (
            <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
              <ProcurementProjectFields />
              <div className="flex justify-end">
                <Button disabled={pending === "project:new"} type="submit">
                  {pending === "project:new" ? "保存中..." : "保存"}
                </Button>
              </div>
            </form>
          )}
        </AdminDialog>
      </div>
      <div className="overflow-x-auto rounded-lg border">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="bg-slate-50 text-slate-500">
            <tr>
              <th className="p-3">采购项目名称及编号</th>
              <th className="p-3">有效期</th>
              <th className="p-3">状态</th>
              <th className="w-44 p-3">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {projects.map((project) => (
              <tr key={project.id}>
                <td className="break-words p-3 align-top">{project.name}</td>
                <td className="p-3 align-top">{project.expiresAt || "长期有效"}</td>
                <td className="p-3 align-top">
                  <span className={`rounded px-2 py-1 text-xs font-bold ${project.status !== "active" || procurementProjectExpired(project) ? "bg-amber-100 text-amber-800" : "bg-emerald-100 text-emerald-800"}`}>
                    {project.status !== "active" ? "已停用" : procurementProjectExpired(project) ? "已过期" : "可申购"}
                  </span>
                </td>
                <td className="p-3 align-top">
                  <div className="flex flex-wrap gap-2">
                    <AdminDialog
                      description="有效期为空表示长期有效；超过有效期后，关联物资不能再被申购。"
                      maxWidth="max-w-3xl"
                      title="修改采购项目"
                      trigger={
                        <Button disabled={pending === `project:${project.id}`} size="sm" type="button" variant="ghost">
                          <Pencil className="h-4 w-4" aria-hidden="true" />
                          修改
                        </Button>
                      }
                    >
                      {(close) => (
                        <form className="space-y-4" onSubmit={(event) => submit(event, close, project)}>
                          <ProcurementProjectFields project={project} />
                          <div className="flex justify-end">
                            <Button disabled={pending === `project:${project.id}`} type="submit">
                              {pending === `project:${project.id}` ? "保存中..." : "保存修改"}
                            </Button>
                          </div>
                        </form>
                      )}
                    </AdminDialog>
                    <Button disabled={pending === `project-delete:${project.id}`} onClick={() => remove(project.id)} size="sm" type="button" variant="ghost">
                      <Trash2 className="h-4 w-4" aria-hidden="true" />
                      停用
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {projects.length === 0 ? <p className="rounded-lg border p-4 text-sm text-slate-500">暂无采购项目。</p> : null}
      {message ? <p className="text-sm text-slate-500">{message}</p> : null}
    </div>
  );
}

function ProcurementProjectFields({ project }: { project?: ProcurementProject }) {
  return (
    <div className="grid gap-3 md:grid-cols-2">
      <Field className="md:col-span-2" defaultValue={project?.name} label="采购项目名称及编号*" name="name" required />
      <Field defaultValue={project?.expiresAt} label="有效期" name="expiresAt" type="date" />
      <label className="block space-y-2">
        <span className="text-sm font-medium">状态</span>
        <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={project?.status ?? "active"} name="status">
          <option value="active">启用</option>
          <option value="disabled">停用</option>
        </select>
      </label>
    </div>
  );
}
