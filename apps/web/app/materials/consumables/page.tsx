import { MaterialResourceCatalogPage } from "@/components/material-resource-type-page";

export default function ConsumablesPage({ searchParams }: { searchParams?: Promise<{ q?: string; category?: string; status?: string }> }) {
  return <MaterialResourceCatalogPage productType="consumable" searchParams={searchParams} />;
}
