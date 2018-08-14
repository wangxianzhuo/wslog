FROM alpine:latest

RUN mkdir -p /usr/app
WORKDIR /usr/app
RUN apk --update add ca-certificates
COPY build/wslog wslog
EXPOSE 9000

ENTRYPOINT [ "./wslog", "--kafka-brokers", "kafka.host:9092" ]