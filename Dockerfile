FROM kopeisec/debian:13

ARG TARGETARCH

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY build/notification-gateway_linux_${TARGETARCH} /usr/local/bin/notification-gateway
COPY config.yaml .

EXPOSE 39498

ENTRYPOINT ["/usr/local/bin/notification-gateway"]
