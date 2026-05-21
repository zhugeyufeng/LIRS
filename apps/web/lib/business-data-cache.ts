export const businessDataRevalidateSeconds = 300;
const maxBusinessDataCacheEntries = 500;

type BusinessDataCacheEntry = {
  expiresAt: number;
  lastAccessedAt: number;
  value: Promise<unknown>;
};

const globalForBusinessDataCache = globalThis as typeof globalThis & {
  __lirsBusinessDataCache?: Map<string, BusinessDataCacheEntry>;
};

const businessDataCache = globalForBusinessDataCache.__lirsBusinessDataCache ?? new Map<string, BusinessDataCacheEntry>();
globalForBusinessDataCache.__lirsBusinessDataCache = businessDataCache;

export function cachedBusinessData<T>(key: string, loader: () => Promise<T>, ttlSeconds = businessDataRevalidateSeconds): Promise<T> {
  const now = Date.now();
  const cached = businessDataCache.get(key);
  if (cached && cached.expiresAt > now) {
    cached.lastAccessedAt = now;
    return cached.value as Promise<T>;
  }
  if (cached) {
    businessDataCache.delete(key);
  }

  const value = loader().catch((error) => {
    if (businessDataCache.get(key)?.value === value) {
      businessDataCache.delete(key);
    }
    throw error;
  });
  businessDataCache.set(key, {
    expiresAt: now + ttlSeconds * 1000,
    lastAccessedAt: now,
    value,
  });
  pruneBusinessDataCache(now);
  return value;
}

export function clearBusinessDataCache() {
  businessDataCache.clear();
}

function pruneBusinessDataCache(now: number) {
  for (const [key, entry] of businessDataCache.entries()) {
    if (entry.expiresAt <= now) {
      businessDataCache.delete(key);
    }
  }
  while (businessDataCache.size > maxBusinessDataCacheEntries) {
    let oldestKey = "";
    let oldestAccessedAt = Number.POSITIVE_INFINITY;
    for (const [key, entry] of businessDataCache.entries()) {
      if (entry.lastAccessedAt < oldestAccessedAt) {
        oldestKey = key;
        oldestAccessedAt = entry.lastAccessedAt;
      }
    }
    if (!oldestKey) {
      break;
    }
    businessDataCache.delete(oldestKey);
  }
}
