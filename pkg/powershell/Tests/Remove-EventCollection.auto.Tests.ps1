. $PSScriptRoot/imports.ps1

Describe -Name "Remove-EventCollection" {
    BeforeEach {
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Remove events with type 'my_CustomType' that were created in the last 10 days" {
        $Response = PSC8y\Remove-EventCollection -Type my_CustomType -DateFrom "-10d"
        $Response | Should -Not -BeNullOrEmpty
    }
    It "Remove events from a device" {
        $Response = PSC8y\Remove-EventCollection -Device $TestDevice.id
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

