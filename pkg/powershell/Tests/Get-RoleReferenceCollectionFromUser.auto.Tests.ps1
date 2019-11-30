. $PSScriptRoot/imports.ps1

Describe -Name "Get-RoleReferenceCollectionFromUser" {
    BeforeEach {
        $User = Get-CurrentUser

    }

    It "Get a list of role references for a user" {
        $Response = PSC8y\Get-RoleReferenceCollectionFromUser -Username $User.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

