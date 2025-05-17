package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/joho/godotenv"
)

// func printExit(err error, message string) {
// 	if err != nil {
// 		fmt.Printf("%v: %v\n", message, err)
// 		os.Exit(1)
// 	}
// }

func GetRg() {
	err := godotenv.Load()
	PrintExit(err, "Error Loading .env file")
	ctx := context.Background()
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	PrintExit(err, "Failed to obtain a credential")

	// Create a resource groups client
	rgClient, err := armresources.NewResourceGroupsClient(os.Getenv("AZURE_SUBSCRIPTION_ID"), cred, nil)
	PrintExit(err, "Failed to create resource groups client")

	// List all resource groups
	pager := rgClient.NewListPager(nil)
	fmt.Println("Listing Azure Resource Groups:")
	fmt.Println("-----------------------------")

	for pager.More() {
		page, err := pager.NextPage(ctx)
		PrintExit(err, "Failed to get next page of resource groups")

		for _, rg := range page.ResourceGroupListResult.Value {
			fmt.Printf("Name: %s\n", *rg.Name)
			fmt.Printf("Location: %s\n", *rg.Location)
			if rg.Tags != nil {
				fmt.Println("Tags:")
				for k, v := range rg.Tags {
					fmt.Printf("\t%s: %s\n", k, *v)
				}
			}
			fmt.Println("-----------------------------")
		}
	}
}
