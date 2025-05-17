package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
)

type PortPro struct {
	Port     int
	Protocol string
}

// CreateNetSecRules creates security rules for OpenVPN and SSH
func CreateNetSecRules(
	ctx context.Context,
	cred *azidentity.DefaultAzureCredential,
	subscriptionID string,
	resourceGroupName string,
	location string,
	nsgName string,
) ([]armnetwork.SecurityRulesClientCreateOrUpdateResponse, error) {
	InfoLogger.Printf("Creating network security rules in NSG: %s", nsgName)

	securityRulesClient, err := armnetwork.NewSecurityRulesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, LogError(err, "failed to create security rules client")
	}

	newNSGRules := map[string]PortPro{
		"Allow-Port-OVPN": {Port: 1194, Protocol: "UDP"},
		"Allow-Port-SSH":       {Port: 22, Protocol: "TCP"},
		"Allow-Port-HTTP":      {Port: 80, Protocol: "TCP"},
		"Allow-Port-HTTPS":     {Port: 443, Protocol: "TCP"},
	}

	var portList []string
	var protocol armnetwork.SecurityRuleProtocol
	for ruleName, portPro := range newNSGRules {
		portList = append(portList, fmt.Sprintf("%s: %d/%s", ruleName, portPro.Port, portPro.Protocol))
	}
	InfoLogger.Printf("Processing rules for ports: %s", strings.Join(portList, ", "))

	var createdRules []armnetwork.SecurityRulesClientCreateOrUpdateResponse
	basePriority := 100

	for ruleName, portProto := range newNSGRules {
		priority := int32(basePriority)
		basePriority += 50
		InfoLogger.Printf("Creating rule: %s for port %d with priority %d", ruleName, portProto.Port, priority)

		if portProto.Protocol == "UDP" {
			protocol = armnetwork.SecurityRuleProtocolUDP
			InfoLogger.Printf("Using UDP protocol for port %d", portProto.Port)
		} else {
			protocol = armnetwork.SecurityRuleProtocolTCP
			InfoLogger.Printf("Using TCP protocol for port %d", portProto.Port)
		}

		securityRule := armnetwork.SecurityRule{
			Properties: &armnetwork.SecurityRulePropertiesFormat{
				Description:              to.Ptr(fmt.Sprintf("Allow inbound traffic on port %d", portProto.Port)),
				Protocol:                 to.Ptr(protocol),
				SourceAddressPrefix:      to.Ptr("0.0.0.0/0"),
				SourcePortRange:          to.Ptr("*"),
				DestinationAddressPrefix: to.Ptr("0.0.0.0/0"),
				DestinationPortRange:     to.Ptr(fmt.Sprintf("%d", portProto.Port)),
				Access:                   to.Ptr(armnetwork.SecurityRuleAccessAllow),
				Priority:                 to.Ptr(priority),
				Direction:                to.Ptr(armnetwork.SecurityRuleDirectionInbound),
			},
		}

		InfoLogger.Printf("Initiating rule creation for %s", ruleName)
		rulePoller, err := securityRulesClient.BeginCreateOrUpdate(
			ctx,
			resourceGroupName,
			nsgName,
			ruleName,
			securityRule,
			nil,
		)
		if err != nil {
			ErrorLogger.Printf("Failed to begin creating rule %s: %v", ruleName, err)
			return createdRules, fmt.Errorf("failed to begin creating rule %s for port %d: %v", ruleName, portProto.Port, err)
		}

		InfoLogger.Printf("Waiting for rule %s creation to complete...", ruleName)
		ruleResult, err := rulePoller.PollUntilDone(ctx, nil)
		if err != nil {
			ErrorLogger.Printf("Failed to complete rule creation for %s: %v", ruleName, err)
			return createdRules, fmt.Errorf("failed to complete rule creation for port %d: %v", portProto.Port, err)
		}

		createdRules = append(createdRules, ruleResult)
		InfoLogger.Printf("Successfully created security rule: %s", ruleName)
	}

	InfoLogger.Printf("All security rules created successfully in NSG: %s", nsgName)
	return createdRules, nil
}
