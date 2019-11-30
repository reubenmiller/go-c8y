. $PSScriptRoot/imports.ps1

Describe -Name "Get-GroupByName" {
    BeforeEach {
        $Group = New-TestGroup

    }

    It "Get user group by its name" {
        $Response = PSC8y\Get-GroupByName -Name $Group.name
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Remove-Group -Id $Group.id

    }
}

