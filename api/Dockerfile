FROM golang as builder

ARG GIT_COMMIT
ENV GIT_COMMIT=$GIT_COMMIT

WORKDIR /http-sms

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go get github.com/swaggo/swag/gen@v1.8.2
RUN go get github.com/swaggo/swag/cmd/swag@v1.8.2
RUN go install github.com/swaggo/swag/cmd/swag
RUN swag init
RUN go install github.com/gobuffalo/packr/v2/packr2@latest
RUN packr2

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$GIT_COMMIT" -o /bin/http-sms .

FROM alpine:latest

RUN addgroup -S http-sms && adduser -S http-sms -G http-sms

USER http-sms
WORKDIR /home/http-sms

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /zoneinfo.zip
COPY --from=builder /bin/http-sms ./

ENV ZONEINFO=/zoneinfo.zip

EXPOSE 8000

ENTRYPOINT ["./http-sms", "--dotenv=false"]
