. $PSScriptRoot/imports.ps1

Describe -Name "Remove-MeasurementCollection" {
    BeforeEach {
        $Measurement = New-TestMeasurement

    }

    It "Delete measurement collection for a device" {
        $Response = PSC8y\Remove-MeasurementCollection -Device $Measurement.source.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $Measurement.source.id

    }
}

