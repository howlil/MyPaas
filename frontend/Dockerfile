## Build stage
FROM node:20-slim AS builder

WORKDIR /build

# Batasi pemakaian RAM compiler JS maksimal 1GB biar VM nggak kena OOM Killer
ENV NODE_OPTIONS="--max_old_space_size=1024"

COPY package.json ./
# Install murni, skip audit dan fund biar super enteng dan cepat
RUN npm install --no-audit --no-fund

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
