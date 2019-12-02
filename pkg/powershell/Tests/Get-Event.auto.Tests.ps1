. $PSScriptRoot/imports.ps1

Describe -Name "Get-Event" {
    BeforeEach {
        $TestEvent = PSC8y\New-TestEvent

    }

    It "Get event" {
        $Response = PSC8y\Get-Event -Id $TestEvent.id
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestEvent.source.id) {
            PSC8y\Remove-ManagedObject -Id $TestEvent.source.id -ErrorAction SilentlyContinue
        }

    }
}

