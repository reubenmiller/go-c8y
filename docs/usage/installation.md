# Installation

The c8y application is packaged as a single binary, which support the following Operating Systems.

* Windows 7+
* MacOS
* *nix

In addition to the single binary, there is also a Powershell Module called `PSc8y`.

## Bash

1. Download the file from the github releases page

    ```sh
    curl https://github.com/reubenmiller/go-c8y/releases/latest/download/c8y.macos --output ~/c8y
    ```

2. Copy the file to a path inside your `$PATH` setting

    ```sh
    chmod +x ./c8y
    cp ~/c8y /usr/local/bin/
    ```

3. Check if the c8y binary is now callable from anywere by checking the version

    ```sh
    c8y version
    ```

4. Add bash completions
    ```sh
    c8y completion bash > ~/.c8y.completions.sh

    echo "source ~/.c8y.completions.sh" >> ~/.bash_profile
    source ~/.bash_profile
    ```

## Powershell

1. Install `PSc8y` module from PSGallery using the following commands

    ```powershell
    Install-Module PSc8y -AllowPrerelease
    Import-Module PSc8y
    ```

    **Note:**

    Powershell 5.1 onwards is required. Powershell Core (pwsh) is also supported, so it can be run on Windows, MacOS and *nix systems.

2. You will have to import it again everytime you start a new powershell console. You can also add it into your powershell profile `Import-Module PSc8y` so it loaded automatically.

### Module installation Problems on (MacOS and Linux)

**Note:**

If you're having problems with installing PowerShell modules using `Install-Module`, then try updating to the latest PowerShellGet module directly from the source using the following:

1. Clone the PowerShellGet repository

    ```sh
    git clone https://github.com/PowerShell/PowerShellGet
    ```

1. Import the module from the cloned directory

    ```sh
    Remove-Module PowerShellGet -ErrorAction SilentlyContinue
    Import-Module PowerShellGet/src/PowerShellGet -Force
    ```
