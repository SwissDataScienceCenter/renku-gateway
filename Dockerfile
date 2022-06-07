FROM python:3.7-slim

RUN apt-get update && apt-get install -y gcc && \
    pip install --upgrade pip==20.1.1 && \
    pip install poetry

COPY pyproject.toml poetry.lock /code/
WORKDIR /code

RUN poetry config virtualenvs.create false && \
    poetry install

COPY ./ /code

USER 1000:1000

CMD ["gunicorn", "-b", "0.0.0.0:5000", "app:app",  "-k", "gevent"]

EXPOSE 5000
