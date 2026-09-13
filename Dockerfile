FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE
RUN go build -o /bin/service ./cmd/${SERVICE}

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=build /bin/service /service
COPY --from=build /src/web /web
ENTRYPOINT ["/service"]
