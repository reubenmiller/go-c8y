Function New-TestDevice {
    [cmdletbinding(
        SupportsShouldProcess = $true,
        ConfirmImpact = "None"
    )]
    Param(
        [Parameter(
            Mandatory = $false,
            Position = 0
        )]
        [string] $Name = "testdevice",

        [switch] $AsAgent
    )
    $Data = @{
        c8y_IsDevice = @{}
    }
    if ($AsAgent) {
        $Data.com_cumulocity_model_Agent = @{}
    }
    $DeviceName = New-RandomString -Prefix "${Name}_"
    $TestDevice = PSC8y\New-ManagedObject `
        -Name $DeviceName `
        -Data $Data `
        -Force

    $TestDevice
}