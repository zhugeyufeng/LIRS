import { AdminMaterialResourceManagementPage } from "@/components/material-resource-type-page";

export default function AdminStandardsPage({ searchParams }: { searchParams?: Promise<{ q?: string; category?: string; status?: string }> }) {
  return <AdminMaterialResourceManagementPage productType="standard" searchParams={searchParams} />;
}
