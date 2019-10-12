Function Name {
    [cmdletbinding(
        SupportsShouldProcess = $true,
        ConfirmImpact = "{{value}}"
    )]
    Param(

    )

    Begin {

    }

    Process {
        foreach ($iDevice in (Expand-C8yDevice $Device)) {

        }
    }

    End {

    }
}
