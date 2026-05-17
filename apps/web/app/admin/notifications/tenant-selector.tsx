"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Tenant } from "@/lib/api";

export function TenantSelector({ selectedTenant, tenants }: { selectedTenant: Tenant; tenants: Tenant[] }) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  function changeTenant(value: string) {
    const next = new URLSearchParams(searchParams.toString());
    next.set("tenantId", value);
    router.push(`${pathname}?${next.toString()}`);
  }

  return (
    <div className="rounded-lg border bg-slate-50/60 p-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <label className="block min-w-0 flex-1 space-y-2">
          <span className="text-sm font-medium text-slate-900">选择机构</span>
          <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" onChange={(event) => changeTenant(event.currentTarget.value)} value={selectedTenant.id}>
            {tenants.map((tenant) => (
              <option key={tenant.id} value={tenant.id}>
                {tenant.name}
              </option>
            ))}
          </select>
        </label>
        <p className="break-words text-xs text-slate-500 md:max-w-lg md:text-right">当前机构：{selectedTenant.name}。切换后通知列表、公告编辑和钉钉推送目标都会使用该机构。</p>
      </div>
    </div>
  );
}
