. $PSScriptRoot/imports.ps1

Describe -Name "Remove-ExternalId" {
    BeforeEach {
        $Device = PSC8y\New-TestDevice
        $ExternalID = PSC8y\New-ExternalID -Device $Device.id -Type "my_SerialNumber" -Name "myserialnumber2"

    }

    It "Delete external identity" {
        $Response = PSC8y\Remove-ExternalId -Type "my_SerialNumber" -Name "myserialnumber2"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Remove-ManagedObject -Id $Device.id

    }
}

