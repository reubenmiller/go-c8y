. $PSScriptRoot/imports.ps1

Describe -Name "Get-UserMembership" {
    BeforeEach {
        $User = PSc8y\Get-CurrentUser

    }

    It "Get a list of groups that a user belongs to" {
        $Response = PSc8y\Get-UserMembership -User $User.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }


    AfterEach {

    }
}

