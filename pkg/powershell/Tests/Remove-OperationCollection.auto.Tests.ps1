. $PSScriptRoot/imports.ps1

Describe -Name "Remove-OperationCollection" {
    BeforeEach {
        $TestDevice = PSC8y\New-TestDevice

    }

    It "Remove all pending operations for a given device" {
        $Response = PSC8y\Remove-OperationCollection -Device $TestDevice.id -Status PENDING
        $Response | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        if ($TestDevice.id) {
            PSC8y\Remove-ManagedObject -Id $TestDevice.id -ErrorAction SilentlyContinue
        }

    }
}

