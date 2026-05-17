import { AdminMaterialResourceManagementPage } from "@/components/material-resource-type-page";

export default function AdminReagentsPage({ searchParams }: { searchParams?: Promise<{ q?: string; category?: string; status?: string }> }) {
  return <AdminMaterialResourceManagementPage productType="reagent" searchParams={searchParams} />;
}
