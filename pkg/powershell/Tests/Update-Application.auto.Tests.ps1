. $PSScriptRoot/imports.ps1

Describe -Name "Update-Application" {
    BeforeEach {
        $App = New-Application -Name "helloworld-app" -Type HOSTED -Key "helloworld-app-key" -ContextPath "helloworld-app"

    }

    It "Update application availability to MARKET" {
        $Response = PSC8y\Update-Application -Application "helloworld-app" -Availability "MARKET"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        Remove-Application -Application $App.id

    }
}

