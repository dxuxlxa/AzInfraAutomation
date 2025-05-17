Write-Output "Starting NSG creation script..."
Write-Output "Logging into Azure..."
Connect-AzAccount

$ResourceGroupName = "MailInfraRG"
$nsgName = "MailNSG"

Write-Output "Retrieving Network Security Group '$nsgName' from Resource Group '$ResourceGroupName'..."
# Get the existing NSG
$NSG = Get-AzNetworkSecurityGroup -ResourceGroupName $ResourceGroupName -Name $nsgName
Write-Output "Successfully retrieved NSG."

$Ports = @(80, 110, 143, 443, 465)
Write-Output "Processing rules for standard ports: $($Ports -join ', ')"

foreach ($Port in $Ports) {
    Write-Output "`nCreating rule for port $Port..."
    $RulName = "Allow-Port-$Port"
    Write-Output "Rule name: $RulName"
    Write-Output "Priority: $([int](100 + ($Ports.IndexOf($Port) * 50)))"
    
    Add-AzNetworkSecurityRuleConfig -Name $RulName `
        -NetworkSecurityGroup $NSG `
        -Description "Allow inbound traffic on port $Port" `
        -Access Allow `
        -Protocol Tcp `
        -Direction Inbound `
        -Priority ([int](100 + ($Ports.IndexOf($Port) * 50))) `
        -SourceAddressPrefix "*" `
        -SourcePortRange "*" `
        -DestinationAddressPrefix "*" `
        -DestinationPortRange $Port
    Write-Output "Successfully created rule for port $Port"
    
}

Write-Output "`nCreating special rule for port 4190..."
# Add port 4190 rule separately
Add-AzNetworkSecurityRuleConfig -Name "Allow-Port-4190" `
    -NetworkSecurityGroup $NSG `
    -Description "Allow inbound traffic on port 4190" `
    -Access Allow `
    -Protocol Tcp `
    -Direction Inbound `
    -Priority 400 `
    -SourceAddressPrefix "*" `
    -SourcePortRange "*" `
    -DestinationAddressPrefix "*" `
    -DestinationPortRange 4190
Write-Output "Successfully created rule for port 4190"

Write-Output "`nApplying changes to Network Security Group..."
$NSG | Set-AzNetworkSecurityGroup
Write-Output "Network Security Group creation completed successfully."