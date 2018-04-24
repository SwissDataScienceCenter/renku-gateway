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

CMD ["python3", "run.py"]

EXPOSE 5000
