. $PSScriptRoot/imports.ps1

Describe -Name "Get-ExternalIDCollection" {
    BeforeEach {
        $Device = New-TestDevice
        $ExtName = New-RandomString -Prefix "IMEI"
        $ExternalID = PSC8y\New-ExternalID -Device $Device.id -Type "my_SerialNumber" -Name "$ExtName"

    }

    It "Get a list of external ids" {
        $Response = PSC8y\Get-ExternalIdCollection -Device $Device.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $Device.id

    }
}

