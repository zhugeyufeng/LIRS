import Link from "next/link";
import { Beaker, Building2, CalendarCheck2, ClipboardCheck, MapPin, Search, Tags } from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { api, Instrument } from "@/lib/api";
import { instrumentStatusLabel } from "@/lib/status-labels";

const statusMeta: Record<string, { label: string; className: string }> = {
  available: { label: "可用", className: "bg-emerald-600 text-white" },
  busy: { label: "繁忙", className: "bg-amber-500 text-white" },
  maintenance: { label: "维护中", className: "bg-destructive text-white" },
  disabled: { label: "已停用", className: "bg-slate-500 text-white" },
};

export default async function HomePage({
  searchParams,
}: {
  searchParams?: Promise<{ search?: string; category?: string; department?: string; status?: string; page?: string }>;
}) {
  const params = (await searchParams) ?? {};
  const pageSize = 6;
  const page = Math.max(Number(params.page ?? 1) || 1, 1);
  const [dashboard, allInstruments] = await Promise.all([
    api.dashboard().catch(() => ({
      todayReservations: 0,
      pendingApprovals: 0,
      inUseReservations: 0,
      completedReservations: 0,
      fulfillmentRate: 0,
      activeInstruments: 0,
      monthlyRevenue: 0,
    })),
    api.instruments().catch(() => [] as Instrument[]),
  ]);
  const instruments = filterInstruments(allInstruments, params);
  const groupingInstruments = filterInstruments(allInstruments, { ...params, department: undefined });
  const categoryInstruments = filterInstruments(allInstruments, { ...params, category: undefined, department: undefined });
  const totalPages = Math.max(Math.ceil(instruments.length / pageSize), 1);
  const currentPage = Math.min(page, totalPages);
  const pagedInstruments = instruments.slice((currentPage - 1) * pageSize, currentPage * pageSize);
  const sections = groupInstruments(pagedInstruments);
  const categories = categoryOptions(groupingInstruments);
  const instrumentCategories = instrumentCategoryOptions(categoryInstruments);
  const selectedCategory = params.department ?? "";
  const currentDate = new Date().toLocaleDateString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    weekday: "long",
    timeZone: "Asia/Shanghai",
  });

  return (
    <AppShell mainClassName="mx-auto w-full max-w-7xl px-3 pt-4 pb-4 sm:px-6 sm:pt-8 sm:pb-4 lg:px-8">
      <section className="mb-8 space-y-4">
        <div className="flex flex-col justify-between gap-3 sm:gap-4 md:flex-row md:items-end">
          <div>
            <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">仪器预约大厅</h1>
            <p className="mt-2 text-sm text-muted-foreground sm:text-lg">当前日期: {currentDate}</p>
          </div>
        </div>
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
          <form action="/instruments" className="rounded-xl border bg-white p-3 shadow-sm sm:p-4">
            <div className="grid gap-3 sm:grid-cols-2 xl:flex xl:items-center">
              <div className="relative min-w-0 w-full xl:max-w-md">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <input
                  className="h-10 w-full rounded-lg border bg-background pl-10 pr-4 text-sm outline-none transition-all focus:ring-2 focus:ring-primary/20"
                  defaultValue={params.search ?? ""}
                  name="search"
                  placeholder="搜索仪器名称、型号、部门..."
                  type="search"
                />
              </div>
              <select className="h-10 w-full min-w-0 rounded-lg border bg-background px-3 text-sm xl:w-auto" defaultValue={params.status ?? ""} name="status">
                <option value="">全部状态</option>
                <option value="available">可用</option>
                <option value="busy">繁忙</option>
                <option value="maintenance">维护中</option>
                <option value="disabled">已停用</option>
              </select>
              <select className="h-10 w-full min-w-0 rounded-lg border bg-background px-3 text-sm xl:w-auto" defaultValue={params.category ?? ""} name="category">
                <option value="">全部仪器分类</option>
                {instrumentCategories.map((category) => (
                  <option key={category.name} value={category.name}>
                    {category.name} ({category.count})
                  </option>
                ))}
              </select>
              <button className="inline-flex h-10 w-full min-w-0 items-center justify-center whitespace-nowrap rounded-lg bg-primary px-4 text-sm font-bold text-white sm:w-auto" type="submit">
                搜索
              </button>
            </div>
            <div className="mt-4 border-t pt-4">
              <div className="mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-3">
                <p className="text-xs font-bold uppercase text-slate-500">部门分类</p>
                {selectedCategory ? (
                  <Link className="inline-flex h-7 items-center justify-center whitespace-nowrap rounded-md px-2 text-xs font-medium text-primary hover:bg-primary/10" href={homeHref({ ...params, department: undefined })} prefetch={false}>
                    清除分类
                  </Link>
                ) : null}
              </div>
              <div className="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap">
                <Link className={`inline-flex h-8 min-w-0 items-center justify-center gap-1 whitespace-nowrap rounded-full px-3 text-xs font-bold transition sm:px-4 ${selectedCategory === "" ? "bg-primary text-white shadow-sm shadow-primary/20" : "bg-slate-100 text-slate-600 hover:bg-slate-200"}`} href={homeHref({ ...params, page: undefined, department: undefined })} prefetch={false}>
                  全部 <span className="ml-1 opacity-70">{groupingInstruments.length}</span>
                </Link>
                {categories.map((category) => (
                  <Link
                    className={`inline-flex h-8 min-w-0 items-center justify-center gap-1 whitespace-nowrap rounded-full px-3 text-xs font-bold transition sm:px-4 ${
                      selectedCategory === category.name ? "bg-primary text-white shadow-sm shadow-primary/20" : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                    }`}
                    href={homeHref({
                      ...params,
                      page: undefined,
                      department: category.name,
                    })}
                    key={category.name}
                    prefetch={false}
                  >
                    <span className="min-w-0 truncate">{category.name}</span> <span className="ml-1 opacity-70">{category.count}</span>
                  </Link>
                ))}
              </div>
            </div>
          </form>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <CalendarCheck2 className="h-4 w-4 text-primary" />
                预约与履约情况
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <Metric label="今日预约" value={`${dashboard.todayReservations} 单`} />
                <Metric label="待审批" value={`${dashboard.pendingApprovals} 单`} />
                <Metric label="使用中" value={`${dashboard.inUseReservations} 单`} />
                <Metric label="已履约" value={`${dashboard.completedReservations} 单`} />
              </div>
              <div className="rounded-lg border border-emerald-100 bg-emerald-50 p-3 text-sm text-emerald-800">
                当前履约率 <span className="font-bold">{dashboard.fulfillmentRate.toFixed(1)}%</span>
              </div>
            </CardContent>
          </Card>
        </div>
      </section>

      <section className="space-y-12">
        {Object.entries(sections).map(([section, items]) => (
          <section className="space-y-6 scroll-mt-24" key={section}>
            <div className="flex flex-wrap items-center gap-3 border-b pb-4">
              <div className="h-6 w-1.5 rounded-full bg-primary" />
              <h2 className="text-lg font-bold sm:text-xl">{section}</h2>
              <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">{items.length} 台仪器</span>
            </div>
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 xl:grid-cols-3">
              {items.map((instrument) => (
                <InstrumentCard instrument={instrument} key={instrument.id} />
              ))}
            </div>
          </section>
        ))}
        <div className="flex flex-col justify-between gap-3 border-t pt-4 sm:flex-row sm:items-center">
          <p className="text-sm text-muted-foreground">
            共 {instruments.length} 台，当前第 {currentPage} / {totalPages} 页
          </p>
          <div className="flex gap-2">
            {currentPage <= 1 ? (
              <Button disabled size="sm" variant="outline">上一页</Button>
            ) : (
              <Button asChild size="sm" variant="outline">
                <Link href={homeHref({ ...params, page: String(currentPage - 1) })} prefetch={false}>上一页</Link>
              </Button>
            )}
            {currentPage >= totalPages ? (
              <Button disabled size="sm" variant="outline">下一页</Button>
            ) : (
              <Button asChild size="sm" variant="outline">
                <Link href={homeHref({ ...params, page: String(currentPage + 1) })} prefetch={false}>下一页</Link>
              </Button>
            )}
          </div>
        </div>
      </section>
    </AppShell>
  );
}

function InstrumentCard({ instrument }: { instrument: Instrument }) {
  const meta = statusMeta[instrument.status] ?? { label: instrumentStatusLabel(instrument.status), className: "bg-slate-500 text-white" };

  return (
    <Card className="group p-4 transition-all duration-300 hover:shadow-md">
      <div className="relative mb-4 aspect-[4/3] overflow-hidden rounded-lg border bg-gradient-to-br from-slate-100 to-blue-50">
        <div className="absolute inset-0 flex items-center justify-center">
          <Beaker className="h-16 w-16 text-primary/35 transition-transform duration-500 group-hover:scale-110" aria-hidden="true" />
        </div>
        <span className={`absolute right-2 top-2 rounded px-2 py-1 text-[10px] font-bold uppercase tracking-wider shadow-sm ${meta.className}`}>
          {meta.label}
        </span>
      </div>
      <h3 className="line-clamp-1 text-base font-bold">{instrument.name}</h3>
      <p className="mt-1 line-clamp-2 min-h-10 text-xs leading-5 text-muted-foreground">{instrument.description}</p>
      <div className="mt-4 grid grid-cols-1 gap-2 border-t pt-4 text-xs text-slate-600 sm:grid-cols-2 sm:gap-3">
        <span className="flex items-center gap-1">
          <Tags className="h-3.5 w-3.5 text-primary" aria-hidden="true" />
          <span className="min-w-0 truncate">{instrument.category}</span>
        </span>
        <span className="flex items-center gap-1">
          <MapPin className="h-3.5 w-3.5 text-primary" aria-hidden="true" />
          <span className="min-w-0 truncate">{instrument.location}</span>
        </span>
        <span className="flex items-center gap-1">
          <Building2 className="h-3.5 w-3.5 text-primary" aria-hidden="true" />
          <span className="min-w-0 truncate">{instrument.department}</span>
        </span>
        <span className="font-bold text-primary">¥{instrument.hourlyRate}/小时</span>
      </div>
      <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <span className="text-[10px] font-bold uppercase text-muted-foreground">费用</span>
        <Button asChild className="w-full sm:w-auto" size="sm">
          <Link href={`/instruments/${instrument.id}`} prefetch={false}>查看详情</Link>
        </Button>
      </div>
    </Card>
  );
}

function filterInstruments(instruments: Instrument[], params: { search?: string; category?: string; department?: string; status?: string }) {
  const search = (params.search ?? "").trim().toLowerCase();
  return instruments.filter((instrument) => {
    const matchesSearch =
      search === "" ||
      [instrument.name, instrument.model, instrument.brand, instrument.department, instrument.groupName, instrument.location, instrument.category]
        .some((value) => (value ?? "").toLowerCase().includes(search));
    const matchesCategory = !params.category || instrument.category === params.category;
    const matchesDepartment = !params.department || instrument.department === params.department;
    const matchesStatus = !params.status || instrument.status === params.status;
    return matchesSearch && matchesCategory && matchesDepartment && matchesStatus;
  });
}

function groupInstruments(instruments: Instrument[]) {
  return instruments.reduce<Record<string, Instrument[]>>((acc, instrument) => {
    const key = instrument.department || "未分类";
    acc[key] = acc[key] ?? [];
    acc[key].push(instrument);
    return acc;
  }, {});
}

function categoryOptions(instruments: Instrument[]) {
  const counts = new Map<string, number>();
  for (const instrument of instruments) {
    const key = instrument.department || "未分类";
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  return Array.from(counts, ([name, count]) => ({ name, count })).sort((a, b) => a.name.localeCompare(b.name, "zh-CN"));
}

function instrumentCategoryOptions(instruments: Instrument[]) {
  const counts = new Map<string, number>();
  for (const instrument of instruments) {
    const key = instrument.category || "未分类";
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  return Array.from(counts, ([name, count]) => ({ name, count })).sort((a, b) => a.name.localeCompare(b.name, "zh-CN"));
}

function homeHref(params: { search?: string; category?: string; department?: string; status?: string; page?: string }) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value && String(value).trim() !== "") {
      query.set(key, String(value));
    }
  }
  const suffix = query.toString();
  return suffix ? `/instruments?${suffix}` : "/instruments";
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border bg-slate-50 p-3 sm:p-4">
      <div className="mb-2 flex items-center gap-2 text-xs text-slate-500">
        <ClipboardCheck className="h-3.5 w-3.5" aria-hidden="true" />
        {label}
      </div>
      <div className="text-xl font-bold sm:text-2xl">{value}</div>
    </div>
  );
}
