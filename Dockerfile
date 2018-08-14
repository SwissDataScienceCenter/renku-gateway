FROM python:3.6-slim

RUN set -e \
  ; apt-get update \
  ; apt-get install -y gcc libffi-dev libssl-dev \
  ; pip install --upgrade pip \
  ; rm -r /root/.cache \
  ;

COPY requirements.txt /app/requirements.txt
RUN pip install -r /app/requirements.txt


COPY run.py .
COPY app /app
COPY endpoints.json .

# NOTE: You might be tempted to change the number of worker processes
#       here. Don't do it unless the implementation in app/auth/web.py
#       has been refactored (and this comment removed...)!
#       Currently, the poor mans approach to session handling and the
#       signaling through blinker do only work within one process!

CMD ["gunicorn", "-b 0.0.0.0:5000", "run:app.app",  "-k gevent"]

EXPOSE 5000
