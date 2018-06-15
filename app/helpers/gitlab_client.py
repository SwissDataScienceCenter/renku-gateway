import requests
from flask import request
import logging
from .. import app

logger = logging.getLogger(__name__)

def get_readme(headers, projectid):
    readme_url = app.config['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/repository/files/README.md/raw?ref=master"
    logger.debug("Getting readme for project with {0}". format(projectid) )

    return requests.request(request.method, readme_url, headers=headers, data=request.data, stream=True, timeout=300)


def get_kus(headers, projectid):
    ku_url = app.config['GITLAB_URL'] + "/api/v4/projects/" + str(projectid) + "/issues?scope=all"
    logger.debug("Getting issues for project with id {0}". format(projectid) )

    return requests.request(request.method, ku_url, headers=headers, data=request.data, stream=True, timeout=300)
