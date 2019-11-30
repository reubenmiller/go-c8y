. $PSScriptRoot/imports.ps1

Describe -Name "Get-MeasurementCollection" {
    BeforeEach {
        $Device = PSC8y\New-TestDevice
        $Measurement = New-TestMeasurement -Device $Device.id -Type "TempReading"

    }

    It "Get a list of measurements" {
        $Response = PSC8y\Get-MeasurementCollection
        $LASTEXITCODE | Should -Be 0
    }
    It "Get a list of measurements for a particular device" {
        $Response = PSC8y\Get-MeasurementCollection -Device $Device.id -Type "TempReading"
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $Device.id

    }
}

