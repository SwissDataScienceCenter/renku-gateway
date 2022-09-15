FROM python:3.7-slim as base
RUN apt-get update && \
    apt-get install -y curl tini && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd -g 1000 renku && \
    useradd -u 1000 -g 1000 -m renku
WORKDIR /home/renku/renku-gateway

FROM base as builder
ENV POETRY_HOME=/opt/poetry
ENV POETRY_VIRTUALENVS_IN_PROJECT=true
ENV POETRY_VIRTUALENVS_OPTIONS_NO_PIP=true
ENV POETRY_VIRTUALENVS_OPTIONS_NO_SETUPTOOLS=true
COPY poetry.lock pyproject.toml ./
RUN mkdir -p /opt/poetry && \
    curl -sSL https://install.python-poetry.org | python3 - && \
    /opt/poetry/bin/poetry install --only main --no-root

FROM base as runtime
USER 1000:1000
COPY --from=builder /home/renku/renku-gateway/.venv .venv
COPY app app
ENTRYPOINT ["tini", "-g", "--"]
CMD [".venv/bin/gunicorn", "-b", "0.0.0.0:5000", "app:app", "-k", "gevent"]
