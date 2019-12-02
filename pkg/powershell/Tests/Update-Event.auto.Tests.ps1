. $PSScriptRoot/imports.ps1

Describe -Name "Update-Event" {
    BeforeEach {
        $TestEvent = PSC8y\New-TestEvent
        $TestEvent = PSC8y\New-TestEvent

    }

    It "Update the text field of an existing event" {
        $Response = PSC8y\Update-Event -Id $TestEvent.id -Text "example text 1"
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Update custom properties of an existing event" {
        $Response = PSC8y\Update-Event -Id $TestEvent.id -Data @{ my_event = @{ active = $true } }
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestEvent.source.id) {
            PSC8y\Remove-ManagedObject -Id $TestEvent.source.id -ErrorAction SilentlyContinue
        }
        if ($TestEvent.source.id) {
            PSC8y\Remove-ManagedObject -Id $TestEvent.source.id -ErrorAction SilentlyContinue
        }

    }
}

