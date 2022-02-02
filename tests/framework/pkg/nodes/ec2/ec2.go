package ec2

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	"github.com/rancher/rancher/tests/framework/pkg/nodes"
)

func CreateNodes(client *rancher.Client, nodeNameBase string, publicIp bool, numOfInstancs int64) ([]*nodes.Node, error) {
	ec2Client, err := client.GetEC2Client()
	if err != nil {
		return nil, err
	}

	sshName := nodes.GetSSHKeyName(ec2Client.Config.AWSSSHKeyName)

	runInstancesInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ec2Client.Config.AWSAMI),
		InstanceType: aws.String(ec2Client.Config.InstanceType),
		MinCount:     aws.Int64(numOfInstancs),
		MaxCount:     aws.Int64(numOfInstancs),
		KeyName:      aws.String(sshName),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					VolumeSize: aws.Int64(int64(ec2Client.Config.VolumeSize)),
				},
			},
		},
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: aws.String(ec2Client.Config.AWSIAMProfile),
		},
		Placement: &ec2.Placement{
			AvailabilityZone: aws.String(ec2Client.Config.AWSRegionAZ),
		},
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int64(0),
				AssociatePublicIpAddress: aws.Bool(publicIp),
				Groups:                   aws.StringSlice(ec2Client.Config.AWSSecurityGroups),
			},
		},
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(nodeNameBase),
					},
					{
						Key:   aws.String("CICD"),
						Value: aws.String(ec2Client.Config.AWSCICDInstanceTag),
					},
				},
			},
		},
	}

	reservation, err := ec2Client.SVC.RunInstances(runInstancesInput)
	if err != nil {
		return nil, err
	}

	var listOfInstanceIds []*string

	for _, instance := range reservation.Instances {
		listOfInstanceIds = append(listOfInstanceIds, instance.InstanceId)
	}

	//wait until instance is running
	err = ec2Client.SVC.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: listOfInstanceIds,
	})
	if err != nil {
		return nil, err
	}

	//wait until instance status is ok
	err = ec2Client.SVC.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{
		InstanceIds: listOfInstanceIds,
	})
	if err != nil {
		return nil, err
	}

	// describe instance to get attributes=
	describe, err := ec2Client.SVC.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: listOfInstanceIds,
	})
	if err != nil {
		return nil, err
	}

	readyInstances := describe.Reservations[0].Instances

	sshKey, err := nodes.GetSSHKey(ec2Client.Config.AWSSSHKeyName)
	if err != nil {
		return nil, err
	}

	var ec2Nodes []*nodes.Node

	for _, readyInstance := range readyInstances {
		ec2Node := &nodes.Node{
			NodeID:          *readyInstance.InstanceId,
			PublicIPAddress: *readyInstance.PublicIpAddress,
			SSHUser:         ec2Client.Config.AWSUser,
			SSHKey:          sshKey,
		}
		ec2Nodes = append(ec2Nodes, ec2Node)
	}

	return ec2Nodes, nil
}

func DeleteNodes(client *rancher.Client, nodes []*nodes.Node) (*ec2.TerminateInstancesOutput, error) {
	ec2Client, err := client.GetEC2Client()
	if err != nil {
		return nil, err
	}
	var instanceIDs []*string

	for _, node := range nodes {
		instanceIDs = append(instanceIDs, aws.String(node.NodeID))
	}

	return ec2Client.SVC.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: instanceIDs,
	})
}
