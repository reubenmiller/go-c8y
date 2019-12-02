. $PSScriptRoot/imports.ps1

Describe -Name "Update-Alarm" {
    BeforeEach {
        $TestAlarm = PSC8y\New-TestAlarm
        $TestAlarm = PSC8y\New-TestAlarm

    }

    It "Acknowledge an existing alarm" {
        $Response = PSC8y\Update-Alarm -Id $TestAlarm.id -Status ACKNOWLEDGED
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Update severity of an existing alarm to CRITICAL" {
        $Response = PSC8y\Update-Alarm -Id $TestAlarm.id -Severity CRITICAL
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestAlarm.source.id) {
            PSC8y\Remove-ManagedObject -Id $TestAlarm.source.id -ErrorAction SilentlyContinue
        }
        if ($TestAlarm.source.id) {
            PSC8y\Remove-ManagedObject -Id $TestAlarm.source.id -ErrorAction SilentlyContinue
        }

    }
}

