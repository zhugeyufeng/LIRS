import { MaterialResourceCatalogPage } from "@/components/material-resource-type-page";

export default function StandardsPage({ searchParams }: { searchParams?: Promise<{ q?: string; category?: string; status?: string }> }) {
  return <MaterialResourceCatalogPage productType="standard" searchParams={searchParams} />;
}
