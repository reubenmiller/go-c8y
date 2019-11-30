. $PSScriptRoot/imports.ps1

Describe -Name "New-Alarm" {
    BeforeEach {
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Create a new alarm for device" {
        $Response = PSC8y\New-Alarm -Device $TestDevice.id -Type c8y_TestAlarm -Time "-0s" -Text "Test alarm" -Severity MAJOR
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

