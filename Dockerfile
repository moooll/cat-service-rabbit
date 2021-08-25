FROM golang:1.16-buster

WORKDIR /cat-service-mongo

ENV MONGO_URI=mongodb://mongo:27017
ENV RABBIT_URI=amqp://guest:guest@rabbit:5672
ENV REDIS_URI=redis:6379

COPY . .
COPY ./wait-for ./wait-for
RUN ["chmod", "+x", "./wait-for"]

RUN apt-get update && apt-get install -y netcat

RUN go mod vendor
RUN go build -o cat

