FROM golang:1.26-alpine AS build
WORKDIR /src
COPY apps/api-go/go.mod apps/api-go/go.sum* ./
RUN go mod download
COPY apps/api-go/ ./
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.22
RUN adduser -D -H appuser
USER appuser
COPY --from=build /out/api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
