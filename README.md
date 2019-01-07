## AWS Lambda Container Image Converter

This container image converter tool (img2lambda) repackages container images (such as Docker images) into AWS Lambda layers, and publishes them as new layer versions.

```
docker build -t lambda-php .

docker run lambda-php hello '{"name": "World"}'

docker run lambda-php goodbye '{"name": "World"}'

../../bin/img2lambda -i docker-daemon:lambda-php:latest
```

TODO:
* Utility for creating a function deployment package from a Docker image
* Support image types other than local Docker images ('docker-daemon' transport), where the layer format is tar. For example, layers directly from a Docker registry will be .tar.gz-formatted. OCI images can be either tar or tar.gz, based on the layer's media type.

## License Summary

This sample code is made available under a modified MIT license. See the LICENSE file.
