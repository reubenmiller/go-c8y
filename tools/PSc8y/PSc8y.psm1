[cmdletbinding()]
Param()
if ($PSVersionTable["PSVersion"].Major -le 2) {
    $RootFolder = Split-Path -Parent -Path $MyInvocation.Mycommand.Definition
} else {
    # Introduced in Powershell 3.0
    $RootFolder = $PSScriptRoot
}

#
# Create session folder
#
if ($env:HOME) {
    $HomePath = Join-Path $env:HOME -ChildPath ".cumulocity"
} else {
    # default to current directory
    $HomePath = Join-Path "." -ChildPath ".cumulocity"
}
if (!(Test-Path $HomePath)) {
    Write-Host "Creating home directory [$HomePath]"
    $null = New-Item -Path $HomePath -ItemType Directory
}

$PublicManual  = @( Get-ChildItem -Path $RootFolder\Public-manual\ -Filter *.ps1 -Recurse -ErrorAction SilentlyContinue )
$Public  = @( Get-ChildItem -Path $RootFolder\Public\ -Filter *.ps1 -Recurse -ErrorAction SilentlyContinue )
$Private = @( Get-ChildItem -Path $RootFolder\Private\ -Filter *.ps1 -Recurse -ErrorAction SilentlyContinue )


Foreach($import in @($PublicManual + $Public + $Private))
{
    Try
    {
        Write-Verbose ("Importing: {0}" -f $import.FullName)
        . $import.FullName
    }
    Catch
    {
        Write-Error -Message "Failed to import function $($import.fullname): $_"
    }
}

foreach($publicFile in @($PublicManual + $Public)) {
    Write-Verbose "Making: $($publicFile.Basename) public"
    Export-ModuleMember -Function $publicFile.Basename
}

# Install binary (and make it executable)
if ($IsLinux -or $IsMacOS) {
    Install-CumulocityBinary
}

Export-ModuleMember -Alias *

$script:Aliases = @{
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
    app = "Get-Application"
    event = "Get-Event"
    m = "Get-Measurements"
    mo = "Get-ManagedObject"
    op = "Get-Operation"

    # utilities
    json = "ConvertTo-Json"
    tojson = "ConvertTo-Json"
    fromjson = "ConvertFrom-Json"
    rest = "Invoke-RestRequest"

    # session
    session = "Get-Session"
}
