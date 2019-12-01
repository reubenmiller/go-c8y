. $PSScriptRoot/imports.ps1

Describe -Name "New-TenantOption" {
    BeforeEach {

    }

    It "Create a tenant option" {
        $Response = PSC8y\New-TenantOption -Category "c8y_cli_tests" -Key "option1" -Value "1"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Remove-TenantOption -Category "c8y_cli_tests" -Key "option1"

    }
}

