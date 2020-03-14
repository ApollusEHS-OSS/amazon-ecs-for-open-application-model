// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"os"
	"sort"
	"strings"

	"github.com/awslabs/amazon-ecs-for-open-application-model/internal/pkg/aws/session"
	"github.com/awslabs/amazon-ecs-for-open-application-model/internal/pkg/deploy/cloudformation"
	"github.com/awslabs/amazon-ecs-for-open-application-model/internal/pkg/deploy/cloudformation/types"
	"github.com/awslabs/amazon-ecs-for-open-application-model/internal/pkg/term/log"
	termprogress "github.com/awslabs/amazon-ecs-for-open-application-model/internal/pkg/term/progress"
	"github.com/iancoleman/strcase"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const (
	deployEnvStart     = "Deploying the infrastructure for the environment."
	deployEnvFailed    = "Failed to deploy the infrastructure for the environment."
	deployEnvSucceeded = "Deployed the environment infrastructure in CloudFormation stack %s.\n"
)

type cfEnvironmentDeployer interface {
	DeployEnvironment(env *types.CreateEnvironmentInput) (*types.Environment, error)
}

// DeployEnvironmentOpts holds the configuration needed to deploy the oam-ecs environment.
type DeployEnvironmentOpts struct {
	prog        progress
	envDeployer cfEnvironmentDeployer
}

// DeployEnvironmentOpts initiates the fields to provision an environment.
func NewDeployEnvironmentOpts() *DeployEnvironmentOpts {
	return &DeployEnvironmentOpts{
		prog: termprogress.NewSpinner(),
	}
}

// Execute deploys the environment CloudFormation stack
func (opts *DeployEnvironmentOpts) Execute() error {
	deployEnvInput := &types.CreateEnvironmentInput{}

	opts.prog.Start(deployEnvStart)

	env, err := opts.envDeployer.DeployEnvironment(deployEnvInput)
	if err != nil {
		opts.prog.Stop(log.Serror(deployEnvFailed))
		return err
	}

	opts.prog.Stop(log.Ssuccessf(deployEnvSucceeded, env.StackName))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Environment Attribute", "Value"})
	table.SetBorder(false)

	keys := make([]string, 0, len(env.StackOutputs))
	for key := range env.StackOutputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		formattedKey := strings.Title(strings.ToLower(strcase.ToDelimited(key, ' ')))
		formattedKey = strings.ReplaceAll(formattedKey, "Cloud Formation", "CloudFormation")
		formattedKey = strings.ReplaceAll(formattedKey, "Ecs", "ECS")
		table.Append([]string{formattedKey, env.StackOutputs[key]})
	}

	table.Render()

	return nil
}

// BuildDeployEnvironmentCmd build the command for creating a new pipeline.
func BuildDeployEnvironmentCmd() *cobra.Command {
	opts := NewDeployEnvironmentOpts()
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the oam-ecs environment",
		Long:  `Creates (or updates) the shared infrastructure, including a VPC and ECS cluster, for oam-ecs applications`,
		Example: `
  Create the oam-ecs environment:
	$ oam-ecs env deploy`,
		PreRunE: runCmdE(func(cmd *cobra.Command, args []string) error {
			session, err := session.Default()
			if err != nil {
				return err
			}
			opts.envDeployer = cloudformation.New(session)
			return nil
		}),
		RunE: runCmdE(func(cmd *cobra.Command, args []string) error {
			return opts.Execute()
		}),
	}

	return cmd
}