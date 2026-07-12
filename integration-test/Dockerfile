# Step 1: Modules caching
FROM golang:1.26-alpine AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN go mod download

# Step 2: Tests
FROM golang:1.26-alpine
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app

RUN go env -w CGO_ENABLED=0

CMD ["go", "test", "-v", "./integration-test/..."]
