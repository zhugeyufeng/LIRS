import Link from "next/link";

type WorkflowTab = "templates" | "rules" | "approvers" | "instances" | "exceptions";

const tabs: { key: WorkflowTab; label: string; href: string }[] = [
  { key: "templates", label: "流程模板", href: "/admin/workflows/templates" },
  { key: "rules", label: "审批规则", href: "/admin/workflows/rules" },
  { key: "approvers", label: "审批人配置", href: "/admin/workflows/approvers" },
  { key: "instances", label: "流程实例", href: "/admin/workflows/instances" },
  { key: "exceptions", label: "超时异常", href: "/admin/workflows/exceptions" },
];

export function WorkflowTabs({ active }: { active: WorkflowTab }) {
  return (
    <nav className="mb-6 grid gap-2 rounded-lg border bg-white p-2 sm:grid-cols-2 lg:grid-cols-5" aria-label="审批流配置层级导航">
      {tabs.map((tab) => (
        <Link
          className={`inline-flex h-10 min-w-0 items-center justify-center rounded-md px-3 text-sm font-medium ${
            active === tab.key ? "bg-primary text-white" : "text-slate-600 hover:bg-slate-50"
          }`}
          href={tab.href}
          key={tab.key}
        >
          <span className="min-w-0 truncate">{tab.label}</span>
        </Link>
      ))}
    </nav>
  );
}
