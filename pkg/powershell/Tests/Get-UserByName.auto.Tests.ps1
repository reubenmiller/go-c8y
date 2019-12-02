. $PSScriptRoot/imports.ps1

Describe -Name "Get-UserByName" {
    BeforeEach {
        $User = PSC8y\New-TestUser

    }

    It "Get a user by name" {
        $Response = PSC8y\Get-UserByName -Name $User.userName
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        Remove-User -Id $User.id

    }
}

