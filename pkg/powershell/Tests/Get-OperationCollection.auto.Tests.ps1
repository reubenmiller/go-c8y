. $PSScriptRoot/imports.ps1

Describe -Name "Get-OperationCollection" {
    BeforeEach {
        $TestAgent = PSC8y\New-TestAgent
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Get a list of pending operations" {
        $Response = PSC8y\Get-OperationCollection -Status PENDING
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get a list of pending operations for a given agent and all of its child devices" {
        $Response = PSC8y\Get-OperationCollection -Agent $TestAgent.id -Status PENDING
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get a list of pending operations for a device" {
        $Response = PSC8y\Get-OperationCollection -Device $TestDevice.id -Status PENDING
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestAgent.id) {
            PSC8y\Remove-ManagedObject -Id $TestAgent.id -ErrorAction SilentlyContinue
        }
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

