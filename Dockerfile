# create parent image
FROM golang:1.23.4-alpine as builder

#set working directory
WORKDIR /app


RUN apk update && apk add --no-cache build-base sqlite-dev

COPY go.mod go.sum ./

#RUN go env -w GOPROXY=https://proxy.golang.org,direct

#download dependencies
RUN go mod download

#copy source code
COPY . .

#RUN ls -al /app

#build app
ENV CGO_ENABLED=1
RUN go build -o main .

#using alpine image for staging
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/static ./static
COPY --from=builder /app/.env .
#COPY --from=builder /app/device_data.db .

EXPOSE 8080

CMD ["./main"]



