# Getting started

## Sessions

Before getting started, you need to configure the Cumulocity session which detail which Cumulocity platform and authenication should be used for each of the commands/requests.

1. Create a new session

    ```sh
    c8ycli sessions create --host http://cumulocity.com --tenant "mytenant" --username "myuser@me.com"
    ```

    You will be prompted for your password. Alternatively you can also enter the password using the `--password` argument.

    You may also provide a more meaningful session name by specifying a `--name` argument.

2. Activate the session using the interactive session selector

    ```sh
    # bash
    export C8Y_SESSION=$( ./c8ycli sessions list )
    ```

3. Test your credentials by getting your current user information from the platform

    ```sh
    c8ycli users getCurrentUser
    ```

    **Note**

    If your credentials are incorrect, then you can update the session file stored in the `~/.cumulocity` directory

4. Now you're ready to go. You can get a list of available commands by using help menu

    ```sh
    c8ycli help
    ```

## Switching sessions

The sessions can be changed again by using the interactive session selector

```sh
export C8Y_SESSION=$( ./c8ycli sessions list )
```
