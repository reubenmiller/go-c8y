. $PSScriptRoot/imports.ps1

Describe -Name "Get-OperationCollection" {
    BeforeEach {
        $Agent = New-TestAgent
        $Operation1 = PSC8y\New-TestOperation -Device $Agent.id
        $Device = New-TestDevice
        New-ChildDeviceReference -Device $Agent.id -NewChild $Device.id
        $Operation1 = PSC8y\New-TestOperation -Device $Device.id

    }

    It "Get a list of pending operations" {
        $Response = PSC8y\Get-OperationCollection -Status PENDING
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get a list of pending operations for a given agent and all of its child devices" {
        $Response = PSC8y\Get-OperationCollection -Agent $Agent.id -Status PENDING
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get a list of pending operations for a device" {
        $Response = PSC8y\Get-OperationCollection -Device $Device.id -Status PENDING
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        Remove-ManagedObject -id $Agent.id
        Remove-ManagedObject -id $Device.id

    }
}

