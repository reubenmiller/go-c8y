[cmdletbinding()]
Param()
$RootFolder = $PSScriptRoot

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

#
# Create session folder
#
$HomePath = Get-SessionHomePath

if (!(Test-Path $HomePath)) {
    Write-Host "Creating home directory [$HomePath]"
    $null = New-Item -Path $HomePath -ItemType Directory
}

# Install binary (and make it executable)
if ($IsLinux -or $IsMacOS) {
    # silence errors
    if ($env:PSC8Y_INSTALL_ON_IMPORT -match "true|1|on") {
        Install-CumulocityBinary -ErrorAction SilentlyContinue
    } else {
        # Make c8y executable
        $binary = Get-CumulocityBinary
        & chmod +x $binary
    }
}

# Set environment variables if a session is set via the C8Y_SESSION env variable
$ExistingSession = Get-Session -WarningAction SilentlyContinue -ErrorAction SilentlyContinue
if ($ExistingSession) {
    Set-EnvironmentVariablesFromSession

    # Display current session
    Write-Host "Current Cumulocity session"
    Write-Host ""
    Write-Host ("    Path: {0}" -f $ExistingSession.Path)
    Write-Host ""
    Write-Host ("description : {0}" -f $ExistingSession.description)
    Write-Host ("host        : {0}" -f $ExistingSession.host)
    Write-Host ("tenant      : {0}" -f $ExistingSession.tenant)
    Write-Host ("username    : {0}" -f $ExistingSession.username)
    Write-Host ("password    : {0}" -f ($ExistingSession.password -replace ".", "*"))
    Write-Host ""
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
    rest = "Invoke-CumulocityRequest"

    # session
    session = "Get-Session"
}
