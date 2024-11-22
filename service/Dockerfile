FROM docker-private.infra.cloudera.com/cloudera_base/hardened/cloudera-python:3.10
COPY ./app/requirements.txt /app/
WORKDIR /app
RUN pip install -r requirements.txt

COPY ./app /app
WORKDIR /app

CMD [ "fastapi", "run" ]
