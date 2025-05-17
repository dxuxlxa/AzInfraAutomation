# ***********************************************
# Script: DeployMail.ps1
# Description: Sets up complete Azure infrastructure 
# for deploying a custom email server.
# Created: 04/25/2025
# ***********************************************

# Write-Output "Logging into Azure..."
# Connect-AzAccount

# -------------------------
# Step 1: Create a Resource Group
# -------------------------
$ResourceGroupName = "MailInfraRG"
$Location = "East US"

Write-Output "Creating Resource Group [$ResourceGroupName] in [$Location]"
New-AzResourceGroup -Name $ResourceGroupName -Location $Location

# -------------------------
# Step 2: Create a Virtual Network & Subnet
# -------------------------
$vnetName      = "MailVNet"
$AddressPrefix = "10.0.0.0/16"  # Virtual Network address space

Write-Output "Creating Virtual Network [$vnetName] with address space [$AddressPrefix]"
$vnet = New-AzVirtualNetwork -ResourceGroupName $ResourceGroupName `
    -Location $Location `
    -Name $vnetName `
    -AddressPrefix $AddressPrefix

# Define subnet details for the mail server
$subnetName   = "MailSubnet" 
$subnetPrefix = "10.0.1.0/24"  # Mail VM Subnet

Write-Output "Adding Subnet [$subnetName] with address prefix [$subnetPrefix] to Virtual Network..."
Add-AzVirtualNetworkSubnetConfig -Name $subnetName `
    -AddressPrefix $subnetPrefix `
    -VirtualNetwork $vnet
# Commit the change and update the variable
$vnet = $vnet | Set-AzVirtualNetwork

# Validate that the subnet exists and capture its ID
if ($vnet.Subnets -and $vnet.Subnets.Count -gt 0) {
    $subnetId = $vnet.Subnets[0].Id
    Write-Output "Subnet ID: $($subnetId)"
}
else {
    Write-Error "No subnets found in the virtual network."
    Exit 1 
}

# -------------------------
# Step 3: Create a Static Public IP Address
# -------------------------
$publicIpName = "MailPublicIP"
Write-Output "Creating a static Public IP Address [$publicIpName]..."
$publicIp = New-AzPublicIpAddress -Name $publicIpName `
    -ResourceGroupName $ResourceGroupName `
    -Location $Location `
    -AllocationMethod Static `
    -Sku Standard `
    -DomainNameLabel "adminsmailserverdomain1"  # e.g., adminsmailserverdomain1.eastus.cloudapp.azure.com

# -------------------------
# Step 4: Create a Network Security Group (NSG) With Inbound Email Port Rules
# -------------------------
$nsgName = "MailNSG"
Write-Output "Creating Network Security Group [$nsgName]..."
$nsg = New-AzNetworkSecurityGroup -ResourceGroupName $ResourceGroupName `
    -Location $Location `
    -Name $nsgName

$MyPublicIP = ""  # Only allowing access from your personal IP address

# Create rule objects for each required port
$nsgRuleSMTP = New-AzNetworkSecurityRuleConfig -Name "Allow-SMTP" `
    -Description "Allow inbound SMTP traffic on port 25" `
    -Access Allow `
    -Protocol Tcp `
    -Direction Inbound `
    -Priority 100 `
    -SourceAddressPrefix $MyPublicIP `
    -SourcePortRange * `
    -DestinationAddressPrefix * `
    -DestinationPortRange 25

$nsgRuleSubmission = New-AzNetworkSecurityRuleConfig -Name "Allow-SMTP-Submission" `
    -Description "Allow inbound SMTP submission traffic on port 587" `
    -Access Allow `
    -Protocol Tcp `
    -Direction Inbound `
    -Priority 110 `
    -SourceAddressPrefix $MyPublicIP `
    -SourcePortRange * `
    -DestinationAddressPrefix * `
    -DestinationPortRange 587

$nsgRuleIMAP = New-AzNetworkSecurityRuleConfig -Name "Allow-IMAP" `
    -Description "Allow inbound IMAP over SSL traffic on port 993" `
    -Access Allow `
    -Protocol Tcp `
    -Direction Inbound `
    -Priority 120 `
    -SourceAddressPrefix $MyPublicIP `
    -SourcePortRange * `
    -DestinationAddressPrefix * `
    -DestinationPortRange 993

$nsgRulePOP3 = New-AzNetworkSecurityRuleConfig -Name "Allow-POP3" `
    -Description "Allow inbound POP3 over SSL traffic on port 995" `
    -Access Allow `
    -Protocol Tcp `
    -Direction Inbound `
    -Priority 130 `
    -SourceAddressPrefix $MyPublicIP `
    -SourcePortRange * `
    -DestinationAddressPrefix * `
    -DestinationPortRange 995

# Add the defined rules to the NSG's SecurityRules collection
Write-Output "Adding inbound rules to the NSG..."
$nsg.SecurityRules.Add($nsgRuleSMTP)
$nsg.SecurityRules.Add($nsgRuleSubmission)
$nsg.SecurityRules.Add($nsgRuleIMAP)
$nsg.SecurityRules.Add($nsgRulePOP3)

# Update the NSG in Azure to enforce the new rules
$nsg | Set-AzNetworkSecurityGroup

# -------------------------
# Step 5: Create a Network Interface (NIC)
# -------------------------
$nicName = "MailNIC"
Write-Output "Creating Network Interface [$nicName]..."
$nic = New-AzNetworkInterface -Name $nicName `
    -ResourceGroupName $ResourceGroupName `
    -Location $Location `
    -SubnetId $subnetId `
    -PublicIpAddressId $publicIp.Id `
    -NetworkSecurityGroupId $nsg.Id

if (-not $nic.Id) {
    Write-Error "Failed to create Network Interface."
    Exit 1
}

# -------------------------
# Step 6: Deploy a Virtual Machine
# -------------------------
$vmName         = "MailVM"
$vmSize         = "Standard_B2ms" 
$imagePublisher = "Canonical"
$imageOffer     = "ubuntu-24_04-lts"
$imageSku       = "server"
$adminUsername  = "admin"
$version        = "24.04.202504080"

# Ensure that the SSH public key exists and retrieve it
$sshPublicKeyPath = "/Users/Admin/.ssh/id_rsa.pub"
if (-not (Test-Path $sshPublicKeyPath)){
    Write-Error "SSH public key not found at $sshPublicKeyPath"
    Exit 1
}
$sshPublicKey = Get-Content -Path $sshPublicKeyPath -Raw

# Get Cloud-Init file and encode it to Base64
$cloudInitFilePath = "mailcow-cloud-init.yaml"
if (Test-Path $cloudInitFilePath) {
    Write-Output "Found cloud-init file at '$cloudInitFilePath'."
    $cloudInitContent = Get-Content -Path $cloudInitFilePath -Raw
    $cloudInitBytes   = [System.Text.Encoding]::UTF8.GetBytes($cloudInitContent)
    $cloudInitEncoded = [System.Convert]::ToBase64String($cloudInitBytes)
} else {
    Write-Warning "Cloud-init file not found at '$cloudInitFilePath'. Skipping cloud-init configuration."
    $cloudInitEncoded = ""
}
Write-Output "Configuring VM [$vmName] with SSH key and cloud-init data..."
$vmConfig = New-AzVMConfig -VMName $vmName -VMSize $vmSize

$dummyPassword = ConvertTo-SecureString "NotUsed" -AsPlainText -Force
$dummyCredential = New-Object System.Management.Automation.PSCredential ($adminUsername, $dummyPassword)
# Set Linux as the OS with SSH authentication enabled and Cloud-Init custom data
$vmConfig = Set-AzVMOperatingSystem -VM $vmConfig `
    -Linux `
    -ComputerName $vmName `
    -Credential $dummyCredential `
    -DisablePasswordAuthentication `
    -CustomData $cloudInitEncoded

# Specify the Ubuntu image
$vmConfig = Set-AzVMSourceImage -VM $vmConfig `
    -PublisherName $imagePublisher `
    -Offer $imageOffer `
    -Skus $imageSku `
    -Version $version

# Attach the NIC to the VM
$vmConfig = Add-AzVMNetworkInterface -VM $vmConfig -Id $nic.Id

# Add the SSH public key to the VM so that it will be placed in the authorized_keys file
$vmConfig = Add-AzVMSshPublicKey -VM $vmConfig `
    -KeyData $sshPublicKey `
    -Path "/home/admin/.ssh/authorized_keys"

Write-Output "Deploying Virtual Machine [$vmName]..."
try {
    New-AzVM -ResourceGroupName $ResourceGroupName -Location $Location -VM $vmConfig
}
catch {
    Write-Error "Failed to deploy VM: $($_.Exception.Message)"
    Exit 1
}

Write-Output "Infrastructure deployment completed. Your Azure resources are set up for the email server."