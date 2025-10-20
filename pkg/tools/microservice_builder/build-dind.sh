#!/bin/sh
set -e

CONTEXT="."
IMAGE_NAME=""
DOCKERFILE_PATH="Dockerfile"
OUTPUT_IMAGE="image.tar"
DIND_NAME="docker"
DIND_IMAGE="${DIND_IMAGE:-docker:28-dind}"

while [ $# -gt 0 ]; do
    case "$1" in
        --context)
            CONTEXT="$2"
            shift
            ;;
        -f)
            DOCKERFILE_PATH="$2"
            shift
            ;;
        -t)
            IMAGE_NAME="$2"
            shift
            ;;
        --output)
            OUTPUT_IMAGE="$2"
            shift
            ;;
    esac
    shift
done


# Check if docker-in-docker is running
if ! docker inspect "$DIND_NAME" >/dev/null 2>&1; then
    # TODO: add support for proxy detection
    echo "Starting up a docker-in-docker instance" >&2
    docker run \
        --env HTTP_PROXY="${HTTP_PROXY:-}" \
        --env HTTPS_PROXY="${HTTPS_PROXY:-}" \
        --env http_proxy="${http_proxy:-}" \
        --env https_proxy="${https_proxy:-}" \
        --privileged \
        --name "$DIND_NAME" \
        -d \
        docker:28-dind
    
    ATTEMPT=1
    while ! docker exec "$DIND_NAME" docker ps >/dev/null 2>&1; do
        echo "Waiting for docker-in-docker to be ready" >&2
        sleep 1
        ATTEMPT=$((ATTEMPT + 1))
        if [ "$ATTEMPT" -gt 10 ]; then
            echo "docker-in-docker container (name=$DIND_NAME) failed to start in time" >&2
            exit 1
        fi
    done
fi

rm -f "$OUTPUT_IMAGE"


BUILD_DIR="/build/$IMAGE_NAME"

if [ -z "$IMAGE_TAR" ]; then
    IMAGE_TAR="/build/${IMAGE_NAME}.tar"
fi

dind_docker() {
    if [ -t 1 ]; then
        docker exec -it "$DIND_NAME" "$@"
    else
        docker exec "$DIND_NAME" "$@"
    fi
}

dind_docker sh -c "rm -rf '${BUILD_DIR}' && mkdir -p '${BUILD_DIR}'"
docker cp "$DOCKERFILE_PATH" "${DIND_NAME}:${BUILD_DIR}/Dockerfile"
docker cp "$CONTEXT" "${DIND_NAME}:${BUILD_DIR}/context"
dind_docker docker run --privileged --rm tonistiigi/binfmt --install all

# Option 1 (working)
dind_docker docker buildx build -t "$IMAGE_NAME" --load --platform linux/amd64 -f "$BUILD_DIR/Dockerfile" "$BUILD_DIR/context"
dind_docker docker save "$IMAGE_NAME" --platform linux/amd64 -o "$IMAGE_TAR"

# Option 2: (not working)
# Note: using --output requires the new containerd image store to be used, however this results in incompatible images
# It results in teh following error: "Docker exporter is not supported for the docker driver"
# dind_docker docker buildx build --output "type=docker,dest=$IMAGE_TAR,name=$IMAGE_NAME" --platform linux/amd64 -f "$BUILD_DIR/Dockerfile" "$BUILD_DIR/context"

docker cp "${DIND_NAME}:${IMAGE_TAR}" "$OUTPUT_IMAGE"
