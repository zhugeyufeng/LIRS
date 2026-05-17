import { createClient } from "redis";

const redisUrl = process.env.REDIS_URL ?? `redis://${process.env.REDIS_ADDR ?? "localhost:6379"}`;
const redisPassword = process.env.REDIS_PASSWORD;

export const redis = createClient({
  url: redisUrl,
  password: redisPassword,
  socket: {
    connectTimeout: Number(process.env.REDIS_CONNECT_TIMEOUT_MS ?? 5000),
    reconnectStrategy: (retries) => Math.min(retries * 100, 3000),
  },
});

redis.on("error", (error) => {
  console.warn("redis error", error);
});

export async function ensureRedis() {
  if (!redis.isOpen) {
    await withTimeout(redis.connect(), Number(process.env.REDIS_CONNECT_TIMEOUT_MS ?? 5000));
  }
  return redis;
}

function withTimeout<T>(promise: Promise<T>, timeoutMs: number) {
  return new Promise<T>((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error("redis connect timeout")), timeoutMs);
    promise
      .then((value) => {
        clearTimeout(timer);
        resolve(value);
      })
      .catch((error) => {
        clearTimeout(timer);
        reject(error);
      });
  });
}
