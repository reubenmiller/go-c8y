
. $PSScriptRoot/imports.ps1

Describe -Name "New Measurement" {
    It "Data" {
        $DeviceID = "testdevice01"
        $Response = PSC8y\New-Measurement `
            -Device:$DeviceID `
            -Time "0d" `
            -Type "ciSeria1" `
            -Verbose `
            -Data @{
                test1 = @{
                    signal1 = @{
                        value = 1.234;
                        unit = "°"
                    }
                }
            }
        $Response | Should -Not -BeNullOrEmpty
        $Response.test1.signal1.value | Should -BeExactly 1.234
        $Response.test1.signal1.unit | Should -BeExactly "°"
    }
}