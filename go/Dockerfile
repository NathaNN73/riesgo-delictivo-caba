FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/trainer ./cmd/trainer

FROM alpine:3.20
COPY --from=build /app/trainer /app/trainer
ENTRYPOINT ["/app/trainer"]
