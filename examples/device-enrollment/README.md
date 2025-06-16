# example: device-enrollment and MQTT service client

A multi-command utility that can register a device with Cumulocity using the Cumulocity Certificate Authority feature, which will request a device certificate from Cumulocity. A registration URL and QR Code is printed on the console which the user can either click or scan to then registration the device.

Once the certificate has been created, another command can be used to subscribe to the Cumulocity mqtt-service.

## Pre-requisites

The following pre-requisites must be met in order to use this example.

* Cumulocity certificate-authority feature is enabled, and a CA certificate has been created
* The mqtt-service microservice is subscribed to your tenant


## Using it

1. Set the `C8Y_HOST` environment variable of the Cumulocity tenant you wish to use

    ```sh
    export C8Y_HOST=example.c8y.io
    
    ```

1. Enroll the device and specify the external ID of the device to be enrolled

    ```sh
    go run main.go enroll --device-id mydevice01
    ```

    Click on the registration URL, sign-in with your credentials, and then confirm the registration request.

    The command will exit once it has downloaded the certificate from Cumulocity. The device certificate is saved in the local folder, `device.key` (private key) and the `device.crt` (public certificate). 

1. Subscribe to the Cumulocity mqtt-service using the above device certificate

    ```sh
    go run main.go subscribe -t hello --duration 10s
    ```

1. You can publish data using

    ```sh
    go run main.go publish -t hello --payload world
    ```
