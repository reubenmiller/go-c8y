---
layout: default
category: Installation
order: 200
title: Powershell
---

1. Install `PSc8y` module from PSGallery using the following commands

    ```powershell
    Install-Module PSc8y -AllowPrerelease
    Import-Module PSc8y
    ```

    **Note:**

    Powershell 5.1 onwards is required. Powershell Core (pwsh) is also supported, so it can be run on Windows, MacOS and *nix systems.

2. You will have to import it again everytime you start a new powershell console. You can also add it into your powershell profile `Import-Module PSc8y` so it is loaded automatically.
