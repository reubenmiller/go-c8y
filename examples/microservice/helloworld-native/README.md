# helloworld

An example golang Cumulocity microservice without any external dependencies.

The microservice has the following endpoints:
* `/health` returns json with the status `UP`

# Building

The following build script will produce a `helloworld.zip` file, which can be uploaded and deployed to Cumulocity as a microservice.

```sh
chmod +x build.sh
./build.sh
```
