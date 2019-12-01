. $PSScriptRoot/imports.ps1

Describe -Name "Remove-UserFromGroup" {
    BeforeEach {
        $User = New-TestUser
        $Group = Get-GroupByName -Name "business"
        Add-UserToGroup -GroupId $Group.id -UserId $User.id

    }

    It "Add a user to a user group" {
        $Response = PSC8y\Remove-UserFromGroup -GroupId $Group.id -UserId $User.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Remove-User -Id $User.id

    }
}

