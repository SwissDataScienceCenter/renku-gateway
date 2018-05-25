DOCKER_REPOSITORY?=renku
PLATFORM_VERSION?=master

IMAGE=incubator-proxy

all:
	@echo "All"
	@echo "Platform version: " ${PLATFORM_VERSION}
	@docker build -t ${DOCKER_REPOSITORY}/${IMAGE}:${PLATFORM_VERSION} .


build:
	@echo "Build"
	@docker build -t ${IMAGE} .
	@docker tag ${IMAGE} ${DOCKER_REPOSITORY}/${IMAGE}
	@docker push ${DOCKER_REPOSITORY}/${IMAGE}


start:
	@echo "Start"
	@docker pull ${DOCKER_REPOSITORY}/${IMAGE}
    @docker run -p 5000:5000 ${DOCKER_REPOSITORY}/${IMAGE}

dev:
	@echo "Run-dev"
	FLASK_DEBUG=1 HOST_NAME=http://localhost:5000 python run.py

