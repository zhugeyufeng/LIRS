import Link from "next/link";

type ApprovalSection = "pending" | "processed";

export function ApprovalSectionTabs({ active }: { active: ApprovalSection }) {
  return (
    <div className="mb-6 flex flex-wrap gap-2 rounded-lg border bg-white p-2">
      <Link
        className={`inline-flex h-10 items-center justify-center rounded-md px-4 text-sm font-medium ${
          active === "pending" ? "bg-primary text-white" : "text-slate-600 hover:bg-slate-50"
        }`}
        href="/approvals"
        prefetch={false}
      >
        待处理申请
      </Link>
      <Link
        className={`inline-flex h-10 items-center justify-center rounded-md px-4 text-sm font-medium ${
          active === "processed" ? "bg-primary text-white" : "text-slate-600 hover:bg-slate-50"
        }`}
        href="/approvals/processed"
        prefetch={false}
      >
        已处理申请
      </Link>
    </div>
  );
}
