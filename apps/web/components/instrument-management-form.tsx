"use client";

import { FormEvent, startTransition, useState } from "react";
import { useRouter } from "next/navigation";
import { Pencil, Plus, Save, Trash2, X } from "lucide-react";
import { browserDelete, browserPatch, browserPost, Instrument, InstrumentPayload, OrganizationUnit } from "@/lib/api";
import { confirmTwice } from "@/lib/confirm";
import { formatServiceWindow } from "@/lib/instrument-rules";
import { instrumentStatusLabel } from "@/lib/status-labels";
import { AdminDialog } from "@/components/admin-dialog";
import { Button } from "@/components/ui/button";

export function InstrumentCreateForm({ departments, groups }: { departments: string[]; groups: OrganizationUnit[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const payload = instrumentPayload(new FormData(formElement));
    try {
      const instrument = await browserPost<Instrument>("/api/instruments", payload);
      setMessage(`已创建仪器：${instrument.name}`);
      formElement.reset();
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
    <div className="space-y-3">
      <AdminDialog
        description="新增仪器后会出现在仪器预约模块，预约窗口和时段长度会同步用于详情页可预约时间块。"
        maxWidth="max-w-5xl"
        title="新增仪器"
        trigger={
          <Button className="w-full">
            <Plus className="h-4 w-4" aria-hidden="true" />
            新增仪器
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <InstrumentFields departments={departments} groups={groups} />
            <div className="flex justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                <Save className="h-4 w-4" aria-hidden="true" />
                {pending ? "保存中..." : "新增仪器"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      {message ? <p className="text-sm text-slate-500">{message}</p> : null}
    </div>
  );
}

export function InstrumentStatusForm({
  departments = [],
  groups = [],
  instrument,
}: {
  departments?: string[];
  groups?: OrganizationUnit[];
  instrument: Instrument;
}) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>, close?: () => void) {
    event.preventDefault();
    if (!confirmTwice(`确定修改仪器“${instrument.name}”吗？`, "请再次确认。仪器信息修改后列表和预约入口会立即更新。")) {
      return;
    }
    setPending(true);
    setMessage("");
    const payload: InstrumentPayload = {
      ...instrumentPayload(new FormData(event.currentTarget)),
      maintenanceSummary: instrument.maintenanceSummary,
    };
    try {
      const updated = await browserPatch<Instrument>(`/api/instruments/${instrument.id}`, payload);
      setMessage(`已更新：${updated.name} / ${instrumentStatusLabel(updated.status)}`);
      close?.();
      startTransition(() => {
        router.refresh();
      });
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "更新失败");
    } finally {
      setPending(false);
    }
  }

  async function deleteInstrument(close?: () => void) {
    if (!confirmTwice(`确定删除“${instrument.name}”吗？删除后该仪器会立即从仪器列表、预约入口和维护选择中移除。`, "请再次确认删除仪器。关联的未完成预约会取消，历史预约和维护记录会保留为已删除仪器。")) {
      return;
    }
    setPending(true);
    setMessage("");
    try {
      const deleted = await browserDelete<Instrument>(`/api/instruments/${instrument.id}`);
      setMessage(`已删除：${deleted.name}`);
      close?.();
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "删除失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="mt-3 flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
      <AdminDialog
        description="修改仪器档案、状态、费率、部门、团队和预约规则。"
        maxWidth="max-w-5xl"
        title={`修改仪器：${instrument.name}`}
        trigger={
          <Button className="w-full sm:w-auto" disabled={pending} size="sm" variant="outline">
            <Pencil className="h-4 w-4" aria-hidden="true" />
            修改
          </Button>
        }
      >
        {(close) => (
          <form className="space-y-4" onSubmit={(event) => submit(event, close)}>
            <InstrumentFields departments={departments} groups={groups} instrument={instrument} />
            <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
              <Button className="w-full sm:w-auto" disabled={pending} onClick={close} type="button" variant="outline">
                <X className="h-4 w-4" aria-hidden="true" />
                取消
              </Button>
              <Button className="w-full sm:w-auto" disabled={pending} type="submit">
                <Save className="h-4 w-4" aria-hidden="true" />
                {pending ? "保存中..." : "保存修改"}
              </Button>
            </div>
          </form>
        )}
      </AdminDialog>
      <Button className="w-full sm:w-auto" disabled={pending} onClick={() => deleteInstrument()} size="sm" type="button" variant="destructive">
        <Trash2 className="h-4 w-4" aria-hidden="true" />
        删除仪器
      </Button>
      {message ? <span className="text-xs text-slate-500">{message}</span> : null}
    </div>
  );
}

function InstrumentFields({
  departments,
  groups,
  instrument,
}: {
  departments: string[];
  groups: OrganizationUnit[];
  instrument?: Instrument;
}) {
  const [selectedDepartment, setSelectedDepartment] = useState(instrument?.department ?? "");
  const [selectedTeam, setSelectedTeam] = useState(instrument?.groupName ?? "");
  const departmentOptions = Array.from(new Set([instrument?.department ?? "", ...departments].filter(Boolean)));
  const teamOptions = groups.filter((group) => group.parentName === "" || group.parentName === selectedDepartment || group.name === instrument?.groupName);

  function changeDepartment(value: string) {
    setSelectedDepartment(value);
    const currentTeamStillAvailable = groups.some((group) => group.name === selectedTeam && (group.parentName === "" || group.parentName === value));
    if (!currentTeamStillAvailable) {
      setSelectedTeam("");
    }
  }

  return (
    <>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Field defaultValue={instrument?.name} label="仪器名称" name="name" placeholder="填写仪器名称" required />
        <Field defaultValue={instrument?.category} label="分类" name="category" placeholder="填写仪器分类" required />
        <label className="block min-w-0 space-y-2">
          <span className="text-sm font-medium">状态</span>
          <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" name="status" defaultValue={instrument?.status ?? "available"}>
            <option value="available">可用</option>
            <option value="busy">繁忙</option>
            <option value="maintenance">维护中</option>
            <option value="disabled">停用</option>
          </select>
          {instrument ? <FieldHint value={`当前：${instrumentStatusLabel(instrument.status)}`} /> : null}
        </label>
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <label className="block min-w-0 space-y-2">
          <span className="text-sm font-medium">所属部门</span>
          <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" name="department" onChange={(event) => changeDepartment(event.currentTarget.value)} required value={selectedDepartment}>
            <option value="">选择所属部门</option>
            {departmentOptions.map((department) => (
              <option key={department} value={department}>
                {department}
              </option>
            ))}
          </select>
          {instrument?.department ? <FieldHint value={`当前：${instrument.department}`} /> : null}
        </label>
        <label className="block min-w-0 space-y-2">
          <span className="text-sm font-medium">归属团队（可选）</span>
          <select className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm" name="groupName" onChange={(event) => setSelectedTeam(event.currentTarget.value)} value={selectedTeam}>
            <option value="">直接归属所选部门</option>
            {teamOptions.map((group) => (
              <option key={`${group.parentName}:${group.name}`} value={group.name}>
                {group.parentName ? `${group.parentName} / ${group.name}` : group.name}
              </option>
            ))}
          </select>
          {instrument?.groupName ? <FieldHint value={`当前：${instrument.groupName}`} /> : <FieldHint value="不选择团队时，该仪器直接归属到所选部门。" />}
        </label>
        <Field defaultValue={instrument?.location} label="位置" name="location" placeholder="填写放置位置" required />
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <Field defaultValue={instrument?.hourlyRate} label="小时费率" min={0} name="hourlyRate" placeholder="填写每小时费用" required step="0.01" type="number" />
        <Field defaultValue={instrument?.brand} label="品牌" name="brand" placeholder="填写品牌" />
        <Field defaultValue={instrument?.model} label="型号" name="model" placeholder="填写型号" />
        <Field defaultValue={instrument?.assetCode} label="资产编号" name="assetCode" placeholder="填写资产编号" />
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <label className="block min-w-0 space-y-2">
          <span className="text-sm font-medium">门禁联动</span>
          <span className="flex h-10 items-center gap-3 rounded-md border bg-white px-3 text-sm">
            <input className="h-4 w-4" defaultChecked={instrument?.accessControlEnabled ?? false} name="accessControlEnabled" type="checkbox" />
            启用该仪器门禁授权
          </span>
          {instrument ? <FieldHint value={`当前：${instrument.accessControlEnabled ? "启用" : "停用"}`} /> : <FieldHint value="勾选后审批通过才会按该仪器匹配门禁。" />}
        </label>
        <Field defaultValue={instrument?.accessControlGroup} label="门禁授权组" name="accessControlGroup" placeholder="填写该仪器对应的门禁授权组，空则使用全局默认" />
        <Field defaultValue={instrument?.accessControlPoint} label="门禁点位编码" name="accessControlPoint" placeholder="填写海康/大华门禁点位或门编号" />
      </div>
      <TextAreaField defaultValue={instrument?.description} label="仪器简介" name="description" placeholder="填写仪器简介" />
      <TextAreaField defaultValue={instrument?.technicalSpecs} label="技术参数" name="technicalSpecs" placeholder="填写技术参数" />
      <TextAreaField defaultValue={instrument?.bookingRule} label="预约规则" name="bookingRule" placeholder="填写预约规则" />
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Field defaultValue={instrument?.maxBookingHours} label="最长预约小时" min={1} name="maxBookingHours" placeholder="填写最长预约小时" type="number" />
        <Field defaultValue={instrument?.minAdvanceHours} label="提前预约小时" min={0} name="minAdvanceHours" placeholder="填写提前预约小时" type="number" />
        <Field defaultValue={instrument?.cancelCutoffHours} label="取消截止小时" min={0} name="cancelCutoffHours" placeholder="填写取消截止小时" type="number" />
        <Field defaultValue={instrument?.checkinWindowMinutes} label="签到窗口分钟" min={0} name="checkinWindowMinutes" placeholder="填写签到窗口分钟" type="number" />
        <Field defaultValue={instrument?.bookingWindowDays} label="预约窗口天数" min={1} name="bookingWindowDays" placeholder="填写可预约天数" type="number" />
        <Field defaultValue={instrument?.bookingIntervalHours} label="预约时段小时" max={12} min={1} name="bookingIntervalHours" placeholder="填写每段预约小时数" type="number" />
        <Field defaultValue={instrument?.serviceStartHour ?? 0} label="每日开放开始小时" max={23} min={0} name="serviceStartHour" placeholder="0 表示 00:00 开始" type="number" />
        <Field defaultValue={instrument?.serviceEndHour ?? 24} label="每日开放结束小时" max={24} min={1} name="serviceEndHour" placeholder="24 表示 24:00 结束" type="number" />
      </div>
      <p className="text-xs leading-5 text-slate-500">
        预约窗口控制可提前多少天提交，预约时段控制每段预约的小时长度，每日开放时间控制可生成的预约块。{instrument ? `当前开放 ${formatServiceWindow(instrument)}` : "默认开放 00:00-24:00。"}
      </p>
    </>
  );
}

function Field({
  defaultValue,
  max,
  min,
  label,
  name,
  placeholder,
  required = false,
  step,
  type = "text",
}: {
  defaultValue?: string | number;
  label: string;
  max?: number;
  min?: number;
  name: string;
  placeholder: string;
  required?: boolean;
  step?: string;
  type?: string;
}) {
  return (
    <label className="block min-w-0 space-y-2">
      <span className="text-sm font-medium">{label}</span>
      <input
        className="h-10 w-full min-w-0 rounded-md border bg-white px-3 text-sm"
        defaultValue={defaultValue ?? ""}
        max={max}
        min={min}
        name={name}
        placeholder={placeholder}
        required={required}
        step={step}
        type={type}
      />
      {defaultValue !== undefined ? <FieldHint value={`当前：${formatCurrent(defaultValue)}`} /> : null}
    </label>
  );
}

function TextAreaField({
  defaultValue,
  label,
  name,
  placeholder,
}: {
  defaultValue?: string;
  label: string;
  name: string;
  placeholder: string;
}) {
  return (
    <label className="block min-w-0 space-y-2">
      <span className="text-sm font-medium">{label}</span>
      <textarea className="min-h-20 w-full rounded-md border bg-white px-3 py-2 text-sm" defaultValue={defaultValue ?? ""} name={name} placeholder={placeholder} />
      {defaultValue !== undefined ? <FieldHint value={`当前：${formatCurrent(defaultValue)}`} /> : null}
    </label>
  );
}

function instrumentPayload(form: FormData): InstrumentPayload {
  return {
    name: String(form.get("name") ?? ""),
    category: String(form.get("category") ?? ""),
    department: String(form.get("department") ?? ""),
    groupName: String(form.get("groupName") ?? ""),
    status: String(form.get("status") ?? "available"),
    location: String(form.get("location") ?? ""),
    hourlyRate: Number(form.get("hourlyRate") ?? 0),
    brand: String(form.get("brand") ?? ""),
    model: String(form.get("model") ?? ""),
    assetCode: String(form.get("assetCode") ?? ""),
    accessControlEnabled: form.get("accessControlEnabled") === "on",
    accessControlGroup: String(form.get("accessControlGroup") ?? ""),
    accessControlPoint: String(form.get("accessControlPoint") ?? ""),
    description: String(form.get("description") ?? ""),
    technicalSpecs: String(form.get("technicalSpecs") ?? ""),
    bookingRule: String(form.get("bookingRule") ?? ""),
    maintenanceSummary: "",
    maxBookingHours: Number(form.get("maxBookingHours") || 72),
    minAdvanceHours: Number(form.get("minAdvanceHours") || 2),
    cancelCutoffHours: Number(form.get("cancelCutoffHours") || 2),
    checkinWindowMinutes: Number(form.get("checkinWindowMinutes") || 30),
    bookingWindowDays: Number(form.get("bookingWindowDays") || 30),
    bookingIntervalHours: Number(form.get("bookingIntervalHours") || 1),
    serviceStartHour: Number(form.get("serviceStartHour") || 0),
    serviceEndHour: Number(form.get("serviceEndHour") || 24),
  };
}

function FieldHint({ value }: { value: string }) {
  return <span className="block break-words text-xs text-slate-500">{value}</span>;
}

function formatCurrent(value: string | number) {
  const text = String(value).trim();
  return text === "" ? "未设置" : text;
}
