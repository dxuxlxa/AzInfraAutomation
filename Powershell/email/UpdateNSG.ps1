# This will facilitate the changing of source port address for destination port ranges 80,110,143,4190,443,465 to only allow source 172.59.25.215
Write-Output "Starting NSG update script..."
# Write-Output "Logging into Azure..."
# Connect-AzAccount

$ResourceGroupName = "MailInfraRG"
$nsgName = "MailNSG"
$sourceIP = "your IP"

Write-Output "Retrieving Network Security Group '$nsgName' from Resource Group '$ResourceGroupName'..."
# Get the existing NSG
$NSG = Get-AzNetworkSecurityGroup -ResourceGroupName $ResourceGroupName -Name $nsgName
Write-Output "Successfully retrieved NSG."

$Ports = @(80, 110, 143, 443, 465)
Write-Output "Processing rules for ports: $($Ports -join ', ')"

foreach ($Port in $Ports) {
    Write-Output "`nProcessing port $Port..."
    $RuleName = "Allow-Port-$Port"
    Write-Output "Rule name: $RuleName"
    Write-Output "Priority: $([int]($Port + 100))"
    
    Set-AzNetworkSecurityRuleConfig -NetworkSecurityGroup $NSG `
        -Name $RuleName `
        -Direction Inbound `
        -SourceAddressPrefix $sourceIP `
        -SourcePortRange "*" `
        -Protocol Tcp `
        -Access Allow `
        -DestinationAddressPrefix "*" `
        -Priority ([int]($Port + 100)) `
        -DestinationPortRange $Port
    Write-Output "Successfully updated rule for port $Port"
}

Write-Output "`nProcessing special port 4190..."
# Update port 4190 rule separately
Set-AzNetworkSecurityRuleConfig -NetworkSecurityGroup $NSG `
    -Name "Allow-Port-4190" `
    -Direction Inbound `
    -SourceAddressPrefix $sourceIP `
    -SourcePortRange "*" `
    -DestinationAddressPrefix "*" `
    -Protocol Tcp `
    -Access Allow `
    -Priority 400 `
    -DestinationPortRange 4190
Write-Output "Successfully updated rule for port 4190"

Write-Output "`nApplying changes to Network Security Group..."
try {
    $NSG | Set-AzNetworkSecurityGroup
} catch {
    Write-Output "An error occurred: $_"
}
Write-Output "Network Security Group update completed successfully."