import Link from "next/link";

type FinanceTab = "overview" | "instrument-fees" | "material-fees" | "bills" | "reconciliation" | "projects" | "invoices";

const tabs: { key: FinanceTab; label: string; href: string }[] = [
  { key: "overview", label: "财务总览", href: "/finance" },
  { key: "instrument-fees", label: "机时费用", href: "/finance/instrument-fees" },
  { key: "material-fees", label: "耗材费用", href: "/finance/material-fees" },
  { key: "bills", label: "月度账单", href: "/finance/bills" },
  { key: "reconciliation", label: "结算对账", href: "/finance/reconciliation" },
  { key: "projects", label: "经费账户", href: "/finance/projects" },
  { key: "invoices", label: "发票记录", href: "/finance/invoices" },
];

export function FinanceTabs({ active }: { active: FinanceTab }) {
  return (
    <nav className="mb-6 grid gap-2 rounded-lg border bg-white p-2 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-7" aria-label="财务与计费层级导航">
      {tabs.map((tab) => (
        <Link
          className={`inline-flex h-10 min-w-0 items-center justify-center rounded-md px-3 text-sm font-medium ${
            active === tab.key ? "bg-primary text-white" : "text-slate-600 hover:bg-slate-50"
          }`}
          href={tab.href}
          key={tab.key}
          prefetch={false}
        >
          <span className="min-w-0 truncate">{tab.label}</span>
        </Link>
      ))}
    </nav>
  );
}
