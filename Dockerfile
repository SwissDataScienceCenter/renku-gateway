FROM python:3.6-slim

RUN apt-get update
RUN apt-get install -y gcc libffi-dev libssl-dev
RUN pip install --upgrade pip


COPY run.py .
COPY app /app

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r /app/requirements.txt

RUN file="$(ls -1)" && echo $file



WORKDIR /
ENTRYPOINT ["python3"]
CMD ["run.py"]

EXPOSE 5000
