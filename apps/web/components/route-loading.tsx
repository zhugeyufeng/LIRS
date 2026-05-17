type RouteLoadingProps = {
  label?: string;
};

export function RouteLoading({ label = "正在加载页面" }: RouteLoadingProps) {
  return (
    <main className="mx-auto w-full max-w-7xl px-4 pt-6 pb-4 sm:px-6 sm:pt-8 sm:pb-4 lg:px-8" aria-busy="true" aria-live="polite">
      <div className="mb-6">
        <div className="h-3 w-24 rounded bg-primary/20" />
        <div className="mt-3 h-8 w-56 rounded bg-slate-200" />
        <p className="mt-3 text-sm text-slate-500">{label}...</p>
      </div>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }, (_, index) => (
          <div className="rounded-lg border bg-white p-4" key={index}>
            <div className="h-3 w-20 rounded bg-slate-200" />
            <div className="mt-4 h-7 w-28 rounded bg-slate-100" />
          </div>
        ))}
      </div>
      <div className="mt-6 grid gap-4 lg:grid-cols-3">
        {Array.from({ length: 6 }, (_, index) => (
          <div className="rounded-lg border bg-white p-4" key={`card-${index}`}>
            <div className="flex items-center gap-3">
              <div className="h-10 w-10 rounded-md bg-primary/10" />
              <div className="min-w-0 flex-1">
                <div className="h-4 w-2/3 rounded bg-slate-200" />
                <div className="mt-2 h-3 w-1/2 rounded bg-slate-100" />
              </div>
            </div>
            <div className="mt-4 space-y-2">
              <div className="h-3 w-full rounded bg-slate-100" />
              <div className="h-3 w-4/5 rounded bg-slate-100" />
            </div>
          </div>
        ))}
      </div>
    </main>
  );
}
