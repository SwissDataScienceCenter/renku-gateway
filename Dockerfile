FROM python:3.7-slim

COPY ./ /code

WORKDIR /code

RUN apt-get update && apt-get install -y gcc && \
    pip install --upgrade pip && \
    pip install pipenv && \
    pipenv install --system --deploy

# NOTE: You might be tempted to change the number of worker processes
#       here. Don't do it unless the implementation in app/auth/web.py
#       has been refactored (and this comment removed...)!
#       Currently, the poor mans approach to session handling and the
#       signaling through blinker do only work within one process!

CMD ["hypercorn", "-b", "0.0.0.0:5000", "run:app.app"]

EXPOSE 5000
