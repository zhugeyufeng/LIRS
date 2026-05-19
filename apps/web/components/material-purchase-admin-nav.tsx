import Link from "next/link";
import { Boxes, ClipboardList, ShoppingCart } from "lucide-react";

type MaterialPurchaseAdminNavKey = "orders" | "catalog" | "projects";

export function MaterialPurchaseAdminNav({ active }: { active: MaterialPurchaseAdminNavKey }) {
  const items = [
    { key: "orders" as const, label: "申购单列表", href: "/admin/materials/purchases", icon: ShoppingCart },
    { key: "catalog" as const, label: "可采购物资目录", href: "/admin/materials/purchases/catalog", icon: Boxes },
    { key: "projects" as const, label: "采购项目名称及编号", href: "/admin/materials/purchases/projects", icon: ClipboardList },
  ];

  return (
    <nav className="mb-6 rounded-lg border bg-white p-2" aria-label="资源申购管理导航">
      <div className="grid gap-2 md:grid-cols-3">
        {items.map((item) => (
          <Link
            className={`inline-flex h-11 min-w-0 items-center justify-center gap-2 rounded-md px-3 text-sm font-bold transition-colors ${
              active === item.key ? "bg-primary text-white" : "text-slate-600 hover:bg-slate-50"
            }`}
            href={item.href}
            key={item.key}
            prefetch={false}
          >
            <item.icon className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span className="min-w-0 truncate">{item.label}</span>
          </Link>
        ))}
      </div>
    </nav>
  );
}
