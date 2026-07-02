FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /buswatch .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /buswatch /buswatch
ENV PORT=8080
EXPOSE 8080
CMD ["/buswatch"]
