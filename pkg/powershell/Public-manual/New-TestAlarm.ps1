Function New-TestAlarm {
<#
.SYNOPSIS
Create a new test alarm
#>
    [cmdletbinding()]
    Param()

    $Device = PSC8y\New-TestDevice

    PSC8y\New-Alarm `
        -Device $Device.id `
        -Time "1970-01-01" `
        -Type "c8y_ci_TestAlarm" `
        -Severity MAJOR `
        -Text "Test CI Alarm"
}
