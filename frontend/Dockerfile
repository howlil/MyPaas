## Build stage
FROM node:20-slim AS builder

WORKDIR /build

COPY package.json ./
# Menggunakan murni npm bawaan Node.js biar nggak ada drama pnpm/corepack crash
RUN npm install

COPY . .

RUN npm run build
# Remove devDependencies to save space before copying to runtime
RUN npm prune --omit=dev

## Runtime stage
FROM node:20-slim

WORKDIR /app

COPY package.json ./
COPY --from=builder /build/node_modules ./node_modules
COPY --from=builder /build/build ./build

EXPOSE 3000

ENV NODE_ENV=production

CMD ["node", "build"]
