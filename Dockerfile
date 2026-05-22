FROM golang:1.25-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/dedsite ./cmd/server

FROM debian:bookworm-slim

WORKDIR /app

RUN useradd --create-home --shell /usr/sbin/nologin dedsite \
	&& mkdir -p /app/data \
	&& chown -R dedsite:dedsite /app

COPY --from=build /out/dedsite /app/dedsite
COPY templates /app/templates
COPY static /app/static

USER dedsite

ENV ADDR=:8082
ENV DATABASE_PATH=/app/data/site.db

EXPOSE 8082
VOLUME ["/app/data"]

CMD ["/app/dedsite"]
