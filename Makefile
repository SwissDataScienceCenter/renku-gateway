DOCKER_REPOSITORY?=rengahub/

IMAGE=incubator-proxy


start:
	@docker pull ${DOCKER_REPOSITORY}/${IMAGE}

# @docker run -p 5000:5000 ${DOCKER_REPOSITORY}/${IMAGE}


build:
	@docker build -t ${IMAGE} .
	@docker tag ${IMAGE} ${DOCKER_REPOSITORY}/${IMAGE}
	@docker push ${DOCKER_REPOSITORY}/${IMAGE}


build-dev:
	@docker build -t ${IMAGE} .
	@docker tag ${IMAGE} ${DOCKER_REPOSITORY}/${IMAGE}:development
	@docker push ${DOCKER_REPOSITORY}/${IMAGE}:development

