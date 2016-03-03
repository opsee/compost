FROM quay.io/opsee/vinz:latest

ENV VAPE_KEYFILE "/vape.test.key"
ENV COMPOST_ADDRESS ""
ENV APPENV ""

RUN apk add --update bash ca-certificates curl

COPY run.sh /
COPY target/linux/amd64/bin/* /
COPY vape.test.key /
COPY static /static

EXPOSE 9096
CMD ["/compost"]
