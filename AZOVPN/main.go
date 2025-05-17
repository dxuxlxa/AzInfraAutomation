package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"azovpn/utils"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/joho/godotenv"
)

func main() {
	// Initialize logging
	if err := utils.InitLogger(); err != nil {
		fmt.Printf("Failed to initialize logging: %v\n", err)
		os.Exit(1)
	}
	defer utils.CloseLogger()

	utils.InfoLogger.Println("Starting OpenVPN Azure VM deployment")

	// Define command-line flags
	forceDelete := flag.Bool("force-delete", false, "Force delete existing resource group without prompting")
	recreate := flag.Bool("recreate", false, "Delete and recreate the resource group if it exists")
	getBillingInfo := flag.Bool("bills", false, "Get up to date statistics on the billing of this resource group")
	flag.Parse()

	utils.InfoLogger.Println("Loading environment variables")
	err := godotenv.Load()
	utils.LogAndExit(err, "Error loading environment file")

	ctx := context.Background()
	utils.InfoLogger.Println("Creating Azure credentials")
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	utils.LogAndExit(err, "Failed to get credentials")

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		utils.LogAndExit(fmt.Errorf("AZURE_SUBSCRIPTION_ID not set"), "Configuration error")
	}

	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	location := os.Getenv("VM_LOCATION")
	utils.InfoLogger.Printf("Using resource group: %s in location: %s", resourceGroupName, location)

	groupsClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	utils.LogAndExit(err, "Failed to create resource groups client")

	// Check if the resource group exists
	utils.InfoLogger.Printf("Checking if resource group %s exists", resourceGroupName)
	checkRG, err := groupsClient.Get(ctx, resourceGroupName, nil)
	if err == nil {
		utils.InfoLogger.Printf("Resource group %q exists", *checkRG.Name)
		if *getBillingInfo {
			log.Println("Unimplemented")
			// utils.GetBillingInfo()
			return
		} else if *forceDelete || *recreate {
			utils.InfoLogger.Printf("Deleting resource group %q...", *checkRG.Name)
			delPoller, err := groupsClient.BeginDelete(ctx, resourceGroupName, nil)
			utils.LogAndExit(err, "Failed to begin resource group deletion")

			_, err = delPoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
				Frequency: 30 * time.Second,
			})
			utils.LogAndExit(err, "Failed to complete resource group deletion")
			utils.InfoLogger.Printf("Resource group %q deleted", resourceGroupName)

			if !*recreate {
				utils.InfoLogger.Println("Resource group deleted, exiting as requested")
				return
			}
		} else {
			utils.LogAndExit(
				fmt.Errorf("resource group %q already exists", resourceGroupName),
				"Use --force-delete to delete or --recreate to delete and recreate",
			)
		}
		log.Println("Waiting for resource group deletion to propagate...")
		for remaining := 120; remaining >= 0; remaining-- {
			minutes := remaining / 60
			seconds := remaining % 60
			fmt.Printf("\r%02d:%02d", minutes, seconds)
			time.Sleep(1 * time.Second)
		}
	}
	fmt.Println("Starting RG Creation") // Move to the next line after countdown

	// Create new resource group
	utils.InfoLogger.Printf("Creating new resource group: %s", resourceGroupName)
	rgResponse, err := groupsClient.CreateOrUpdate(ctx, resourceGroupName, armresources.ResourceGroup{
		Location: &location,
		Name:     &resourceGroupName,
	}, nil)
	utils.LogAndExit(err, "Failed to create resource group")
	utils.InfoLogger.Printf("Resource group %q created in %q", *rgResponse.Name, *rgResponse.Location)

	vnetName := os.Getenv("VNET_NAME")
	addressPrefix := os.Getenv("ADDRESS_PREFIX")
	utils.InfoLogger.Printf("Creating virtual network %s with address prefix %s", vnetName, addressPrefix)

	vnetResult, err := utils.CreateVnet(ctx, cred, subscriptionID, resourceGroupName, location, vnetName, addressPrefix)
	utils.LogAndExit(err, "Failed to create virtual network")
	utils.InfoLogger.Printf("Virtual network %q created", *vnetResult.Name)

	// Create Subnet and Public IP
	utils.InfoLogger.Println("Creating subnet and public IP address")
	subnetPoller, publicIPPoller, err := utils.CreateAddresses(ctx, cred, subscriptionID, resourceGroupName, location, vnetName)
	utils.LogAndExit(err, "Failed to begin network address creation")

	// Poll for subnet completion
	utils.InfoLogger.Println("Waiting for subnet creation to complete...")
	subnetResult, err := subnetPoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 6 * time.Second,
	})
	utils.LogAndExit(err, "Failed to complete subnet creation")
	utils.InfoLogger.Printf("Virtual subnetwork %q created", *subnetResult.Name)

	// Poll for public IP completion
	utils.InfoLogger.Println("Waiting for public IP creation to complete...")
	publicIPResult, err := publicIPPoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 6 * time.Second,
	})
	utils.LogAndExit(err, "Failed to complete public IP creation")
	utils.InfoLogger.Printf("Public IP %q created\n", *publicIPResult.Name)
	utils.InfoLogger.Printf("Public IP %q created\n", *publicIPResult.PublicIPAddress.Properties.IPAddress)

	// Create Network Security Group (NSG)
	nsgName := os.Getenv("NSG_NAME")
	utils.InfoLogger.Printf("Creating network security group: %s", nsgName)
	nsgPoller, err := utils.CreateNsg(ctx, cred, subscriptionID, resourceGroupName, location)
	utils.LogAndExit(err, "Failed to begin creating NSG")

	nsgResult, err := nsgPoller.PollUntilDone(ctx, nil)
	utils.LogAndExit(err, "Failed to create NSG")
	utils.InfoLogger.Printf("Network Security Group %q created", *nsgResult.Name)

	utils.InfoLogger.Println("Creating network security rules")
	netSecRules, err := utils.CreateNetSecRules(ctx, cred, subscriptionID, resourceGroupName, location, nsgName)
	utils.LogAndExit(err, "Failed in the netsec rules creation")
	for i := range netSecRules {
		utils.InfoLogger.Printf("Network security rule %q created", *netSecRules[i].Name)
	}

	subnetID := subnetResult.ID
	publicIPID := publicIPResult.ID
	nsgID := nsgResult.ID

	// Create a Network Interface (NIC)
	utils.InfoLogger.Println("Creating network interface")
	nicPoller, err := utils.CreateNIC(ctx, subscriptionID, cred, subnetID, publicIPID, nsgID, location, resourceGroupName)
	utils.LogAndExit(err, "Failed to begin nic creation")

	utils.InfoLogger.Println("Waiting for NIC creation to complete...")
	nicResult, err := nicPoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 7 * time.Second,
	})
	utils.LogAndExit(err, "Failed to complete NIC creation")
	utils.InfoLogger.Printf("NIC %q created", *nicResult.Name)

	// Deploy VM
	nicID := *nicResult.ID
	utils.InfoLogger.Println("Starting virtual machine deployment")
	vmPoller, err := utils.CreateVM(ctx, cred, subscriptionID, resourceGroupName, location, nicID)
	utils.LogAndExit(err, "Failed to begin VM creation")

	utils.InfoLogger.Println("Waiting for VM creation to complete...")
	vmResult, err := vmPoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 7 * time.Second,
	})
	utils.LogAndExit(err, "Failed to complete VM creation")

	utils.InfoLogger.Printf("VM %q created successfully", *vmResult.Name)
	utils.InfoLogger.Println("OpenVPN Azure VM deployment completed successfully")

	fmt.Printf("OpenVPN VM can be accessed by ssh -i ~/.ssh/id_rsa.pem user@%v\n", publicIPResult.Properties.LinkedPublicIPAddress)
}
