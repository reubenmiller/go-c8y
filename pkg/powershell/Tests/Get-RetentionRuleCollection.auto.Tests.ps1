. $PSScriptRoot/imports.ps1

Describe -Name "Get-RetentionRuleCollection" {
    BeforeEach {

    }

    It "Get a list of retention rules" {
        $Response = PSC8y\Get-RetentionRuleCollection
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {

    }
}

