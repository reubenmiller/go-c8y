. $PSScriptRoot/imports.ps1

Describe -Name "Get-UserCollection#" {
    BeforeEach {

    }

    It "Get a list of users" {
        $Response = PSC8y\Get-UserCollection
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

