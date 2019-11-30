. $PSScriptRoot/imports.ps1

Describe -Name "Get-Operation" {
    BeforeEach {
        $TestOperation = PSC8y\New-TestOperation

    }

    It "Get operation by id" {
        $Response = PSC8y\Get-Operation -Id $TestOperation.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        if ($TestOperation.deviceId) {
            PSC8y\Remove-ManagedObject -Id $TestOperation.deviceId -ErrorAction SilentlyContinue
        }

    }
}

