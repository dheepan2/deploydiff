FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/deploydiff .

FROM alpine:3.21

COPY --from=build /out/deploydiff /usr/local/bin/deploydiff
COPY entrypoint.sh /entrypoint.sh

RUN chmod 0555 /entrypoint.sh /usr/local/bin/deploydiff

ENTRYPOINT ["/entrypoint.sh"]
