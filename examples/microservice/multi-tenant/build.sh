#!/bin/sh
image=multi-tenant-demo
tag=1.0.0

docker build -t "${image}:${tag}" .
docker save "${image}:${tag}" > image.tar

zip "${image}.zip" "cumulocity.json" "image.tar"

rm image.tar
