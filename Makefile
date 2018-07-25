# -*- coding: utf-8 -*-
#
# Copyright 2018 - Swiss Data Science Center (SDSC)
# A partnership between École Polytechnique Fédérale de Lausanne (EPFL) and
# Eidgenössische Technische Hochschule Zürich (ETHZ).
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DOCKER_REPOSITORY?=renku/
IMAGE?=renku-gateway

DOCKER_LABEL?=$(shell git branch 2> /dev/null | sed -e '/^[^*]/d' -e 's/^* //')
ifeq ($(DOCKER_LABEL), master)
	DOCKER_LABEL=latest
endif

GIT_MASTER_HEAD_SHA:=$(shell git rev-parse --short=7 --verify HEAD)

# Note that this is the default target executed when typing 'make'
tag: build
	@echo "Tagging image: docker tag ${DOCKER_REPOSITORY}${IMAGE}:${GIT_MASTER_HEAD_SHA} ${DOCKER_REPOSITORY}${IMAGE}:${DOCKER_LABEL}"
	@docker tag ${DOCKER_REPOSITORY}${IMAGE}:${GIT_MASTER_HEAD_SHA} ${DOCKER_REPOSITORY}${IMAGE}:${DOCKER_LABEL}

build:
	@echo "Building image: docker build -t ${DOCKER_REPOSITORY}${IMAGE}:${GIT_MASTER_HEAD_SHA} ."
	@docker build -t ${DOCKER_REPOSITORY}${IMAGE}:${GIT_MASTER_HEAD_SHA} .

push: tag
	@echo "Pushing image image: docker push ${DOCKER_REPOSITORY}${IMAGE}:${DOCKER_LABEL}"
	@docker push ${DOCKER_REPOSITORY}${IMAGE}:${DOCKER_LABEL}

start:
	@echo "Start"
	@docker pull ${DOCKER_REPOSITORY}${IMAGE}
    @docker run -p 5000:5000 ${DOCKER_REPOSITORY}${IMAGE}

dev-docker:
	@echo "Running development server to develop against renku running inside docker"
	FLASK_DEBUG=1 HOST_NAME=http://localhost:5000 python run.py

dev-minikube:
	@echo "Running development to develop against renku running inside minikube"
	FLASK_DEBUG=1 \
	HOST_NAME=http://localhost:5000 \
	RENKU_ENDPOINT=http://$(shell minikube ip) \
	GITLAB_URL=http://$(shell minikube ip)/gitlab \
	KEYCLOAK_URL=http://$(shell minikube ip) \
	GATEWAY_SERVICE_PREFIX=/api \
	python run.py

login:
	@echo "${DOCKER_PASSWORD}" | docker login -u="${DOCKER_USERNAME}" --password-stdin ${DOCKER_REGISTRY}
