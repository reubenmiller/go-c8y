Function New-TestEvent {
<#
.SYNOPSIS
Create a new test event
#>
    [cmdletbinding()]
    Param(
        [switch] $WithBinary
    )

    $Device = PSC8y\New-TestDevice

    $Event = PSC8y\New-Event `
        -Device $Device.id `
        -Time "1970-01-01" `
        -Type "c8y_ci_TestAlarm" `
        -Text "Test CI Alarm"

    if ($WithBinary) {
        $tempfile = New-TemporaryFile
        "Cumulocity test content" | Out-File -LiteralPath $tempfile
        $null = PSC8y\New-EventBinary `
            -Id $Event.id `
            -File $tempfile

        Remove-Item $tempfile
    }

    $Event
}
