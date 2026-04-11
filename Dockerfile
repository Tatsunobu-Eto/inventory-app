# ---- ビルドステージ ----
FROM golang:1.23-alpine AS builder
WORKDIR /app

# vendorディレクトリを使用してオフラインビルド（go mod downloadなし）
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o inventory .

# ---- 実行ステージ ----
# golang:1.23-alpineを再利用（追加イメージのpull不要）
FROM golang:1.23-alpine
WORKDIR /app

COPY --from=builder /app/inventory .

RUN mkdir -p uploads

EXPOSE 8080
CMD ["./inventory"]
