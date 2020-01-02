write-host "Waiting for other jobs to complete"

$headers = @{
  "Authorization" = "Bearer $env:ApiKey"
  "Content-type" = "application/json"
}

[datetime]$stop = ([datetime]::Now).AddMinutes($env:TimeOutMins)
[bool]$success = $false

while(!$success -and ([datetime]::Now) -lt $stop) {
    $project = Invoke-RestMethod -Uri "https://ci.appveyor.com/api/projects/$env:APPVEYOR_ACCOUNT_NAME/$env:APPVEYOR_PROJECT_SLUG" -Headers $headers -Method GET
    $success = $true  
    $project.build.jobs | foreach-object {
        if (($_.jobId -ne $env:APPVEYOR_JOB_ID) -and ($_.status -ne "success")) {
            $success = $false
        };
        $_.jobId; $_.status
    }
    if (!$success) {Start-sleep 5}
}

if (!$success) {
    throw "Test jobs were not finished in $env:TimeOutMins minutes"
    Exit 1
}
