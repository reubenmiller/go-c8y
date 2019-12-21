. $PSScriptRoot/imports.ps1

Describe -Name "New-ApplicationBinary" {
    BeforeEach {
        $App = New-Application -Name my-temp-app2 -Type HOSTED -Key "my-temp-app2-key" -ContextPath "my-temp-app2"

    }

    It "Upload application microservice binary" {
        $Response = PSC8y\New-ApplicationBinary -Application $App.id -File ./helloworld.zip
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        Remove-Item ./helloworld.zip
        Remove-Application -Application $App.id

    }
}

