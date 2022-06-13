#!/bin/bash

echo -e "\nTo start/restart the flask development server, run the following command:\n"
echo -e "SSL_CERT_FILE=$TELEPRESENCE_ROOT$SSL_CERT_FILE \\
REQUESTS_CA_BUNDLE=$TELEPRESENCE_ROOT$REQUESTS_CA_BUNDLE \\
FLASK_DEBUG=1 \\
FLASK_APP=app:app \\
HOST_NAME=$HOST_NAME \\
OAUTHLIB_INSECURE_TRANSPORT=1 \\
poetry run flask run"
echo ""

$(echo $SHELL)
