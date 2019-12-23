. $PSScriptRoot/imports.ps1

Describe -Name "Get-ApplicationBootstrapUser" {
    BeforeEach {
        $App = New-Application -Name "helloworld-microservice" -Type MICROSERVICE -Key "helloworld-microservice-key" -ContextPath "helloworld-microservice"

    }

    It "Get application bootstrap user" {
        $Response = PSc8y\Get-ApplicationBootstrapUser -Application $App.name
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        Remove-Application -Application $App.id

    }
}

