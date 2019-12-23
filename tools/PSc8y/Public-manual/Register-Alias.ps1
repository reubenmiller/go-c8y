Function Register-Alias {
    [cmdletbinding()]
    Param()

    $Aliases = @{
        # collections
        alarms = "Get-AlarmCollection"
        apps = "Get-ApplicationCollection"
        devices = "Get-DeviceCollection"
        events = "Get-EventCollection"
        fmo = "Find-ManagedObjectCollection"
        measurements = "Get-MeasurementCollection"
        ops = "Get-OperationCollection"
        series = "Get-MeasurementSeries"

        # single items
        alarm = "Get-Alarm"
        event = "Get-Event"
        m = "Get-Measurements"
        mo = "Get-ManagedObject"
        op = "Get-Operation"

        # utilities
        json = "ConvertTo-Json"

        # session
        session = "Get-Session"
    }

    foreach ($Alias in $Aliases.Keys) {
        $Value = $Aliases[$Alias]

        if ($Value -is [string]) {
            Set-Alias -Name $Alias -Value $Aliases[$Alias] -Scope "Global"
        }
    }
}
