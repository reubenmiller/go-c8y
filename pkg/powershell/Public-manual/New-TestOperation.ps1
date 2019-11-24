Function New-TestOperation {
<#
.SYNOPSIS
Create a new test operation
#>
    [cmdletbinding()]
    Param()

    $Agent = PSC8y\New-TestAgent

    PSC8y\New-Operation `
        -Device $Agent.id `
        -Description "Test operation" `
        -Data @{
            c8y_Restart = @{
                parameters = @{}
            }
        }
}
