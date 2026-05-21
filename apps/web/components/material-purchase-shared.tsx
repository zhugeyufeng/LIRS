"use client";

import { Download } from "lucide-react";
import { ProcurementProject, PurchasableMaterial } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function Field({ className = "", defaultValue = "", label, list, name, required = false, step, type = "text" }: { className?: string; defaultValue?: string | number; label: string; list?: string; name: string; required?: boolean; step?: string; type?: string }) {
  return (
    <label className={`block space-y-2 ${className}`}>
      <span className="text-sm font-medium">{label}</span>
      <input className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={defaultValue} list={list} min={type === "number" ? 0 : undefined} name={name} required={required} step={step} type={type} />
    </label>
  );
}

export function Info({ className = "", label, value }: { className?: string; label: string; value: string }) {
  return (
    <div className={`min-w-0 ${className}`}>
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-800">{value}</p>
    </div>
  );
}

export function formatMoney(value: number) {
  return `¥${value.toFixed(2)}`;
}

export function purchasableMaterialOptionLabel(item: PurchasableMaterial) {
  return `${item.idNo} ${item.projectName} ${item.brand} ${item.spec}`;
}

export function purchasableMaterialExpired(item: PurchasableMaterial) {
  return item.procurementProjectStatus === "disabled" || dateExpired(item.procurementExpiresAt);
}

export function procurementProjectExpired(project: ProcurementProject) {
  return dateExpired(project.expiresAt);
}

export function purchasableMaterialSearchText(item: PurchasableMaterial) {
  return [
    item.idNo,
    item.sequenceNo,
    item.procurementProject,
    item.projectName,
    item.brand,
    item.spec,
    item.unit,
    item.remark,
    item.technicalRequirement,
    item.minSpec,
  ]
    .join(" ")
    .toLowerCase();
}

export function DownloadButton({ filename, label, path }: { filename: string; label: string; path: string }) {
  async function download() {
    const response = await fetch(path, { credentials: "include" });
    if (!response.ok) {
      return;
    }
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  }

  return (
    <Button className="w-full sm:w-auto" onClick={download} type="button" variant="outline">
      <Download className="h-4 w-4" aria-hidden="true" />
      {label}
    </Button>
  );
}

function dateExpired(value: string) {
  if (!value) {
    return false;
  }
  return value < new Date().toISOString().slice(0, 10);
}
