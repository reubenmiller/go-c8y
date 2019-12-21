. $PSScriptRoot/imports.ps1

Describe -Name "Remove-Measurement" {
    BeforeEach {
        $Measurement = New-TestMeasurement

    }

    It "Delete measurement" {
        $Response = PSC8y\Remove-Measurement -id $Measurement.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $Measurement.source.id

    }
}

