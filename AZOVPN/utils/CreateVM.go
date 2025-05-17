package utils

import (
	"context"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
)

// CreateVM creates a new virtual machine with the specified parameters
func CreateVM(ctx context.Context, cred *azidentity.DefaultAzureCredential, subscriptionID string, resourceGroupName string, location string, nicID string) (*runtime.Poller[armcompute.VirtualMachinesClientCreateOrUpdateResponse], error) {
	InfoLogger.Printf("Starting VM creation in resource group %s", resourceGroupName)

	vmName := os.Getenv("VM_NAME")
	adminUsername := os.Getenv("ADMIN_USERNAME")
	sshPublicKeyPath := os.Getenv("SSH_PUB_KEY_PATH")
	sshPublicKeyContent := os.Getenv("SSH_PUB_KEY_CONTENT")

	InfoLogger.Printf("Creating VM with name: %s, username: %s", vmName, adminUsername)
	InfoLogger.Printf("Using SSH public key path: %s", sshPublicKeyPath)

	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		ErrorLogger.Printf("Failed to create VM client: %v", err)
		return nil, err
	}

	InfoLogger.Printf("Configuring VM parameters...")
	vmParams := armcompute.VirtualMachine{
		Location: to.Ptr(location),
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: to.Ptr(armcompute.VirtualMachineSizeTypesStandardB2Ms),
			},
			StorageProfile: &armcompute.StorageProfile{
				ImageReference: &armcompute.ImageReference{
					Publisher: to.Ptr(os.Getenv("PUBLISHER_NAME")),
					Offer:     to.Ptr(os.Getenv("OFFER")),
					SKU:       to.Ptr(os.Getenv("SKU")),
					Version:   to.Ptr(os.Getenv("VM_VERSION")),
				},
				OSDisk: &armcompute.OSDisk{
					CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesFromImage),
					ManagedDisk: &armcompute.ManagedDiskParameters{
						StorageAccountType: to.Ptr(armcompute.StorageAccountTypesStandardLRS),
					},
					DeleteOption: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
				},
			},
			OSProfile: &armcompute.OSProfile{
				ComputerName:  to.Ptr(vmName),
				AdminUsername: to.Ptr(adminUsername),
				LinuxConfiguration: &armcompute.LinuxConfiguration{
					DisablePasswordAuthentication: to.Ptr(true),
					SSH: &armcompute.SSHConfiguration{
						PublicKeys: []*armcompute.SSHPublicKey{
							{
								Path:    to.Ptr(sshPublicKeyPath),
								KeyData: to.Ptr(sshPublicKeyContent),
							},
						},
					},
				},
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
					{
						ID: to.Ptr(nicID),
						Properties: &armcompute.NetworkInterfaceReferenceProperties{
							Primary: to.Ptr(true),
						},
					},
				},
			},
		},
	}
	return vmClient.BeginCreateOrUpdate(ctx, resourceGroupName, vmName, vmParams, nil)
}
