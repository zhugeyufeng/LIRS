import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { MaterialPurchaseAdminNav } from "@/components/material-purchase-admin-nav";
import { ProcurementProjectManager } from "@/components/material-purchase-form";
import { MaterialsNav } from "@/components/materials-nav";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function AdminProcurementProjectsPage() {
  await requireAdminSection("materials");
  const procurementProjects = await api.procurementProjects().catch(() => []);

  return (
    <AdminShell active="materials" title="采购项目名称及编号" description="维护采购项目名称、编号、启停状态和有效期；超过有效期后关联物资不能申购。">
      <MaterialsNav active="purchases" admin />
      <MaterialPurchaseAdminNav active="projects" />

      <Card>
        <CardHeader>
          <CardTitle>采购项目名称及编号</CardTitle>
        </CardHeader>
        <CardContent>
          <ProcurementProjectManager projects={procurementProjects} />
        </CardContent>
      </Card>
    </AdminShell>
  );
}
