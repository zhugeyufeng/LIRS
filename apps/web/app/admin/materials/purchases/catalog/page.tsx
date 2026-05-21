import { AdminShell, requireAdminSection } from "@/components/admin-shell";
import { MaterialPurchaseAdminNav } from "@/components/material-purchase-admin-nav";
import { MaterialsNav } from "@/components/materials-nav";
import { PurchasableMaterialManager } from "@/components/purchasable-material-manager";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function AdminPurchasableMaterialsCatalogPage() {
  await requireAdminSection("materials");
  const [purchasableMaterials, procurementProjects] = await Promise.all([api.purchasableMaterials().catch(() => []), api.procurementProjects().catch(() => [])]);

  return (
    <AdminShell active="materials" title="可采购物资目录" description="维护可供申购选择的采购物资，支持新增、导入、搜索、修改和删除。">
      <MaterialsNav active="purchases" admin />
      <MaterialPurchaseAdminNav active="catalog" />

      <Card>
        <CardHeader>
          <CardTitle>可采购物资目录</CardTitle>
        </CardHeader>
        <CardContent>
          <PurchasableMaterialManager items={purchasableMaterials} projects={procurementProjects} />
        </CardContent>
      </Card>
    </AdminShell>
  );
}
