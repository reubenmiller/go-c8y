. $PSScriptRoot/imports.ps1

Describe -Name "Remove-EventBinary" {
    BeforeEach {

    }

    It "Delete an binary attached to an event" {
        $Response = PSC8y\Remove-EventBinary -Id 12345
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {

    }
}

