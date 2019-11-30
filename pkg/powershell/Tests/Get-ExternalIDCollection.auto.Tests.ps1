. $PSScriptRoot/imports.ps1

Describe -Name "Get-ExternalIDCollection" {
    BeforeEach {

    }

    It "Get a list of external ids" {
        $Response = PSC8y\Get-ExternalIdCollection
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

