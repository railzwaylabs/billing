# syntax=docker/dockerfile:1.7

FROM node:24-alpine AS builder
WORKDIR /src
RUN corepack enable

ARG VITE_APP_NAME="Billing Console"
ARG VITE_BACKEND_URL=""
ENV VITE_APP_NAME=$VITE_APP_NAME
ENV VITE_BACKEND_URL=$VITE_BACKEND_URL

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/console/package.json apps/console/package.json
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

COPY apps/console ./apps/console
RUN pnpm --filter @railzway/billing-console build

FROM nginx:1.29-alpine
COPY apps/console/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /src/apps/console/dist /usr/share/nginx/html
EXPOSE 80
