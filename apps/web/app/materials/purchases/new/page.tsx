import Link from "next/link";
import { ArrowLeft, ShoppingCart } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { MaterialPurchaseForm } from "@/components/material-purchase-form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

type SearchParams = {
  materialId?: string;
};

export default async function MaterialPurchaseNewPage({ searchParams }: { searchParams?: Promise<SearchParams> }) {
  const params = (await searchParams) ?? {};
  const materials = await api.materials();
  const material = params.materialId ? materials.find((item) => item.id === params.materialId) : undefined;

  return (
    <AppShell>
      <Link className="mb-5 inline-flex items-center gap-2 text-sm text-slate-600 hover:text-primary" href={material ? `/materials/${material.id}` : "/materials/purchases"}>
        <ArrowLeft className="h-4 w-4" />
        {material ? "返回资源详情" : "返回申购记录"}
      </Link>
      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
        <section>
          <div className="mb-4">
            <p className="text-xs font-bold uppercase tracking-widest text-primary">资源申购</p>
            <h1 className="mt-2 text-2xl font-bold text-slate-900 sm:text-3xl">新建申购</h1>
            <p className="mt-2 text-sm leading-6 text-muted-foreground">
              {material ? "当前申购对象已经固定，提交后进入审批流程。" : "选择资源、填写数量、预算和申购原因后提交审批。"}
            </p>
          </div>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShoppingCart className="h-5 w-5 text-primary" />
                申购表单
              </CardTitle>
            </CardHeader>
            <CardContent>
              <MaterialPurchaseForm inline material={material} materials={materials} />
            </CardContent>
          </Card>
        </section>
        <aside>
          <Card>
            <CardHeader>
              <CardTitle>申购说明</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm leading-6 text-slate-700">
              <p>申购提交后进入审批流程，通过后由试剂管理员标记下单，到货后写入库存流水。</p>
              {material ? (
                <div className="rounded-lg border bg-slate-50 p-3">
                  <p className="font-bold text-slate-900">{material.name}</p>
                  <p className="mt-1 text-slate-500">{material.category} / 当前库存 {material.stock}{material.unit}</p>
                </div>
              ) : null}
            </CardContent>
          </Card>
        </aside>
      </div>
    </AppShell>
  );
}
