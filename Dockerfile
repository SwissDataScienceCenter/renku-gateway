FROM python:3.7-slim

RUN pip install --upgrade pip==22.1.2 && \
    pip install poetry && \
    addgroup renku --gid 1000 && \
    adduser renku --uid 1000 --gid 1000

COPY pyproject.toml poetry.lock /code/
WORKDIR /code

RUN poetry config virtualenvs.create false && \
    poetry install

COPY ./ /code

USER 1000:1000

CMD ["gunicorn", "-b", "0.0.0.0:5000", "app:app",  "-k", "gevent"]

EXPOSE 5000
