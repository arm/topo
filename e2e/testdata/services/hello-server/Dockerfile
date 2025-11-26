FROM python:3.11-slim

ARG NAME=World

WORKDIR /app

COPY server.py .

ENV NAME=${NAME}
ENV PORT=8080

CMD ["python", "server.py"]
