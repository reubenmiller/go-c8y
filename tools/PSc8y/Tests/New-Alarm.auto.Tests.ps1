. $PSScriptRoot/imports.ps1

Describe -Name "New-Alarm" {
    BeforeEach {
        $TestDevice = PSc8y\New-TestDevice

    }

    It "Create a new alarm for device" {
        $Response = PSc8y\New-Alarm -Device $TestDevice.id -Type c8y_TestAlarm -Time "-0s" -Text "Test alarm" -Severity MAJOR
        $LASTEXITCODE | Should -Be 0
        $Response | Should -Not -BeNullOrEmpty
    }


    AfterEach {
        if ($TestDevice.id) {
            PSc8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

