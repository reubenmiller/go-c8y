. $PSScriptRoot/imports.ps1

Describe -Name "Get-EventCollection" {
    BeforeEach {
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Get events with type 'my_CustomType' that were created in the last 10 days" {
        $Response = PSC8y\Get-EventCollection -Type my_CustomType -DateFrom "-10d"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get events from a device" {
        $Response = PSC8y\Get-EventCollection -Device $TestDevice.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

