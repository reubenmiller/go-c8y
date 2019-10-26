Function Expand-Application {
<#
.SYNOPSIS
Expand a list of applications replacing any ids or names with the actual application object.

.NOTES
If the given object is already an application object, then it is added with no additional lookup

.PARAMETER InputObject
List of ids, names or application objects

.PARAMETER Type
Limit the types of object by a specific type

.EXAMPLE
Expand-C8yApplication "app-name"

Retrieve the application objects by name or id

.EXAMPLE
Get-C8yApplication *app* | Expand-C8yApplication

Get all the application object (with app in their name). Note the Expand cmdlet won't do much here except for returning the input objects.

.EXAMPLE
Expand-C8yApplication * -Type MICROSERVICE

Expand applications that match a name of "*" and have a type of "MICROSERVICE"

#>
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory=$true,
            ValueFromPipeline=$true,
            Position=0
        )]
        [object[]] $InputObject,

        [ValidateSet("MICROSERVICE", "EXTERNAL", "HOSTED")]
        [string] $Type
    )

    Process {
        [array] $AllApplications = foreach ($iApp in $InputObject)
        {
            if (($iApp -is [string]) -or ($iApp -match "^\d+$"))
            {
                if ($Type) {
                    Get-C8yApplication -Name $iApp -Type:$Type -WhatIf:$false
                } else {
                    Get-C8yApplication -Name $iApp -WhatIf:$false
                }

            }
            elseif (($iApp.applicationId -is [string]) -or ($iApp.applicationId -match "^\d+$"))
            {
                if ($Type)
                {
                    Get-C8yApplication -Name $iApp.applicationId -Type:$Type -WhatIf:$false
                } else {
                    Get-C8yApplication -Name $iApp.applicationId -WhatIf:$false
                }
            }
            else
            {
                if ($Type)
                {
                    # Only return if the type matching the expected type
                    if ($Type -eq $iApp.type)
                    {
                        $iApp
                    }
                }
                else
                {
                    $iApp
                }

            }
        }

        $AllApplications
    }
}
