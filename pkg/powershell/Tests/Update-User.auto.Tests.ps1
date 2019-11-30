. $PSScriptRoot/imports.ps1

Describe -Name "Update-User" {
    BeforeEach {
        $User = PSC8y\New-TestUser

    }

    It "Update a user" {
        $Response = PSC8y\Update-User -Id $User.id -FirstName "Simon"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

