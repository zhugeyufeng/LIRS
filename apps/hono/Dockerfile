# syntax=docker/dockerfile:1.7
FROM node:22-alpine AS deps
WORKDIR /src
COPY package.json package-lock.json .npmrc ./
COPY apps/hono/package.json apps/hono/package.json
RUN --mount=type=cache,target=/root/.npm npm ci --workspace @lirs/hono --include-workspace-root=false --prefer-offline

FROM node:22-alpine AS builder
WORKDIR /src
COPY --from=deps /src/node_modules ./node_modules
COPY package.json package-lock.json ./
COPY apps/hono ./apps/hono
RUN npm --workspace apps/hono run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=deps /src/node_modules ./node_modules
COPY --from=builder /src/apps/hono/dist ./apps/hono/dist
COPY apps/hono/package.json ./apps/hono/package.json
WORKDIR /app/apps/hono
EXPOSE 8090
CMD ["npm", "run", "start"]
