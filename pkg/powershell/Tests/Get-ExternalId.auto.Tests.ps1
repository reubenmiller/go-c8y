. $PSScriptRoot/imports.ps1

Describe -Name "Get-ExternalId" {
    BeforeEach {
        $Device = PSC8y\New-TestDevice
        $ExternalID = PSC8y\New-ExternalID -Device $Device.id -Type "my_SerialNumber" -Name "myserialnumber"

    }

    It "Get external identity" {
        $Response = PSC8y\Get-ExternalId -Type "my_SerialNumber" -Name "myserialnumber"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        Remove-ManagedObject -Id $Device.id

    }
}

