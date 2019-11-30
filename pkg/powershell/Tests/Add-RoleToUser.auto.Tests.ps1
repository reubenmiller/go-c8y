. $PSScriptRoot/imports.ps1

Describe -Name "Add-RoleToUser" {
    BeforeEach {
        $User = PSC8y\New-TestUser

    }

    It "Get a role (ROLE_ALARM_READ) to a user" {
        $Response = PSC8y\Add-RoleToUser -Username $User.id -Role "ROLE_ALARM_READ"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        PSC8y\Remove-User -Id $User.id

    }
}

