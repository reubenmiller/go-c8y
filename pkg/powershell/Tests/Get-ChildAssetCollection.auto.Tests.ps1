. $PSScriptRoot/imports.ps1

Describe -Name "Get-ChildAssetCollection" {
    BeforeEach {
        $Device = PSC8y\New-TestDevice
        $ChildDevice = PSC8y\New-TestDevice
        PSC8y\New-ChildAssetReference -Group $Device.id -NewChildDevice $ChildDevice.id
        $Group = PSC8y\New-TestDeviceGroup
        $ChildGroup = PSC8y\New-TestDeviceGroup
        PSC8y\New-ChildAssetReference -Group $Group.id -NewChildGroup $ChildGroup.id

    }

    It "Get a list of the child assets of an existing device" {
        $Response = PSC8y\Get-ChildAssetCollection -Group $Group.id
        $LASTEXITCODE | Should -Be 0
    }
    It "Get a list of the child assets of an existing group" {
        $Response = PSC8y\Get-ChildAssetCollection -Group $Group.id
        $LASTEXITCODE | Should -Be 0
    }

    AfterEach {
        PSC8y\Remove-ManagedObject -Id $ChildDevice.id
        PSC8y\Remove-ManagedObject -Id $Device.id
        PSC8y\Remove-ManagedObject -Id $ChildGroup.id
        PSC8y\Remove-ManagedObject -Id $Group.id

    }
}

