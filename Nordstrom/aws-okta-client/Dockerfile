### Docker Build container
### to run locally: 
# $ docker build Dockerfile . -t aws-okta
# $ docker run --rm -it -v "$PWD":/src/aws-okta -w /src/aws-okta -t aws-okta bash
# root:/src# make <build-options>

FROM golang:1.12-stretch as build

WORKDIR /home/container_user
COPY . src

RUN apt-get update && apt-get install --no-install-recommends -y \
    libusb-1.0-0-dev \
    ca-certificates \
    build-essential \
    python-dev \
    jq \
    git 
    
#RUN curl -O https://bootstrap.pypa.io/get-pip.py

#RUN python get-pip.py
#RUN pip install awscli
#RUN pip install gsutil
