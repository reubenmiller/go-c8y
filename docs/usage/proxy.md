# Proxy (behind a corporate proxy)

c8ycli has first class support for corporate proxies.

## Setting using environment variables

```sh
# bash
export HTTP_PROXY=http://10.0.0.1:8080
export NO_PROXY=localhost,127.0.0.1,edge01.server


# Powershell
$env:HTTP_PROXY = "http://10.0.0.1:8080"
$env:NO_PROXY = "localhost,127.0.0.1,edge01.server"
```

## Overriding proxy environment settings for individual commands

The proxy environment variables can be ignore for individual commands by using the `--noProxy` option.


```sh
c8ycli devices list --noProxy
```

Or, an alternative proxy can be used by setting the `--proxy` argument

```sh
c8ycli devices list --proxy "http://10.0.0.1:8080
```
