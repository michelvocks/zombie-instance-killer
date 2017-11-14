package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	gitlabRunnerSecurityGroupName         = "toolbox-np-sg-gitrunnerautoscalingdis"
	launchTimeMinDurationMinutes  float64 = 15.0
)

func main() {
	// Input parameters
	profileName := flag.String("profile", "admin", "aws profile name defined in ~/.aws/config")
	flag.Parse()

	sess, err := session.NewSessionWithOptions(session.Options{
		// enable shared config support.
		SharedConfigState: session.SharedConfigEnable,

		// Optionally set the profile to use from the shared config.
		Profile: *profileName,
	})
	if err != nil {
		panic(err)
	}

	// setup connection
	ec2svc := ec2.New(sess)
	params := &ec2.DescribeInstancesInput{
		Filters: nil,
	}

	// get all instances
	resp, err := ec2svc.DescribeInstances(params)
	if err != nil {
		panic(err)
	}

	// iterate all ec2 instances
	terminateInstance := ec2.TerminateInstancesInput{}
	for idx := range resp.Reservations {
		for _, inst := range resp.Reservations[idx].Instances {
			// check if the instance is attached to the gitlab security group
			for _, secGroup := range inst.SecurityGroups {
				if *secGroup.GroupName == gitlabRunnerSecurityGroupName {
					// check if the instance has no tags
					if len(inst.Tags) == 0 {
						// instance looks like zombie instance.
						// double check the launch time
						duration := time.Since(*inst.LaunchTime)

						if duration.Minutes() >= launchTimeMinDurationMinutes {
							// generate output for debugging purpose
							fmt.Printf("Instance %s found with IP %s. Launch Time: %s, Current Time: %s, Duration: %f\n", *inst.InstanceId, *inst.PrivateIpAddress, (*inst.LaunchTime).UTC().String(), time.Now().UTC().String(), duration.Minutes())

							// add instance id to terminateinstance request obj
							terminateInstance.InstanceIds = append(terminateInstance.InstanceIds, inst.InstanceId)
						}
					}
				}
			}
		}
	}

	// check if we got instances which we should terminate
	if len(terminateInstance.InstanceIds) > 0 {
		// terminate
		out, err := ec2svc.TerminateInstances(&terminateInstance)
		if err != nil {
			panic(err)
		}

		// print output
		fmt.Printf("Terminate output: %s\n", out.String())
	}
}
