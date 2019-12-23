. $PSScriptRoot/imports.ps1

Describe -Name "New-Operation" {
    BeforeEach {
        $TestAgent = PSc8y\New-TestAgent

    }

    It "Create operation for a device" {
        $Response = PSc8y\New-Operation -Device $TestAgent.id -Description "Restart device" -Data @{ c8y_Restart = @{} }
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }


    AfterEach {
        if ($TestAgent.id) {
            PSc8y\Remove-ManagedObject -Id $TestAgent.id -ErrorAction SilentlyContinue
        }

    }
}

