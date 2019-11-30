. $PSScriptRoot/imports.ps1

Describe -Name "Get-RoleReferenceCollectionFromGroup" {
    BeforeEach {
        $Group = Get-GroupByName -Name "business"

    }

    It "Get a list of role references for a user group" {
        $Response = PSC8y\Get-RoleReferenceCollectionFromGroup -GroupId $Group.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

