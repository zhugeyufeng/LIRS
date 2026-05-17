import { AdminMaterialResourceManagementPage } from "@/components/material-resource-type-page";

export default function AdminConsumablesPage({ searchParams }: { searchParams?: Promise<{ q?: string; category?: string; status?: string }> }) {
  return <AdminMaterialResourceManagementPage productType="consumable" searchParams={searchParams} />;
}
