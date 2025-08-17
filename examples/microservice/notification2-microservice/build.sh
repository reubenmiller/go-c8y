#!/usr/bin/env bash
set -e
image=myevent-worker
tag=1.0.0-SNAPSHOT

dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
(
    cd "$dir/../../../"
    docker buildx build --load --progress plain --platform=linux/amd64 -t "${image}:${tag}" -f "$dir/Dockerfile" .
)

docker save "${image}:${tag}" > image.tar

zip "${image}.zip" "cumulocity.json" "image.tar"

rm image.tar
