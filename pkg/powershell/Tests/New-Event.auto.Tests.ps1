. $PSScriptRoot/imports.ps1

Describe -Name "New-Event" {
    BeforeEach {
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Create a new event for a device" {
        $Response = PSC8y\New-Event -Device $TestDevice.id -Type c8y_TestAlarm -Time "-0s" -Text "Test event"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

