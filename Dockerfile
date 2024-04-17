FROM python:3.11-slim-bookworm as builder
WORKDIR /code
RUN pip install --upgrade pip && \
    pip install poetry && \
    virtualenv .venv
COPY pyproject.toml poetry.lock ./
RUN poetry install --without dev --no-root
COPY app ./app
RUN poetry install --without dev

FROM python:3.11-slim-bookworm
WORKDIR /code
ENV TINI_VERSION v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
RUN chmod a+x /tini && \
    addgroup renku --gid 1000 && \
    adduser renku --uid 1000 --gid 1000
COPY --chown=1000:1000 --from=builder /code/.venv .venv
COPY --chown=1000:1000 --from=builder /code/app app
USER 1000:1000
ENTRYPOINT [ "/tini", "-g", "--", "./.venv/bin/gunicorn", "-b", "0.0.0.0:5000", "app:app" ]
EXPOSE 5000
