. $PSScriptRoot/imports.ps1

Describe -Name "Copy-Application" {
    BeforeEach {
        New-Application -Name my-example-app -Type HOSTED -Key "my-example-app-key" -ContextPath "my-example-app"

    }

    It "Copy an existing application" {
        $Response = PSc8y\Copy-Application -Application "my-example-app"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        Remove-Application -Application "my-example-app"
        Remove-Application -Application "clonemy-example-app"

    }
}

