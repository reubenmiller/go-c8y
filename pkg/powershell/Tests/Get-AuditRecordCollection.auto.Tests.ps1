. $PSScriptRoot/imports.ps1

Describe -Name "Get-AuditRecordCollection" {
    BeforeEach {
        $Device = New-TestDevice
        Remove-ManagedObject -Id $Device.id

    }

    It "Get a list of audit records" {
        $Response = PSC8y\Get-AuditRecordCollection -PageSize 100
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Get a list of audit records" {
        $Response = PSC8y\Get-AuditRecordCollection -Source $Device.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        Remove-ManagedObject -Id $Device.id

    }
}

