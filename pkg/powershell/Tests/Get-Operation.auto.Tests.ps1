. $PSScriptRoot/imports.ps1

Describe -Name "Get-Operation" {
    BeforeEach {
        $TestOperation = PSC8y\New-TestOperation

    }

    It "Get operation by id" {
        $Response = PSC8y\Get-Operation -Id $TestOperation.id
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestOperation.deviceId) {
            PSC8y\Remove-ManagedObject -Id $TestOperation.deviceId -ErrorAction SilentlyContinue
        }

    }
}

