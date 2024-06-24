#!/bin/bash

# Tags the built image with the commit hash and pushes to Docker Hub.
docker tag $IMAGE_NAME $DOCKER_REPO:$SOURCE_COMMIT
docker push $DOCKER_REPO:$SOURCE_COMMIT