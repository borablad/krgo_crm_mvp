# syntax=docker/dockerfile:1
# Один образ приложения: фронтенд собирается и встраивается в Go-бинарник.

FROM node:24-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

FROM golang:1.27-alpine AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=web /src/frontend/dist ./internal/web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/borablad/krgo_crm_mvp/backend/internal/app.Version=${VERSION}" \
    -o /out/cargo ./cmd/cargo

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/cargo /cargo
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/cargo"]
CMD ["serve"]
