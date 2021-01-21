FROM golang:1.12.5 as build

COPY package.json ./package.json

RUN curl -sL https://deb.nodesource.com/setup_6.x | bash -

RUN apt-get update && apt-get install -y \
    python-dev \
    zip \
    jq


RUN curl -O https://bootstrap.pypa.io/get-pip.py

RUN python get-pip.py
RUN pip install awscli
