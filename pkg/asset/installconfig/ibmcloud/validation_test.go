package ibmcloud_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/networking-go-sdk/dnsrecordsv1"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/installer/pkg/asset/installconfig/ibmcloud"
	"github.com/openshift/installer/pkg/asset/installconfig/ibmcloud/mock"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"
	ibmcloudtypes "github.com/openshift/installer/pkg/types/ibmcloud"
)

type editFunctions []func(ic *types.InstallConfig)

var (
	validRegion                  = "us-south"
	validCIDR                    = "10.0.0.0/16"
	validCISInstanceCRN          = "crn:v1:bluemix:public:internet-svcs:global:a/valid-account-id:valid-instance-id::"
	validClusterName             = "valid-cluster-name"
	validDNSZoneID               = "valid-zone-id"
	validBaseDomain              = "valid.base.domain"
	validPublicSubnetUSSouth1ID  = "public-subnet-us-south-1-id"
	validPublicSubnetUSSouth2ID  = "public-subnet-us-south-2-id"
	validPrivateSubnetUSSouth1ID = "private-subnet-us-south-1-id"
	validPrivateSubnetUSSouth2ID = "private-subnet-us-south-2-id"
	validSubnets                 = []string{
		validPublicSubnetUSSouth1ID,
		validPublicSubnetUSSouth2ID,
		validPrivateSubnetUSSouth1ID,
		validPrivateSubnetUSSouth2ID,
	}
	validSubnet1Name  = "valid-subnet-1"
	validSubnet2Name  = "valid-subnet-2"
	validSubnet3Name  = "valid-subnet-3"
	validVPCID        = "valid-id"
	validVPC          = "valid-vpc"
	validRG           = "valid-resource-group"
	validZoneUSSouth1 = "us-south-1"
	validZoneUSSouth2 = "us-south-2"
	validZoneUSSouth3 = "us-south-3"
	validZones        = []string{
		validZoneUSSouth1,
		validZoneUSSouth2,
		validZoneUSSouth3,
	}
	validZoneSubnetNameMap = map[string]string{
		validZoneUSSouth1: validSubnet1Name,
		validZoneUSSouth2: validSubnet2Name,
		validZoneUSSouth3: validSubnet3Name,
	}

	wrongRG           = "wrong-resource-group"
	wrongSubnetName   = "wrong-subnet"
	wrongVPCID        = "wrong-id"
	wrongVPC          = "wrong-vpc"
	wrongZone         = "wrong-zone"
	anotherValidVPCID = "another-valid-id"
	anotherValidVPC   = "another-valid-vpc"
	anotherValidRG    = "another-valid-resource-group"

	validResourceGroups = []resourcemanagerv2.ResourceGroup{
		{
			Name: &validRG,
			ID:   &validRG,
		},
		{
			Name: &anotherValidRG,
			ID:   &anotherValidRG,
		},
	}
	validVPCs = []vpcv1.VPC{
		{
			Name: &validVPC,
			ID:   &validVPCID,
			ResourceGroup: &vpcv1.ResourceGroupReference{
				Name: &validRG,
				ID:   &validRG,
			},
		},
		{
			Name: &anotherValidVPC,
			ID:   &anotherValidVPCID,
			ResourceGroup: &vpcv1.ResourceGroupReference{
				Name: &anotherValidRG,
				ID:   &anotherValidRG,
			},
		},
	}
	invalidVPC = []vpcv1.VPC{
		{
			Name: &wrongVPC,
			ID:   &wrongVPCID,
			ResourceGroup: &vpcv1.ResourceGroupReference{
				Name: &validRG,
				ID:   &validRG,
			},
		},
	}
	validVPCInvalidRG = []vpcv1.VPC{
		{
			Name: &validVPC,
			ID:   &validVPCID,
			ResourceGroup: &vpcv1.ResourceGroupReference{
				Name: &wrongRG,
				ID:   &wrongRG,
			},
		},
	}
	validSubnet1 = &vpcv1.Subnet{
		Name: &validSubnet1Name,
		VPC: &vpcv1.VPCReference{
			Name: &validVPC,
			ID:   &validVPCID,
		},
		ResourceGroup: &vpcv1.ResourceGroupReference{
			Name: &validRG,
			ID:   &validRG,
		},
		Zone: &vpcv1.ZoneReference{
			Name: &validZoneUSSouth1,
		},
	}
	validSubnet2 = &vpcv1.Subnet{
		Name: &validSubnet2Name,
		VPC: &vpcv1.VPCReference{
			Name: &validVPC,
			ID:   &validVPCID,
		},
		ResourceGroup: &vpcv1.ResourceGroupReference{
			Name: &validRG,
			ID:   &validRG,
		},
		Zone: &vpcv1.ZoneReference{
			Name: &validZoneUSSouth2,
		},
	}
	validSubnet3 = &vpcv1.Subnet{
		Name: &validSubnet3Name,
		VPC: &vpcv1.VPCReference{
			Name: &validVPC,
			ID:   &validVPCID,
		},
		ResourceGroup: &vpcv1.ResourceGroupReference{
			Name: &validRG,
			ID:   &validRG,
		},
		Zone: &vpcv1.ZoneReference{
			Name: &validZoneUSSouth3,
		},
	}
	wrongSubnet = &vpcv1.Subnet{
		Name: &wrongSubnetName,
		VPC: &vpcv1.VPCReference{
			Name: &validVPC,
			ID:   &validVPCID,
		},
		ResourceGroup: &vpcv1.ResourceGroupReference{
			Name: &validRG,
			ID:   &validRG,
		},
		Zone: &vpcv1.ZoneReference{
			Name: &wrongZone,
		},
	}

	validInstanceProfies = []vpcv1.InstanceProfile{{Name: &[]string{"type-a"}[0]}, {Name: &[]string{"type-b"}[0]}}

	machinePoolInvalidType = func(ic *types.InstallConfig) {
		ic.ControlPlane.Platform.IBMCloud = &ibmcloudtypes.MachinePool{
			InstanceType: "invalid-type",
		}
	}

	existingDNSRecordsResponse = []dnsrecordsv1.DnsrecordDetails{
		{
			ID: core.StringPtr("valid-dns-record-1"),
		},
		{
			ID: core.StringPtr("valid-dns-record-2"),
		},
	}
	noDNSRecordsResponse = []dnsrecordsv1.DnsrecordDetails{}
)

func validInstallConfig() *types.InstallConfig {
	return &types.InstallConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: validClusterName,
		},
		BaseDomain: validBaseDomain,
		Networking: &types.Networking{
			MachineNetwork: []types.MachineNetworkEntry{
				{CIDR: *ipnet.MustParseCIDR(validCIDR)},
			},
		},
		Publish: types.ExternalPublishingStrategy,
		Platform: types.Platform{
			IBMCloud: validMinimalPlatform(),
		},
		ControlPlane: &types.MachinePool{
			Platform: types.MachinePoolPlatform{
				IBMCloud: validMachinePool(),
			},
		},
		Compute: []types.MachinePool{{
			Platform: types.MachinePoolPlatform{
				IBMCloud: validMachinePool(),
			},
		}},
	}
}

func validMinimalPlatform() *ibmcloudtypes.Platform {
	return &ibmcloudtypes.Platform{
		Region: validRegion,
	}
}

func validMachinePool() *ibmcloudtypes.MachinePool {
	return &ibmcloudtypes.MachinePool{}
}

func validResourceGroupName(ic *types.InstallConfig) {
	ic.Platform.IBMCloud.ResourceGroupName = "valid-resource-group"
}

func validVPCName(ic *types.InstallConfig) {
	ic.Platform.IBMCloud.VPCName = "valid-vpc"
}

func validControlPlaneSubnetsForZones(ic *types.InstallConfig, zones []string) {
	// If no zones are passed, we select all valid zones
	if zones == nil || len(zones) == 0 {
		zones = validZones
	}
	for _, zone := range zones {
		ic.Platform.IBMCloud.ControlPlaneSubnets = append(ic.Platform.IBMCloud.ControlPlaneSubnets, validZoneSubnetNameMap[zone])
	}
}

func validComputeSubnetsForZones(ic *types.InstallConfig, zones []string) {
	// If no zones are passed, we select all valid zones
	if zones == nil || len(zones) == 0 {
		zones = validZones
	}
	for _, zone := range zones {
		ic.Platform.IBMCloud.ComputeSubnets = append(ic.Platform.IBMCloud.ComputeSubnets, validZoneSubnetNameMap[zone])
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name     string
		edits    editFunctions
		errorMsg string
	}{
		{
			name:     "valid install config",
			edits:    editFunctions{},
			errorMsg: "",
		},
		{
			name: "VPC with no ResourceGroup supplied",
			edits: editFunctions{
				validVPCName,
			},
			errorMsg: `resourceGroupName: Not found: ""$`,
		},
		{
			name: "VPC not found",
			edits: editFunctions{
				validResourceGroupName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.VPCName = "missing-vpc"
				},
			},
			errorMsg: `vpcName: Not found: "missing-vpc"$`,
		},
		{
			name: "VPC not in ResourceGroup",
			edits: editFunctions{
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ResourceGroupName = "wrong-resource-group"
				},
				validVPCName,
			},
			errorMsg: `platform.ibmcloud.vpcName: Invalid value: "valid-vpc": vpc is not in provided ResourceGroup: wrong-resource-group`,
		},
		{
			name: "VPC with no control plane subnets",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
			},
			errorMsg: `\Qplatform.ibmcloud.controlPlaneSubnets: Invalid value: []string(nil): controlPlaneSubnets cannot be empty when providing a vpcName: valid-vpc\E`,
		},
		{
			name: "control plane subnet not found",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{"missing-cp-subnet"}
				},
			},
			errorMsg: `platform.ibmcloud.controlPlaneSubnets: Not found: "missing-cp-subnet"`,
		},
		{
			name: "control plane subnet IBM error",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{"ibm-error-cp-subnet"}
				},
			},
			errorMsg: `platform.ibmcloud.controlPlaneSubnets: Internal error: ibmcloud error`,
		},
		{
			name: "control plane subnet invalid VPC",
			edits: editFunctions{
				validResourceGroupName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.VPCName = "wrong-vpc"
				},
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{"valid-subnet"}
				},
			},
			errorMsg: `platform.ibmcloud.controlPlaneSubnets: Invalid value: "valid-subnet": controlPlaneSubnets contains subnet: valid-subnet, not found in expected vpcID: wrong-id`,
		},
		{
			name: "control plane subnet invalid ResourceGroup",
			edits: editFunctions{
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ResourceGroupName = "wrong-resource-group"
				},
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{"valid-subnet"}
				},
			},
			errorMsg: `platform.ibmcloud.controlPlaneSubnets: Invalid value: "valid-subnet": controlPlaneSubnets contains subnet: valid-subnet, not found in expected resourceGroupName: wrong-resource-group`,
		},
		{
			name: "control plane subnet no zones",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
				},
			},
		},
		{
			name: "control plane subnet no machinepoolplatform",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
				},
				func(ic *types.InstallConfig) {
					ic.ControlPlane.Platform.IBMCloud = nil
				},
			},
		},
		{
			name: "control plane subnet invalid zones",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
				},
				func(ic *types.InstallConfig) {
					ic.ControlPlane.Platform.IBMCloud.Zones = validZones
				},
			},
			errorMsg: `\Qplatform.ibmcloud.controlPlaneSubnets: Invalid value: []string{"valid-subnet-1"}: number of zones (1) covered by controlPlaneSubnets does not match number of provided or default zones (3) for control plane in us-south\E`,
		},
		{
			name: "control plane subnet valid zones some",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
				},
				func(ic *types.InstallConfig) {
					ic.ControlPlane.Platform.IBMCloud.Zones = []string{"us-south-2", "us-south-3"}
				},
			},
		},
		{
			name: "control plane subnet valid zones all",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
				},
				func(ic *types.InstallConfig) {
					ic.ControlPlane.Platform.IBMCloud.Zones = validZones
				},
			},
		},
		{
			name: "VPC with no compute subnets",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
			},
			errorMsg: `\Qplatform.ibmcloud.computeSubnets: Invalid value: []string(nil): computeSubnets cannot be empty when providing a vpcName: valid-vpc\E`,
		},
		{
			name: "compute subnet not found",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ComputeSubnets = []string{"missing-compute-subnet"}
				},
			},
			errorMsg: `platform.ibmcloud.computeSubnets: Not found: "missing-compute-subnet"`,
		},
		{
			name: "compute subnet IBM error",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ComputeSubnets = []string{"ibm-error-compute-subnet"}
				},
			},
			errorMsg: `platform.ibmcloud.computeSubnets: Internal error: ibmcloud error`,
		},
		{
			name: "compute subnet invalid VPC",
			edits: editFunctions{
				validResourceGroupName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.VPCName = "wrong-vpc"
				},
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ComputeSubnets = []string{"valid-subnet"}
				},
			},
			errorMsg: `platform.ibmcloud.computeSubnets: Invalid value: "valid-subnet": computeSubnets contains subnet: valid-subnet, not found in expected vpcID: wrong-id`,
		},
		{
			name: "compute subnet invalid ResourceGroup",
			edits: editFunctions{
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ResourceGroupName = "wrong-resource-group"
				},
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ComputeSubnets = []string{"valid-subnet"}
				},
			},
			errorMsg: `platform.ibmcloud.computeSubnets: Invalid value: "valid-subnet": computeSubnets contains subnet: valid-subnet, not found in expected resourceGroupName: wrong-resource-group`,
		},
		{
			name: "compute subnet no zones",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
				},
			},
		},
		{
			name: "compute subnet no machinepoolplatform",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
				},
				func(ic *types.InstallConfig) {
					ic.Compute[0].Platform.IBMCloud = nil
				},
			},
		},
		{
			name: "compute subnet invalid zones",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet1Name}
				},
				func(ic *types.InstallConfig) {
					ic.Compute[0].Platform.IBMCloud.Zones = validZones
				},
			},
			errorMsg: `\Qplatform.ibmcloud.computeSubnets: Invalid value: []string{"valid-subnet-1"}: number of zones (1) covered by computeSubnets does not match number of provided or default zones (3) for compute[0] in us-south\E`,
		},
		{
			name: "single compute subnet valid zones some",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet2Name}
				},
				func(ic *types.InstallConfig) {
					ic.Compute[0].Platform.IBMCloud.Zones = []string{validZoneUSSouth2}
				},
			},
		},
		{
			name: "multiple compute subnet invalid zones some",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet2Name, validSubnet3Name}
				},
				func(ic *types.InstallConfig) {
					secondCompute := types.MachinePool{
						Platform: types.MachinePoolPlatform{
							IBMCloud: validMachinePool(),
						},
					}
					ic.Compute = append(ic.Compute, secondCompute)
					ic.Compute[0].Platform.IBMCloud.Zones = []string{validZoneUSSouth2, validZoneUSSouth3}
					ic.Compute[1].Platform.IBMCloud.Zones = []string{validZoneUSSouth3}
				},
			},
			errorMsg: `\Qplatform.ibmcloud.computeSubnets: Invalid value: []string{"valid-subnet-2", "valid-subnet-3"}: number of zones (2) covered by computeSubnets does not match number of provided or default zones (1) for compute[1] in us-south\E`,
		},
		{
			name: "multiple compute subnet valid zones some",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet2Name, validSubnet3Name}
				},
				func(ic *types.InstallConfig) {
					secondCompute := types.MachinePool{
						Platform: types.MachinePoolPlatform{
							IBMCloud: validMachinePool(),
						},
					}
					ic.Compute = append(ic.Compute, secondCompute)
					ic.Compute[0].Platform.IBMCloud.Zones = []string{validZoneUSSouth2, validZoneUSSouth3}
					ic.Compute[1].Platform.IBMCloud.Zones = []string{validZoneUSSouth2, validZoneUSSouth3}
				},
			},
		},
		{
			name: "single compute subnet valid zones all",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
				},
				func(ic *types.InstallConfig) {
					ic.Compute[0].Platform.IBMCloud.Zones = validZones
				},
			},
		},
		{
			name: "multiple compute subnet valid zones all",
			edits: editFunctions{
				validResourceGroupName,
				validVPCName,
				func(ic *types.InstallConfig) {
					ic.Platform.IBMCloud.ControlPlaneSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
					ic.Platform.IBMCloud.ComputeSubnets = []string{validSubnet1Name, validSubnet2Name, validSubnet3Name}
				},
				func(ic *types.InstallConfig) {
					secondCompute := types.MachinePool{
						Platform: types.MachinePoolPlatform{
							IBMCloud: validMachinePool(),
						},
					}
					ic.Compute = append(ic.Compute, secondCompute)
					ic.Compute[0].Platform.IBMCloud.Zones = validZones
					ic.Compute[1].Platform.IBMCloud.Zones = validZones
				},
			},
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ibmcloudClient := mock.NewMockAPI(mockCtrl)

	// Mocks: valid install config and all other tests ('AnyTimes()')
	ibmcloudClient.EXPECT().GetSubnet(gomock.Any(), validPublicSubnetUSSouth1ID).Return(&vpcv1.Subnet{Zone: &vpcv1.ZoneReference{Name: &validZoneUSSouth1}}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetSubnet(gomock.Any(), validPublicSubnetUSSouth2ID).Return(&vpcv1.Subnet{Zone: &vpcv1.ZoneReference{Name: &validZoneUSSouth1}}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetSubnet(gomock.Any(), validPrivateSubnetUSSouth1ID).Return(&vpcv1.Subnet{Zone: &vpcv1.ZoneReference{Name: &validZoneUSSouth1}}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetSubnet(gomock.Any(), validPrivateSubnetUSSouth2ID).Return(&vpcv1.Subnet{Zone: &vpcv1.ZoneReference{Name: &validZoneUSSouth1}}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetSubnet(gomock.Any(), "subnet-invalid-zone").Return(&vpcv1.Subnet{Zone: &vpcv1.ZoneReference{Name: &[]string{"invalid"}[0]}}, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetVSIProfiles(gomock.Any()).Return(validInstanceProfies, nil).AnyTimes()
	ibmcloudClient.EXPECT().GetVPCZonesForRegion(gomock.Any(), validRegion).Return([]string{"us-south-1", "us-south-2", "us-south-3"}, nil).AnyTimes()

	// Mocks: VPC with no ResourceGroup supplied
	// No mocks required

	// Mocks: VPC not found
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)

	// Mocks: VPC not in ResourceGroup
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)

	// Mocks: VPC with no control plane subnets
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)

	// Mocks: control plane subnet not found
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), "missing-cp-subnet", validRegion).Return(nil, &ibmcloud.VPCResourceNotFoundError{})

	// Mocks: control plane subnet IBM error
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), "ibm-error-cp-subnet", validRegion).Return(nil, errors.New("ibmcloud error"))

	// Mocks: control plane subnet invalid VPC
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(invalidVPC, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), "valid-subnet", validRegion).Return(validSubnet1, nil)

	// Mocks: control plane subnet invalid ResourceGroup
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCInvalidRG, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), "valid-subnet", validRegion).Return(validSubnet1, nil)

	// Mocks: control plane subnet no zones
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil).Times(2)

	// Mocks: control plane subnet no machinepoolplatform
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil).Times(2)

	// Mocks: control plane subnet invalid zones
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil)

	// Mocks: control plane subnet valid zones some
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil).Times(2)

	// Mocks: control plane subnet valid zones all
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil).Times(2)

	// Mocks: VPC with no compute subnets
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)

	// Mocks: compute subnet not found
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), "missing-compute-subnet", validRegion).Return(nil, &ibmcloud.VPCResourceNotFoundError{})

	// Mocks: compute subnet IBM error
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), "ibm-error-compute-subnet", validRegion).Return(nil, errors.New("ibmcloud error"))

	// Mocks: compute subnet invalid VPC
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(invalidVPC, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), "valid-subnet", validRegion).Return(validSubnet1, nil)

	// Mocks: compute subnet invalid ResourceGroup
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCInvalidRG, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), "valid-subnet", validRegion).Return(validSubnet1, nil)

	// Mocks: compute subnet no zones
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil).Times(2)

	// Mocks: compute subnet no machinepoolplatform
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil).Times(2)

	// Mocks: compute subnet invalid zones
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil)

	// Mocks: single compute subnet valid zones some
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil)

	// Mocks: multiple compute subnet invalid zones some
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil).Times(2)

	// Mocks: multiple compute subnet valid zones some
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil).Times(2)

	// Mocks: single compute subnet valid zones all
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil).Times(2)

	// Mocks: multiple compute subnet valid zones all
	ibmcloudClient.EXPECT().GetResourceGroups(gomock.Any()).Return(validResourceGroups, nil)
	ibmcloudClient.EXPECT().GetVPCs(gomock.Any(), validRegion).Return(validVPCs, nil)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet1Name, validRegion).Return(validSubnet1, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet2Name, validRegion).Return(validSubnet2, nil).Times(2)
	ibmcloudClient.EXPECT().GetSubnetByName(gomock.Any(), validSubnet3Name, validRegion).Return(validSubnet3, nil).Times(2)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			editedInstallConfig := validInstallConfig()
			for _, edit := range tc.edits {
				edit(editedInstallConfig)
			}

			aggregatedErrors := ibmcloud.Validate(ibmcloudClient, editedInstallConfig)
			if tc.errorMsg != "" {
				assert.Regexp(t, tc.errorMsg, aggregatedErrors)
			} else {
				assert.NoError(t, aggregatedErrors)
			}
		})
	}
}

func TestValidatePreExistingPublicDNS(t *testing.T) {
	cases := []struct {
		name     string
		internal bool
		edits    editFunctions
		errorMsg string
	}{
		{
			name:     "no pre-existing External DNS records",
			internal: false,
			errorMsg: "",
		},
		{
			name:     "pre-existing External DNS records",
			internal: false,
			errorMsg: `^record api\.valid-cluster-name\.valid\.base\.domain already exists in CIS zone \(valid-zone-id\) and might be in use by another cluster, please remove it to continue$`,
		},
		{
			name:     "cannot get External zone ID",
			internal: false,
			errorMsg: `^baseDomain: Internal error$`,
		},
		{
			name:     "cannot get External DNS records",
			internal: false,
			errorMsg: `^baseDomain: Internal error$`,
		},
		{
			name:     "no validation of Internal PublishStrategy",
			internal: true,
			errorMsg: "",
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ibmcloudClient := mock.NewMockAPI(mockCtrl)

	dnsRecordName := fmt.Sprintf("api.%s.%s", validClusterName, validBaseDomain)

	metadata := ibmcloud.NewMetadata(validBaseDomain, "us-south", nil, nil)
	metadata.SetCISInstanceCRN(validCISInstanceCRN)

	// Mocks: no pre-existing External DNS records
	ibmcloudClient.EXPECT().GetDNSZoneIDByName(gomock.Any(), validBaseDomain, types.ExternalPublishingStrategy).Return(validDNSZoneID, nil)
	ibmcloudClient.EXPECT().GetDNSRecordsByName(gomock.Any(), validCISInstanceCRN, validDNSZoneID, dnsRecordName).Return(noDNSRecordsResponse, nil)

	// Mocks: pre-existing External DNS records
	ibmcloudClient.EXPECT().GetDNSZoneIDByName(gomock.Any(), validBaseDomain, types.ExternalPublishingStrategy).Return(validDNSZoneID, nil)
	ibmcloudClient.EXPECT().GetDNSRecordsByName(gomock.Any(), validCISInstanceCRN, validDNSZoneID, dnsRecordName).Return(existingDNSRecordsResponse, nil)

	// Mocks: cannot get External zone ID
	ibmcloudClient.EXPECT().GetDNSZoneIDByName(gomock.Any(), validBaseDomain, types.ExternalPublishingStrategy).Return("", fmt.Errorf(""))

	// Mocks: cannot get External DNS records
	ibmcloudClient.EXPECT().GetDNSZoneIDByName(gomock.Any(), validBaseDomain, types.ExternalPublishingStrategy).Return(validDNSZoneID, nil)
	ibmcloudClient.EXPECT().GetDNSRecordsByName(gomock.Any(), validCISInstanceCRN, validDNSZoneID, dnsRecordName).Return(nil, fmt.Errorf(""))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			validInstallConfig := validInstallConfig()
			if tc.internal {
				validInstallConfig.Publish = types.InternalPublishingStrategy
			}
			aggregatedErrors := ibmcloud.ValidatePreExistingPublicDNS(ibmcloudClient, validInstallConfig, metadata)
			if tc.errorMsg != "" {
				assert.Regexp(t, tc.errorMsg, aggregatedErrors)
			} else {
				assert.NoError(t, aggregatedErrors)
			}
		})
	}
}
