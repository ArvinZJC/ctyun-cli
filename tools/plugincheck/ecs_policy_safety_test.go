/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import "testing"

// TestECSPolicyMutationsRequireConfirmation protects security policy changes
// from being submitted without the shared dangerous-command confirmation.
func TestECSPolicyMutationsRequireConfirmation(t *testing.T) {
	context := loadStorageReviewContext(t, "ecs")
	for _, action := range []string{"create", "update", "delete"} {
		operationID := "v4.ecs.remote-attestation.policy." + action
		command := context.commands[operationID]
		if command.Dangerous.Confirm != "yes" {
			t.Errorf("%s must require confirmation", operationID)
		}
	}
}
