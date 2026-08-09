## Build stage
FROM node:24-slim AS builder

WORKDIR /build

ENV NODE_OPTIONS="--max_old_space_size=1024"

RUN corepack enable pnpm

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .

RUN pnpm build
RUN pnpm prune --prod

## Runtime stage
FROM node:24-slim

WORKDIR /app

COPY package.json ./
COPY --from=builder /build/node_modules ./node_modules
COPY --from=builder /build/build ./build

EXPOSE 3000

ENV NODE_ENV=production

CMD ["node", "build"]
