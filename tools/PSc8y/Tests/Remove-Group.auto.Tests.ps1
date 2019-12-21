. $PSScriptRoot/imports.ps1

Describe -Name "Remove-Group" {
    BeforeEach {
        $Group = New-TestGroup

    }

    It "Delete a user group" {
        $Response = PSC8y\Remove-Group -Id $Group.id
        $LASTEXITCODE | Should -Be 0
    }
    It "Delete a user group (using pipeline)" {
        $Response = PSC8y\Get-GroupByName -Name $Group.name | Remove-Group
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

