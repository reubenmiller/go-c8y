---
layout: default
category: Installation
order: 100
title: Bash
---

1. Download the file from the github releases page

    **Linux**

    ```sh
    curl https://github.com/reubenmiller/go-c8y/releases/latest/download/c8y.linux --output ~/c8y
    ```

    **MacOS**

    ```sh
    curl https://github.com/reubenmiller/go-c8y/releases/latest/download/c8y.macos --output ~/c8y
    ```

2. Copy the file to a path inside your `$PATH` variable

    ```sh
    chmod +x ~/c8y
    sudo cp ~/c8y /usr/local/bin/
    ```

3. Check if the c8y binary is now callable from anywere by checking the version

    ```sh
    c8y version
    ```

    **Response**

    ```plaintext
    Cumulocity command line tool
    v0.7.0-345-g164bcec -- master
    ```

4. Add bash completions
    ```sh
    echo "source ~/.c8y.completions.sh" >> ~/.bashrc
    source ~/.bashrc
    ```
