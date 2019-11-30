. $PSScriptRoot/imports.ps1

Describe -Name "New-ExternalID" {
    BeforeEach {
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Get external identity" {
        $Response = PSC8y\New-ExternalID -Device $TestDevice.id -Type "my_SerialNumber" -Name "myserialnumber"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

