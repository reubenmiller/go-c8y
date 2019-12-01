. $PSScriptRoot/imports.ps1

Describe -Name "Get-TenantOption" {
    BeforeEach {
        New-TenantOption -Category "c8y_cli_tests" -Key "option2" -Value "2"

    }

    It "Get a tenant option" {
        $Response = PSC8y\Get-TenantOption -Category "c8y_cli_tests" -Key "option2"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Remove-TenantOption -Category "c8y_cli_tests" -Key "option2"

    }
}

