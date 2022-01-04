FROM debian:stretch

ARG IMAGE_VERSION=0.1

RUN  apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY ./bin/alertmanager-cli /usr/local/bin/alertmanager-cli

EXPOSE 8080

CMD ["alertmanager-cli"]