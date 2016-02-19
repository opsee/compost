FROM quay.io/opsee/vinz:latest

ENV COMPOST_ADDRESS ""
ENV APPENV ""

RUN apk add --update bash ca-certificates curl

COPY run.sh /
COPY target/linux/amd64/bin/* /

EXPOSE 9096
CMD ["/compost"]
