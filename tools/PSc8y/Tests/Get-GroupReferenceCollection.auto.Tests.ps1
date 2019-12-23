. $PSScriptRoot/imports.ps1

Describe -Name "Get-GroupReferenceCollection" {
    BeforeEach {
        $User = PSc8y\Get-CurrentUser

    }

    It "Get a list of groups that a user belongs to" {
        $Response = PSc8y\Get-GroupReferenceCollection -User $User.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {

    }
}

