. $PSScriptRoot/imports.ps1

Describe -Name "New-Group" {
    BeforeEach {

    }

    It "Create a user group" {
        $Response = PSC8y\New-Group -Name "customGroup1"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Get-GroupByName -Name "customGroup1" | Remove-Group

    }
}

