. $PSScriptRoot/imports.ps1

Describe -Name "Update-RetentionRule" {
    BeforeEach {
        $RetentionRule = New-RetentionRule -DataType ALARM -MaximumAge 365

    }

    It "Update a retention rule" {
        $Response = PSC8y\Update-RetentionRule -Id $RetentionRule.id -DataType MEASUREMENT -FragmentType "custom_FragmentType"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        Remove-RetentionRule -Id $RetentionRule.id

    }
}

