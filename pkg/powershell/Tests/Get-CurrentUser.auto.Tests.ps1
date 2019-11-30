. $PSScriptRoot/imports.ps1

Describe -Name "Get-CurrentUser" {
    BeforeEach {

    }

    It "Get the current user" {
        $Response = PSC8y\Get-CurrentUser
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

