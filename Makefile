all: fmt build

build:
	gb build

clean:
	rm -fr target bin pkg

fmt:
	@gofmt -w ./

docker: fmt
	docker run \
		--env-file ./$(APPENV) \
		-e "TARGETS=linux/amd64" \
		-v `pwd`:/build quay.io/opsee/build-go \
		&& docker build -t quay.io/opsee/compost .

run: docker
	docker run \
		--env-file ./$(APPENV) \
		-e AWS_DEFAULT_REGION \
		-e AWS_ACCESS_KEY_ID \
		-e AWS_SECRET_ACCESS_KEY \
		-p 9096:9096 \
		--rm \
		quay.io/opsee/compost:latest

.PHONY: docker run migrate clean all
