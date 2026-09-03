FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/adapter-emulator .

FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/adapter-emulator /usr/local/bin/adapter-emulator
ENTRYPOINT ["adapter-emulator"]
