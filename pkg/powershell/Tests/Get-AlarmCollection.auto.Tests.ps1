. $PSScriptRoot/imports.ps1

Describe -Name "Get-AlarmCollection" {
    BeforeEach {

    }

    It "Get alarms with the severity set to MAJOR" {
        $Response = PSC8y\Get-AlarmCollection -Severity MAJOR -PageSize 100
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get active alarms which occurred in the last 10 minutes" {
        $Response = PSC8y\Get-AlarmCollection -DateFrom "-10m" -Status ACTIVE
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {

    }
}

