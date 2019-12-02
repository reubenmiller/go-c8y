. $PSScriptRoot/imports.ps1

Describe -Name "New-Measurement" {
    BeforeEach {
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Create measurement" {
        $Response = PSC8y\New-Measurement -Device $TestDevice.id -Time "0s" -Type "myType" -Data @{ c8y_Winding = @{ temperature = @{ value = 1.2345; unit = "Â°C" } } }
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

