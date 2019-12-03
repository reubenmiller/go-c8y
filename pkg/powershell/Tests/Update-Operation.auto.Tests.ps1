. $PSScriptRoot/imports.ps1

Describe -Name "Update-Operation" {
    BeforeEach {
        $TestOperation = PSC8y\New-TestOperation
        $Agent = PSC8y\New-TestAgent
        $Operation1 = PSC8y\New-TestOperation -Device $Agent.id
        $Operation2 = PSC8y\New-TestOperation -Device $Agent.id

    }

    It "Update an operation" {
        $Response = PSC8y\Update-Operation -Id $TestOperation.id -Status EXECUTING
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Update multiple operations" {
        $Response = PSC8y\Get-OperationCollection -Device $Agent.id -Status PENDING | Update-Operation -Status FAILED -FailureReason "manually cancelled"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestOperation.deviceId) {
            PSC8y\Remove-ManagedObject -Id $TestOperation.deviceId -ErrorAction SilentlyContinue
        }
        PSC8y\Remove-ManagedObject -Id $Agent.id

    }
}

