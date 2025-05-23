// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
package steps_aws_test

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	terratest_aws "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/open-edge-platform/edge-manageability-framework/installer/internal/config"
	"github.com/open-edge-platform/edge-manageability-framework/installer/internal/steps"
	steps_aws "github.com/open-edge-platform/edge-manageability-framework/installer/internal/steps/aws"

	"github.com/stretchr/testify/suite"
)

type StateBucketTest struct {
	suite.Suite
	config config.OrchInstallerConfig

	step              *steps_aws.CreateAWSStateBucket
	randomText        string
	terraformExecPath string
}

func TestCreateAWSStateBucket(t *testing.T) {
	suite.Run(t, new(StateBucketTest))
}

func (s *StateBucketTest) SetupTest() {
	rootPath, err := filepath.Abs("../../../../")
	if err != nil {
		s.NoError(err)
		return
	}
	s.randomText = strings.ToLower(rand.Text()[0:8])
	s.config = config.OrchInstallerConfig{}
	s.config.AWS.Region = "us-west-2"
	s.config.Global.OrchName = "test"
	s.config.Generated.DeploymentID = s.randomText
	s.terraformExecPath, err = steps.InstallTerraformAndGetExecPath()
	if err != nil {
		s.NoError(err)
		return
	}

	s.step = &steps_aws.CreateAWSStateBucket{
		RootPath:           rootPath,
		KeepGeneratedFiles: false,
		TerraformExecPath:  s.terraformExecPath,
	}
}

func (s *StateBucketTest) TearDownTest() {
	s.config.Generated.Action = "uninstall"
	_, err := steps.GoThroughStepFunctions(s.step, &s.config)
	if err != nil {
		s.NoError(err)
	}
	if _, err := os.Stat(s.terraformExecPath); err == nil {
		err = os.Remove(s.terraformExecPath)
		if err != nil {
			s.NoError(err)
		}
	}
}

func (s *StateBucketTest) TestInstall() {
	s.config.Generated.Action = "install"
	_, err := steps.GoThroughStepFunctions(s.step, &s.config)
	if err != nil {
		s.NoError(err)
		return
	}
	expectBucketName := s.config.Global.OrchName + "-" + s.config.Generated.DeploymentID
	terratest_aws.AssertS3BucketExists(s.T(), s.config.AWS.Region, expectBucketName)
}
