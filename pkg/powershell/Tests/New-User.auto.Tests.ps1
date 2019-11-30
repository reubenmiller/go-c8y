. $PSScriptRoot/imports.ps1

Describe -Name "New-User" {
    BeforeEach {
        $NewPassword = [guid]::NewGuid().Guid.Substring(1,10)

    }

    It "Create a user" {
        $Response = PSC8y\New-user -Username "testuser1" -Password "$NewPassword"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Get-UserByName -Name "testuser1" | Remove-User

    }
}

