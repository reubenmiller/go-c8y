. $PSScriptRoot/imports.ps1

Describe -Name "Remove-EventBinary" {
    BeforeEach {
        $Event = New-TestEvent -WithBinary

    }

    It "Delete an binary attached to an event" {
        $Response = PSC8y\Remove-EventBinary -Id $Event.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

