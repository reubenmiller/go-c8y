. $PSScriptRoot/imports.ps1

Describe -Name "Get-GroupCollection" {
    BeforeEach {

    }

    It "Get a list of user groups for the current tenant" {
        $Response = PSC8y\Get-GroupCollection
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

