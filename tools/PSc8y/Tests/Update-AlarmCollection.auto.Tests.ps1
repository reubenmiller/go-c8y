. $PSScriptRoot/imports.ps1

Describe -Name "Update-AlarmCollection" {
    BeforeEach {
        $Device = PSC8y\New-TestDevice
        $Alarm = PSC8y\New-Alarm -Device $Device.id -Type c8y_TestAlarm -Time "-0s" -Text "Test alarm" -Severity MAJOR

    }

    It "Update the status of all active alarms on a device to ACKNOWLEDGED" {
        $Response = PSC8y\Update-AlarmCollection -Device $Device.id -Status ACTIVE -NewStatus ACKNOWLEDGED
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $Device.id

    }
}

