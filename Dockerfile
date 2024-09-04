FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o learn-cooling ./cmd/learn-cooling

FROM gcr.io/distroless/base-debian11
COPY --from=builder /app/learn-cooling /learn-cooling
ENTRYPOINT ["/learn-cooling"]
