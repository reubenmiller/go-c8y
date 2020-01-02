
# Filtering (client side)

The response returned from the platform can also be filtered on the client side by using the `--filter` argument.

### Filtering application with name that start with "co*"

```sh
applications list --pageSize 100 --filter "name like co*"
```

## Selecting properties

In cases where you don't want all of the properties being returned in the json object, then a list of property names can be given using the `--select` argument.

Nested properties are also supported.

### Only return the "id", "name" and "owner.tenant" properties for each application

```sh
c8ycli applications list --pageSize 10 --select "id,name,owner.tenant"
```

### Only return the "id" and "name" properties for each application

```sh
c8ycli applications list --pageSize 10 --select "owner.tenant"
```

# Formatting data

```sh
c8ycli applications get --id cockpit --format "id"
```

```sh
7
```

**Note**

The formatting argument only supports one value. If the commands returns more than 1 result, then only the first result will be used.
