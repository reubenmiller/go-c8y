. $PSScriptRoot/imports.ps1

Describe -Name "Get-GroupMembership" {
    BeforeEach {
        $Group = Get-GroupByName -Name "business"

    }

    It "List the users within a user group" {
        $Response = PSC8y\Get-GroupMembership -Id $Group.id
        $LASTEXITCODE | Should -Be 0
    }
    It "List the users within a user group (using pipeline)" {
        $Response = PSC8y\Get-GroupByName -Name "business" | Get-GroupMembership
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

