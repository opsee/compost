#!/bin/bash
set -e

APPENV=${APPENV:-compostenv}

/opt/bin/s3kms -r us-west-1 get -b opsee-keys -o dev/$APPENV > /$APPENV

source /$APPENV && \
  /opt/bin/s3kms -r us-west-1 get -b opsee-keys -o dev/vape.key > /vape.key && \
	/compost
