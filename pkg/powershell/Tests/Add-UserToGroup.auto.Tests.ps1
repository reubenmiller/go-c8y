. $PSScriptRoot/imports.ps1

Describe -Name "Add-UserToGroup" {
    BeforeEach {
        $User = New-TestUser
        $Group = Get-GroupByName -Name "business"

    }

    It "Add a user to a user group" {
        $Response = PSC8y\Add-UserToGroup -GroupId $Group.id -UserId $User.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Remove-User -Id $User.id

    }
}

