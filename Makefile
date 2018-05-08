DOCKER_REPOSITORY?=rengahub

IMAGE=incubator-proxy

all:
	@echo "All"
	@docker build -t ${IMAGE} .
	@docker tag ${IMAGE} ${DOCKER_REPOSITORY}/${IMAGE}


build:
	@echo "Build"
	@docker build -t ${IMAGE} .
	@docker tag ${IMAGE} ${DOCKER_REPOSITORY}/${IMAGE}
	@docker push ${DOCKER_REPOSITORY}/${IMAGE}


start:
	@echo "Start"
	@docker pull ${DOCKER_REPOSITORY}/${IMAGE}
    @docker run -p 5000:5000 ${DOCKER_REPOSITORY}/${IMAGE}

build-dev:
	@echo "Build-dev"
	@docker build -t ${IMAGE} .
	@docker tag ${IMAGE} ${DOCKER_REPOSITORY}/${IMAGE}:development
	@docker push ${DOCKER_REPOSITORY}/${IMAGE}:development

