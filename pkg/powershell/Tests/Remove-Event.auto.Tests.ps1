. $PSScriptRoot/imports.ps1

Describe -Name "Remove-Event" {
    BeforeEach {
        $TestEvent = PSC8y\New-TestEvent

    }

    It "Delete an event" {
        $Response = PSC8y\Remove-Event -Id $TestEvent.id
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestEvent.source.id) {
            PSC8y\Remove-ManagedObject -Id $TestEvent.source.id -ErrorAction SilentlyContinue
        }

    }
}

