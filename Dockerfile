FROM golang:1.26.4-alpine AS builder

WORKDIR /app
COPY go.* /app/
RUN go mod download
COPY . /app/
# Fully static, stripped binary so it runs in an empty image.
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server

FROM scratch
COPY --from=builder /app/server /server
USER 65532:65532
ENTRYPOINT ["/server"]
