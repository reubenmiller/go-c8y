. $PSScriptRoot/imports.ps1

Describe -Name "Get-RetentionRule" {
    BeforeEach {
        $RetentionRule = New-RetentionRule -DataType ALARM -MaximumAge 365

    }

    It "Get a retention rule" {
        $Response = PSC8y\Get-RetentionRule -Id $RetentionRule.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Remove-RetentionRule -Id $RetentionRule.id

    }
}

