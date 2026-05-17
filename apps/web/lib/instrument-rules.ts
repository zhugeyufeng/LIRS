import type { Instrument } from "@/lib/api";

export function formatServiceWindow(instrument: Pick<Instrument, "serviceStartHour" | "serviceEndHour">) {
  return `${formatHour(instrument.serviceStartHour)}-${formatHour(instrument.serviceEndHour)}`;
}

function formatHour(value: number) {
  const hour = Number.isFinite(value) ? Math.max(0, Math.min(24, Math.trunc(value))) : 0;
  return `${String(hour).padStart(2, "0")}:00`;
}
