param(
    [Parameter(Mandatory = $true)]
    [string]$HardwareId,
    [string]$Output = "device_diagnostics.json"
)

$ErrorActionPreference = "Stop"
$devices = Get-PnpDevice -PresentOnly:$false | Where-Object {
    $ids = (Get-PnpDeviceProperty -InstanceId $_.InstanceId -KeyName 'DEVPKEY_Device_HardwareIds' -ErrorAction SilentlyContinue).Data
    $ids -and ($ids -contains $HardwareId -or ($ids | Where-Object { $_ -like "$HardwareId*" }))
}

$result = foreach ($device in $devices) {
    $properties = @{}
    foreach ($key in @(
        'DEVPKEY_Device_HardwareIds',
        'DEVPKEY_Device_CompatibleIds',
        'DEVPKEY_Device_DriverVersion',
        'DEVPKEY_Device_DriverProvider',
        'DEVPKEY_Device_ProblemCode',
        'DEVPKEY_Device_ResourcePickerTags'
    )) {
        $properties[$key] = (Get-PnpDeviceProperty -InstanceId $device.InstanceId -KeyName $key -ErrorAction SilentlyContinue).Data
    }

    [ordered]@{
        instance_id = $device.InstanceId
        class = $device.Class
        friendly_name = $device.FriendlyName
        status = $device.Status
        problem_code = $device.Problem
        properties = $properties
    }
}

[ordered]@{
    collected_at = (Get-Date).ToUniversalTime().ToString('o')
    hardware_id = $HardwareId
    devices = @($result)
} | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 $Output

Write-Host "Saved bounded PCI device diagnostics to $Output"
