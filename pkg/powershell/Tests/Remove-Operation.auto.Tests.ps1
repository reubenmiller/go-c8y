. $PSScriptRoot/imports.ps1

Describe -Name "Remove-Operation" {
    BeforeEach {
        $TestOperation = PSC8y\New-TestOperation
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Remove an operation" {
        $Response = PSC8y\Remove-Operation -Id $TestOperation.id
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Remove multiple operations" {
        $Response = PSC8y\Get-OperationCollection -Device $TestDevice.id -Status EXECUTING | Remove-Operation
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

