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

