. $PSScriptRoot/imports.ps1

Describe -Name "Remove-User" {
    BeforeEach {
        $User = PSC8y\New-TestUser

    }

    It "Delete a user" {
        $Response = PSC8y\Remove-User -Id $User.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {

    }
}

