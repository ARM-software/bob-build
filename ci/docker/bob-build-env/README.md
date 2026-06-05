# bob-build-env docker image

This docker image contains the required dependencies to run the build tests in ci, including
the new bazel based build tests.

# Creating the docker image

```
docker build -t bob-build-env ci/docker/bob-build-env/
docker tag bob-build-env:latest gpuddk--docker.artifactory.arm.com/gpuddk/bob-build-env:latest
```

# Pushing to Artifactory

```
export EMAIL=<your_email>
export ART_TOKEN=<artifactory_token>
docker login -u$EMAIL -p$ART_TOKEN gpuddk--docker.artifactory.arm.com
docker push gpuddk--docker.artifactory.arm.com/gpuddk/bob-build-env:latest
```
