. $PSScriptRoot/imports.ps1

Describe -Name "Expand-Device" {
    BeforeAll {
        $Device = PSC8y\New-TestDevice
    }

    It "Expand device (with object)" {
        $Result = PSC8y\Expand-Device $Device
        $Result.id | Should -BeExactly $Device.id
    }

    It "Expand device (with object) by pipeline" {
        $Result = $Device | PSC8y\Expand-Device
        $Result.id | Should -BeExactly $Device.id
    }

    It "Expand device (with id)" {
        $Result = PSC8y\Expand-Device $Device.id
        $Result.id | Should -BeExactly $Device.id
    }

    It "Expand device (with id) by pipeline" {
        $Result = $Device.id | PSC8y\Expand-Device
        $Result.id | Should -BeExactly $Device.id
    }

    It "Expand device (with name)" {
        $Result = PSC8y\Expand-Device $Device.name
        $Result.id | Should -BeExactly $Device.id
    }

    It "Expand device (with name) by pipeline" {
        $Result = $Device.name | PSC8y\Expand-Device
        $Result.id | Should -BeExactly $Device.id
    }

    It "Expand device from Get-DeviceCollection" {
        $Result = Get-DeviceCollection $Device.name | PSC8y\Expand-Device
        $Result.id | Should -BeExactly $Device.id
    }

    AfterAll {
        Remove-ManagedObject -Id $Device.id
    }
}
