## Build stage
FROM node:20-alpine AS builder

WORKDIR /build

RUN corepack enable pnpm

COPY package.json pnpm-lock.yaml ./
RUN pnpm install

COPY . .

RUN pnpm build
# Remove devDependencies to save space before copying to runtime
RUN pnpm prune --prod

## Runtime stage
FROM node:20-alpine

WORKDIR /app

COPY package.json ./
COPY --from=builder /build/node_modules ./node_modules
COPY --from=builder /build/build ./build

EXPOSE 3000

ENV NODE_ENV=production

CMD ["node", "build"]
