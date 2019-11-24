Function New-TestEvent {
<#
.SYNOPSIS
Create a new test event
#>
    [cmdletbinding()]
    Param()

    $Device = PSC8y\New-TestDevice

    PSC8y\New-Event `
        -Device $Device.id `
        -Time "1970-01-01" `
        -Type "c8y_ci_TestAlarm" `
        -Text "Test CI Alarm"
}
