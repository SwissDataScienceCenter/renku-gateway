FROM python:3.7-slim

RUN apt-get update && apt-get install -y gcc && \
    pip install --upgrade pip==20.1.1 && \
    pip install pipenv

COPY Pipfile* /code/
WORKDIR /code

RUN  pipenv install --system --deploy

COPY ./ /code

CMD ["gunicorn", "-b", "0.0.0.0:5000", "app:app",  "-k", "gevent"]

EXPOSE 5000
