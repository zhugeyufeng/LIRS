import Link from "next/link";
import { ClipboardList, PackageSearch, ShoppingCart, TestTube2, FlaskConical, Boxes } from "lucide-react";

export type MaterialsNavKey = "overview" | "standards" | "reagents" | "consumables" | "requests" | "purchases";

export function MaterialsNav({ active, admin = false }: { active: MaterialsNavKey; admin?: boolean }) {
  const basePath = admin ? "/admin/materials" : "/materials";
  const items = [
    { key: "overview" as const, label: "资源总览", href: basePath, icon: PackageSearch },
    { key: "standards" as const, label: "标准品", href: `${basePath}/standards`, icon: TestTube2 },
    { key: "reagents" as const, label: "试剂", href: `${basePath}/reagents`, icon: FlaskConical },
    { key: "consumables" as const, label: "耗材", href: `${basePath}/consumables`, icon: Boxes },
    { key: "requests" as const, label: "申领管理", href: `${basePath}/requests`, icon: ClipboardList },
    { key: "purchases" as const, label: "申购管理", href: `${basePath}/purchases`, icon: ShoppingCart },
  ];

  return (
    <nav className="mb-6 rounded-lg border bg-white p-2" aria-label={admin ? "后台资源管理导航" : "资源管理导航"}>
      <div className="grid gap-2 sm:grid-cols-3 xl:grid-cols-6">
        {items.map((item) => (
          <Link
            className={`inline-flex h-11 min-w-0 items-center justify-center gap-2 rounded-md px-3 text-sm font-bold transition-colors ${
              active === item.key ? "bg-primary text-white" : "text-slate-600 hover:bg-slate-50"
            }`}
            href={item.href}
            key={item.key}
          >
            <item.icon className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span className="min-w-0 truncate">{item.label}</span>
          </Link>
        ))}
      </div>
    </nav>
  );
}
