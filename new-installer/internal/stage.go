// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package internal

import "context"

type OrchInstallerStage interface {
	Name() string
	// PreStage: initialize the stage, such as creating directories, downloading files, etc.
	// It also process the output/runtime-state from previous stage.
	PreStage(ctx context.Context, config OrchInstallerConfig, runtimeState *OrchInstallerRuntimeState) *OrchInstallerStageError

	// RunStage: run the stage, such as running terraform, ansible, etc.
	RunStage(ctx context.Context, config OrchInstallerConfig, runtimeState *OrchInstallerRuntimeState) *OrchInstallerStageError

	// PostStage: cleanup the stage, such as removing directories, files, etc.
	// It should also handle the error from the previous stage and rollback if needed.
	// It should also return the final output of the stage.
	PostStage(ctx context.Context, config OrchInstallerConfig, runtimeState *OrchInstallerRuntimeState, prevStageError *OrchInstallerStageError) *OrchInstallerStageError
}
