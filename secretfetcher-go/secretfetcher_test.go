package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestGetProjectIDFromEnvOrADC tests the getProjectIDFromEnvOrADC function.
func TestGetProjectIDFromEnvOrADC(t *testing.T) {
	ctx := context.Background()

	originalProjectEnv := os.Getenv("GOOGLE_CLOUD_PROJECT")
	defer os.Setenv("GOOGLE_CLOUD_PROJECT", originalProjectEnv)

	// Case 1: GOOGLE_CLOUD_PROJECT is set
	expectedProjectID := "test-project-from-env"
	os.Setenv("GOOGLE_CLOUD_PROJECT", expectedProjectID)
	projectID, err := getProjectIDFromEnvOrADC(ctx)
	if err != nil {
		t.Errorf("TestGetProjectIDFromEnvOrADC (env set): unexpected error: %v", err)
	}
	if projectID != expectedProjectID {
		t.Errorf("TestGetProjectIDFromEnvOrADC (env set): expected project ID '%s', got '%s'", expectedProjectID, projectID)
	}

	// Case 2: GOOGLE_CLOUD_PROJECT is not set, expect ADC lookup to fail in typical test env
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")
	_, err = getProjectIDFromEnvOrADC(ctx)
	if err == nil {
		// This might pass if ADC is actually configured and finds a project.
		// For a strict unit test where we want to ensure the "not found" path,
		// this would require more sophisticated mocking of the google.FindDefaultCredentials call.
		t.Log("TestGetProjectIDFromEnvOrADC (env unset): received no error, ADC might be configured. This test path assumes ADC also fails to find credentials.")
	} else {
		// Check if the error message indicates that ADC failed.
		// The specific error from google.FindDefaultCredentials when none are found.
		expectedErrorPart := "could not find default credentials"
		if !strings.Contains(err.Error(), expectedErrorPart) {
			t.Errorf("TestGetProjectIDFromEnvOrADC (env unset): expected error to contain '%s', got: %v", expectedErrorPart, err)
		}
	}
}

// TestBuildFullSecretVersionName tests the buildFullSecretVersionName function.
func TestBuildFullSecretVersionName(t *testing.T) {
	ctx := context.Background()
	originalProjectEnv := os.Getenv("GOOGLE_CLOUD_PROJECT")
	defer os.Setenv("GOOGLE_CLOUD_PROJECT", originalProjectEnv)

	testCases := []struct {
		name             string
		secretIdentifier string
		projectEnv       string 
		wantName         string
		wantErr          bool
		wantErrMsgPart   string 
	}{
		{
			name:             "short name with project env",
			secretIdentifier: "my-secret",
			projectEnv:       "test-project-1",
			wantName:         "projects/test-project-1/secrets/my-secret/versions/latest",
			wantErr:          false,
		},
		{
			name:             "full path",
			secretIdentifier: "projects/test-project-2/secrets/another-secret",
			projectEnv:       "", 
			wantName:         "projects/test-project-2/secrets/another-secret/versions/latest",
			wantErr:          false,
		},
		{
			name:             "full path with version specified (should use latest)",
			secretIdentifier: "projects/test-project-3/secrets/versioned-secret/versions/3",
			projectEnv:       "", 
			wantName:         "projects/test-project-3/secrets/versioned-secret/versions/latest",
			wantErr:          false,
		},
		{
			name:             "empty secret identifier",
			secretIdentifier: "",
			projectEnv:       "any-project",
			wantErr:          true,
			wantErrMsgPart:   "secret identifier cannot be empty",
		},
		{
			name:             "short name without project env (expect error from getProjectID due to ADC fail)",
			secretIdentifier: "my-secret-no-env",
			projectEnv:       "", 
			wantErr:          true,
			// This error is wrapped by buildFullSecretVersionName, but the root cause is ADC failure.
			// The getProjectIDFromEnvOrADC will return an error containing "could not find default credentials".
			// buildFullSecretVersionName wraps this. We check for the ADC specific part.
			wantErrMsgPart:   "could not find default credentials", 
		},
		{
			name:             "invalid full path format (too few parts)",
			secretIdentifier: "projects/test-project-4",
			projectEnv:       "",
			wantErr:          true,
			wantErrMsgPart:   "invalid secret path format",
		},
		{
			name:             "invalid full path format (not starting with projects/)",
			secretIdentifier: "secrets/test-project-5/secrets/my-secret",
			projectEnv:       "",
			wantErr:          true,
			wantErrMsgPart:   "invalid secret path format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.projectEnv != "" {
				os.Setenv("GOOGLE_CLOUD_PROJECT", tc.projectEnv)
			} else {
				os.Unsetenv("GOOGLE_CLOUD_PROJECT")
			}

			gotName, err := buildFullSecretVersionName(ctx, tc.secretIdentifier)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected an error for '%s', but got nil", tc.secretIdentifier)
				} else if tc.wantErrMsgPart != "" && !strings.Contains(err.Error(), tc.wantErrMsgPart) {
					t.Errorf("expected error for '%s' to contain '%s', got: %v", tc.secretIdentifier, tc.wantErrMsgPart, err)
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error for '%s', but got: %v", tc.secretIdentifier, err)
				}
				if gotName != tc.wantName {
					t.Errorf("expected name '%s', but got '%s'", tc.wantName, gotName)
				}
			}
		})
	}
}
