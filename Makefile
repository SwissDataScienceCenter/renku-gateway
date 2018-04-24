DOCKER_REPOSITORY=rengahub

IMAGE=incubator-proxy

start:
	@docker pull ${DOCKER_REPOSITORY}/${IMAGE}
	@docker run -p 5000:5000 ${DOCKER_REPOSITORY}/${IMAGE}
