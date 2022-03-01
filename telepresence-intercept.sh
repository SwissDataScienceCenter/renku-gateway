#!/bin/bash

if [[ -z $1 ]] || [[ -z $2 ]]
then
    echo "Script to launch the telepresence intercept."
    echo "Example usage:"
    echo "telepresence-intercept.sh <namespace> <gateway-auth-service-name>"
    exit
fi

telepresence intercept -n $1 $2 --port 5000:http --mount=true -- ./telepresence-configure.sh
