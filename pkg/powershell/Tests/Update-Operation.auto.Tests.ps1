. $PSScriptRoot/imports.ps1

Describe -Name "Update-Operation" {
    BeforeEach {
        $TestOperation = PSC8y\New-TestOperation
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Update an operation" {
        $Response = PSC8y\Update-Operation -Id $TestOperation.id -Status EXECUTING
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Update multiple operations" {
        $Response = PSC8y\Get-OperationCollection -Device $TestDevice.id -Status EXECUTING | Update-Operation -Status FAILED -FailureReason "manually cancelled"
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestOperation.deviceId) {
            PSC8y\Remove-ManagedObject -Id $TestOperation.deviceId -ErrorAction SilentlyContinue
        }
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

