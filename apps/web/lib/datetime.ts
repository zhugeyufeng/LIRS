const timeZone = "Asia/Shanghai";

const dateFormatter = new Intl.DateTimeFormat("zh-CN", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  timeZone,
});

const timeFormatter = new Intl.DateTimeFormat("zh-CN", {
  hour: "2-digit",
  minute: "2-digit",
  timeZone,
});

const dateTimeFormatter = new Intl.DateTimeFormat("zh-CN", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  timeZone,
});

export function formatDateOnly(value: string) {
  return formatDateValue(value, dateFormatter);
}

export function formatTimeOnly(value: string) {
  return formatDateValue(value, timeFormatter);
}

export function formatDateTime(value: string) {
  return formatDateValue(value, dateTimeFormatter);
}

export function formatDateTimeRange(startTime: string, endTime: string) {
  if (formatDateOnly(startTime) === formatDateOnly(endTime)) {
    return `${formatDateOnly(startTime)} ${formatTimeOnly(startTime)} - ${formatTimeOnly(endTime)}`;
  }
  return `${formatDateTime(startTime)} - ${formatDateTime(endTime)}`;
}

export function formatDurationHours(startTime: string, endTime: string) {
  const start = new Date(startTime);
  const end = new Date(endTime);
  const hours = (end.getTime() - start.getTime()) / 3600000;
  if (!Number.isFinite(hours) || hours <= 0) {
    return "未知";
  }
  return Number.isInteger(hours) ? `${hours} 小时` : `${hours.toFixed(1)} 小时`;
}

function formatDateValue(value: string, formatter: Intl.DateTimeFormat) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return formatter.format(date);
}
