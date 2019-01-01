## AWS Lambda Container Image Converter

This container image converter tool (img2lambda) repackages container images (such as Docker images) into AWS Lambda layers, and publishes them as new layer versions.

```
docker build -t lambda-php .

docker run lambda-php hello '{"name": "World"}'

docker run lambda-php goodbye '{"name": "World"}'
```

## License Summary

This sample code is made available under a modified MIT license. See the LICENSE file.