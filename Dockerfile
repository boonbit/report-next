# report-next designer: web bundle + Go server in one image.
# Needs network once (node/npm and Go module downloads).
FROM node:22-alpine AS web
WORKDIR /web
COPY report-next/web/package.json ./
RUN npm install --no-audit --no-fund
COPY report-next/web/ ./
RUN npm run build

FROM golang:1.25-alpine AS server
WORKDIR /app
COPY report-next/go.mod report-next/go.sum ./
RUN go mod download
COPY report-next/ ./
COPY --from=web /web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /out/report-designer ./cmd/report-designer

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=server /out/report-designer ./
COPY --from=web /web/dist ./web/dist
# Seed data for the demo dictionary (server reads it at startup when present).
COPY --from=server /app/internal/report/testdata/customers.csv ./internal/report/testdata/customers.csv
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://localhost:8080/ || exit 1
CMD ["./report-designer"]
