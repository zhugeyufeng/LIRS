import { MaterialResourceCatalogPage } from "@/components/material-resource-type-page";

export default function ReagentsPage({ searchParams }: { searchParams?: Promise<{ q?: string; category?: string; status?: string }> }) {
  return <MaterialResourceCatalogPage productType="reagent" searchParams={searchParams} />;
}
