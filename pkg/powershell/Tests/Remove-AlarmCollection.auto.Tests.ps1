. $PSScriptRoot/imports.ps1

Describe -Name "Remove-AlarmCollection" {
    BeforeEach {
        $TestDevice = PSC8y\New-TestDevice
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Remove alarms on the device with the severity set to MAJOR" {
        $Response = PSC8y\Remove-AlarmCollection -Device $TestDevice.id -Severity MAJOR
        $LASTEXITCODE | Should -Be 0
    }
    It "Remove alarms on the device which are active and created in the last 10 minutes" {
        $Response = PSC8y\Remove-AlarmCollection -Device $TestDevice.id -DateFrom "-10m" -Status ACTIVE
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

