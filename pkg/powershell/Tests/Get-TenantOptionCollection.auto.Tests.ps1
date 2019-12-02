. $PSScriptRoot/imports.ps1

Describe -Name "Get-TenantOptionCollection" {
    BeforeEach {

    }

    It "Get a list of tenant options" {
        $Response = PSC8y\Get-TenantOptionCollection
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {

    }
}

