package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"golang.org/x/oauth2/google" // For ADC project ID discovery
)

// getProjectIDFromEnvOrADC tries to determine the GCP project ID.
// It checks the GOOGLE_CLOUD_PROJECT environment variable first,
// then tries to get it from Application Default Credentials.
func getProjectIDFromEnvOrADC(ctx context.Context) (string, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID != "" {
		return projectID, nil
	}

	// Try to get Project ID from Application Default Credentials
	// Note: In a pure unit test environment, this might not find credentials
	// or a project ID unless the test runner environment is configured with ADC.
	credentials, err := google.FindDefaultCredentials(ctx, secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return "", fmt.Errorf("error finding default Google Cloud credentials: %w", err)
	}

	if credentials.ProjectID == "" {
		return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable is not set, and could not determine Project ID from Application Default Credentials")
	}
	return credentials.ProjectID, nil
}

// buildFullSecretVersionName constructs the full secret version name.
// It's extracted for easier testing.
func buildFullSecretVersionName(ctx context.Context, secretIdentifier string) (string, error) {
	if secretIdentifier == "" {
		return "", fmt.Errorf("secret identifier cannot be empty")
	}

	var fullSecretVersionName string

	if strings.Contains(secretIdentifier, "/") {
		baseSecretPath := secretIdentifier
		if strings.Contains(secretIdentifier, "/versions/") {
			parts := strings.SplitN(secretIdentifier, "/versions/", 2)
			baseSecretPath = parts[0]
		}
		if !strings.HasPrefix(baseSecretPath, "projects/") || strings.Count(baseSecretPath, "/") < 3 {
			return "", fmt.Errorf("invalid secret path format for '%s'. Expected 'projects/PROJECT_ID/secrets/SECRET_ID'", secretIdentifier)
		}
		fullSecretVersionName = fmt.Sprintf("%s/versions/latest", baseSecretPath)
	} else {
		projectID, err := getProjectIDFromEnvOrADC(ctx)
		if err != nil {
			return "", fmt.Errorf("could not determine Project ID for secret '%s': %w. To resolve, set GOOGLE_CLOUD_PROJECT, ensure ADC has a project, or use full path", secretIdentifier, err)
		}
		fullSecretVersionName = fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretIdentifier)
	}
	return fullSecretVersionName, nil
}

// accessSecret retrieves the secret value.
// This function encapsulates the client interaction.
// For unit testing the main logic, this part would typically be mocked.
func accessSecret(ctx context.Context, fullSecretVersionName string) ([]byte, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating Secret Manager client: %w", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fullSecretVersionName,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error accessing secret version '%s': %w", fullSecretVersionName, err)
	}

	if result.Payload == nil {
		return nil, fmt.Errorf("secret payload is unexpectedly nil for '%s'", fullSecretVersionName)
	}
	return result.Payload.Data, nil
}

func main() {
	ctx := context.Background()

	var secretIdentifierArg string
	flag.StringVar(&secretIdentifierArg, "secret-identifier", os.Getenv("SECRET_NAME"), "Secret name (e.g., 'my-secret') or full secret path (e.g., 'projects/PROJECT_ID/secrets/SECRET_ID')")
	flag.Parse() 

	if secretIdentifierArg == "" {
		fmt.Fprintln(os.Stderr, "Error: The -secret-identifier flag or SECRET_NAME environment variable must be provided and non-empty.")
		os.Exit(1) 
	}

	fullSecretVersionName, err := buildFullSecretVersionName(ctx, secretIdentifierArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2) 
	}

	secretPayload, err := accessSecret(ctx, fullSecretVersionName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(3) 
	}

	_, err = os.Stdout.Write(secretPayload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing secret payload to stdout: %v\n", err)
		os.Exit(4) 
	}
}
