. $PSScriptRoot/imports.ps1

Describe -Name "Update-AlarmCollection" {
    BeforeEach {
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Update the status of all active alarms on a device to ACKNOWLEDGED" {
        $Response = PSC8y\Update-AlarmCollection -Device $TestDevice.id -Status ACTIVE -NewStatus ACKNOWLEDGED
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

