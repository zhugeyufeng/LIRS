export const businessDataRevalidateSeconds = 300;

type BusinessDataCacheEntry = {
  expiresAt: number;
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
    return cached.value as Promise<T>;
  }

  const value = loader().catch((error) => {
    if (businessDataCache.get(key)?.value === value) {
      businessDataCache.delete(key);
    }
    throw error;
  });
  businessDataCache.set(key, {
    expiresAt: now + ttlSeconds * 1000,
    value,
  });
  return value;
}

export function clearBusinessDataCache() {
  businessDataCache.clear();
}
