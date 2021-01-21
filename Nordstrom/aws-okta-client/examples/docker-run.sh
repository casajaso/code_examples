#!/usr/bin/env bash
 
docker run -it \
-p 8888:8888 \                              #not required
-v $PWD:$PWD \                              #not required
-w $PWD \                                   #not required
-v $HOME/ \                                 #not required
-e AWS_DEFAULT_REGION="us-west-2"  \        #not required
-e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
-e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
-e AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN \
-e AWS_SECURITY_TOKEN=$AWS_SECURITY_TOKEN \
gitlab-registry.nordstrom.com/path/to/docker/image:latest