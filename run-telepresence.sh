#!/bin/bash
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

set -e

CURRENT_CONTEXT=`kubectl config current-context`

# On Mac we can not use `pipenv run flask run` because of
# https://www.telepresence.io/reference/methods
FLASK_EXECUTABLE=`pipenv --venv`/bin/flask

if [[ $CURRENT_CONTEXT == 'minikube' ]]
then
  echo "Exchanging k8s deployments using the following context: ${CURRENT_CONTEXT}"
  SERVICE_NAME=renku-gateway-auth
  DEV_NAMESPACE=renku
else
  echo "You are going to exchange k8s deployments using the following context: ${CURRENT_CONTEXT}"
  read -p "Do you want to proceed? [y/n]"
  if [[ ! $REPLY =~ ^[Yy]$ ]]
  then
      exit 1
  fi

  if [[ ! $DEV_NAMESPACE ]]
  then
    read -p "enter your k8s namespace: "
    DEV_NAMESPACE=$REPLY
  fi
  SERVICE_NAME=${DEV_NAMESPACE}-renku-gateway-auth
fi


echo "================================================================================================================="
echo "Once telepresence has started, copy-paste the following command to start the development server:"
echo "FLASK_DEBUG=1 \
FLASK_APP=app:app \
HOST_NAME=\$HOST_NAME \
OAUTHLIB_INSECURE_TRANSPORT=1 \
${FLASK_EXECUTABLE} run"
echo ""
echo "Or use the following to run in the VS Code debugger:"
echo "FLASK_DEBUG=1 \
FLASK_APP=app:app \
HOST_NAME=\$HOST_NAME \
VSCODE_DEBUG=1 \
OAUTHLIB_INSECURE_TRANSPORT=1 \
${FLASK_EXECUTABLE} run --no-reload"
echo "================================================================================================================="


# The `inject-tcp` proxying switch helps when running multiple instances of telepresence but creates problems when
# suid bins need to run. This is a problem when running on Linux only.
# Reference: https://www.telepresence.io/reference/methods

if [[ "$OSTYPE" == "linux-gnu" ]]
then
  telepresence --swap-deployment ${SERVICE_NAME} --namespace ${DEV_NAMESPACE} --expose 5000 --run-shell
else
  telepresence --swap-deployment ${SERVICE_NAME} --namespace ${DEV_NAMESPACE} --method inject-tcp --expose 5000 --run-shell
fi
