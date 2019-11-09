Function New-C8yApiGoGetValueFromFlag {
    [cmdletbinding()]
    Param(
        [Parameter(
            Mandatory = $true
        )]
        [object] $Parameters,

        [Parameter(
            Mandatory = $true
        )]
        [ValidateSet("query", "path", "body")]
        [string] $SetterType
    )

    $prop = $Parameters.name
    $queryParam = $Parameters.property
    if (!$queryParam) {
        $queryParam = $Parameters.name
    }

    $Type = $Parameters.type

    $Definitions = @{}
    $Setters = @{
        "boolean" = @{}
        "datetime" = @{}
        "[]string" = @{}
        "application" = @{}
        "[]device" = @{}
        "string" = @{}
        "tenant" = @{}
    }

    # Boolean
    $Setters."boolean"."query" = "query.Add(`"${queryParam}`", `"true`")"
    $Setters."boolean"."path" = "pathParameters[`"${queryParam}`"] = `"true`""
    $Setters."boolean"."body" = "body.Set(`"${queryParam}`", `"true`")"
    $Definitions."boolean" = @"
    if v, err := cmd.Flags().GetBool("${prop}"); err == nil {
        if v {
            $($Setters."boolean".$SetterType)
        }
    } else {
        return newUserError("Flag does not exist")
    }
"@

    $Setters."datetime"."query" = "query.Add(`"${queryParam}`", v)"
    $Setters."datetime"."path" = "pathParameters[`"${queryParam}`"] = v"
    $Setters."datetime"."body" = "body.Set(`"${queryParam}`", decodeC8yTimestamp(v))"
    $Definitions."datetime" = @"
    if cmd.Flags().Changed("${prop}") {
        if v, err := tryGetTimestampFlag(cmd, "${prop}"); err == nil && v != "" {
            $($Setters."datetime".$SetterType)
        } else {
            return newUserError("invalid date format", err)
        }
    }
"@

    # string array
    $Setters."[]string"."query" = "query.Add(`"${queryParam}`", url.QueryEscape(v))"
    $Setters."[]string"."path" = "pathParameters[`"${queryParam}`"] = v"
    $Setters."[]string"."body" = "body.Set(`"${queryParam}`", v)"
    $Definitions."[]string" = @"
    if items, err := cmd.Flags().GetStringSlice("${prop}"); err == nil {
        if len(items) > 0 {
            for _, v := range items {
                if v != "" {
                    $($Setters."[]string".$SetterType)
                }
            }
        }
    } else {
        return newUserError("Flag does not exist")
    }
"@


    # application
    $Setters."application"."query" = "query.Add(`"${queryParam}`", url.QueryEscape(newIDValue(item).GetID()))"
    $Setters."application"."path" = "pathParameters[`"${queryParam}`"] = newIDValue(item).GetID()"
    $Setters."application"."body" = "body.Set(`"${queryParam}`", newIDValue(item).GetID())"
    $Definitions."application" = @"
    if cmd.Flags().Changed("${prop}") {
        ${prop}InputValues, ${prop}Value, err := getApplicationSlice(cmd, args, "${prop}")

        if err != nil {
            return newUserError("no matching applications found", ${prop}InputValues, err)
        }

        if len(${prop}Value) == 0 {
            return newUserError("no matching applications found", ${prop}InputValues)
        }

        for _, item := range ${prop}Value {
            if item != "" {
                $($Setters."application".$SetterType)
            }
        }
    }
"@

    # device array
    $Setters."[]device"."query" = "query.Add(`"${queryParam}`", newIDValue(item).GetID())"
    $Setters."[]device"."path" = "pathParameters[`"${queryParam}`"] = newIDValue(item).GetID()"
    $Setters."[]device"."body" = "body.Set(`"${queryParam}`", newIDValue(item).GetID())"
    $Definitions."[]device" = @"
    if cmd.Flags().Changed("${prop}") {
        ${prop}InputValues, ${prop}Value, err := getFormattedDeviceSlice(cmd, args, "${prop}")

        if err != nil {
            return newUserError("no matching devices found", ${prop}InputValues, err)
        }

        if len(${prop}Value) == 0 {
            return newUserError("no matching devices found", ${prop}InputValues)
        }

        for _, item := range ${prop}Value {
            if item != "" {
                $($Setters."[]device".$SetterType)
            }
        }
    }
"@

    # tenant
    $Setters."tenant"."query" = "query.Add(`"${queryParam}`", url.QueryEscape(v))"
    $Setters."tenant"."path" = "pathParameters[`"${queryParam}`"] = v"
    $Setters."tenant"."body" = "body.Set(`"${queryParam}`", v)"
    $Definitions."tenant" = @"
    if v := getTenantWithDefaultFlag(cmd, "${prop}", client.TenantName); v != `"`" {
        $($Setters.tenant.$SetterType)
    }
"@

    # string
    $Setters."string"."query" = "query.Add(`"${queryParam}`", url.QueryEscape(v))"
    $Setters."string"."path" = "pathParameters[`"${queryParam}`"] = v"
    $Setters."string"."body" = "body.Set(`"${queryParam}`", v)"
    $Definitions."string" = @"
    if v, err := cmd.Flags().GetString("${prop}"); err == nil {
        if v != "" {
            $($Setters.string.$SetterType)
        }
    } else {
        return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "${prop}", err))
    }
"@

    # json - don't do anything because it should be manually set
    $Definitions."json" = ""

    $MatchingType = $Definitions.Keys -eq $Type

    if ($null -eq $MatchingType) {
        # Default to a string
        $MatchingType = "string"
    }

    $Definitions[$MatchingType]
}

#
# Definitions
#
