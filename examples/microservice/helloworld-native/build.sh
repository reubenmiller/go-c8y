#!/bin/sh
image=helloworld
tag=1.0.0

docker build -t "${image}:${tag}" .
docker save "${image}:${tag}" > image.tar

zip "helloworld.zip" "cumulocity.json" "image.tar"

rm image.tar
