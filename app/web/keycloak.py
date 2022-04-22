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
"""Web auth routes."""
import secrets
from urllib.parse import urljoin

from flask import (
    Blueprint,
    current_app,
    redirect,
    render_template,
    request,
    session,
    url_for,
)

from ..common.oauth_provider_app import KeycloakProviderApp
from ..common.utils import handle_login_request, handle_token_request, KC_SUFFIX

blueprint = Blueprint("web_auth", __name__, url_prefix="auth")

SCOPE = ["profile", "email", "openid", "offline_access"]

LOGIN_SEQUENCE = ("web_auth.login", "gitlab_auth.login")


@blueprint.route("/login/next")
def login_next():
    session["login_seq"] += 1
    if session["login_seq"] < len(LOGIN_SEQUENCE):
        next_login = LOGIN_SEQUENCE[session["login_seq"]]
        return render_template(
            "redirect.html",
            redirect_url=urljoin(current_app.config["HOST_NAME"], url_for(next_login)),
        )
    else:
        redirect_url = session["redirect_url"]
        session.clear()
        return redirect(redirect_url)


@blueprint.route("/login")
def login():
    session.clear()
    session["id"] = secrets.token_hex(64)
    session["redirect_url"] = request.args.get("redirect_url")
    session["login_seq"] = 0

    provider_app = KeycloakProviderApp(
        client_id=current_app.config["OIDC_CLIENT_ID"],
        client_secret=current_app.config["OIDC_CLIENT_SECRET"],
        base_url=current_app.config["OIDC_ISSUER"],
    )
    return handle_login_request(
        provider_app,
        urljoin(current_app.config["HOST_NAME"], url_for("web_auth.token")),
        KC_SUFFIX,
        SCOPE,
    )


@blueprint.route("/token")
def token():
    return handle_token_request(request, KC_SUFFIX)


@blueprint.route("/user-profile")
def user_profile():
    return current_app.make_response(
        redirect("{}/account?referrer=renku".format(current_app.config["OIDC_ISSUER"]))
    )


@blueprint.route("/logout")
def logout():

    logout_pages = [
        f"{current_app.config['OIDC_ISSUER']}/protocol/openid-connect/logout"
    ]
    if current_app.config["LOGOUT_GITLAB_UPON_RENKU_LOGOUT"]:
        logout_pages.append(
            urljoin(current_app.config["HOST_NAME"], url_for("gitlab_auth.logout"))
        )

    return render_template(
        "redirect_logout.html",
        redirect_url=request.args.get("redirect_url"),
        logout_pages=logout_pages,
        len=len(logout_pages),
    )
