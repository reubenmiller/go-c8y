. $PSScriptRoot/imports.ps1

Describe -Name "Get-Group" {
    BeforeEach {
        $Group = New-TestGroup

    }

    It "Get a user group" {
        $Response = PSC8y\Get-Group -Id $Group.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Remove-Group -Id $Group.id

    }
}

