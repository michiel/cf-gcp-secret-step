# GCP Secret Fetcher (secretfetcher)

`secretfetcher` is a command-line utility written in Go that retrieves the latest version of a secret from Google Cloud Secret Manager and prints its payload to standard output. It is designed as a lightweight replacement for specific `gcloud secrets versions access latest --secret="<SECRET_NAME>" --quiet` commands, suitable for use in CI/CD pipelines or environments where `gcloud` might be too heavy or unavailable.

## Features

-   Fetches the latest version of a specified secret.
-   Supports both short secret names (e.g., `my-secret`) and full secret resource paths (e.g., `projects/my-project/secrets/my-secret`).
-   If a short secret name is provided, the Google Cloud Project ID is determined from the `GOOGLE_CLOUD_PROJECT` environment variable or Application Default Credentials (ADC).
-   Prints only the raw secret payload to `stdout`, similar to `gcloud ... --quiet`.
-   Error messages are printed to `stderr`.
-   Exits with a non-zero status code on failure.

## Prerequisites

-   Go 1.18 or later (for building).
-   Configured Google Cloud Application Default Credentials (ADC) with permissions to access the desired secrets (`secretmanager.versions.access` IAM permission). This is typically handled by the environment where the tool is run (e.g., a GCP VM, GKE pod with Workload Identity, or local `gcloud auth application-default login`).

## Installation / Building

1.  **Clone the repository or download the source files (`secretfetcher.go`, `secretfetcher_test.go`, `go.mod`).**

2.  **Initialize Go module (if not already present, e.g., if you only downloaded .go files):**
    ```bash
    go mod init <your_module_name_e.g._secretfetcher>
    go mod tidy
    ```
    (If `go.mod` is already provided, `go mod tidy` is sufficient to ensure dependencies are correct).

3.  **Build the executable:**
    ```bash
    go build -o secretfetcher .
    ```
    This will create an executable named `secretfetcher` in the current directory.

    For a static, platform-specific binary (e.g., for Linux Alpine in Docker):
    ```bash
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s -w' -o secretfetcher_linux_amd64 .
    ```
    The `-ldflags '-s -w'` strip debugging information and symbol table, making the binary smaller.

## Usage

```bash
./secretfetcher -secret-identifier="<SECRET_IDENTIFIER>"
```

Or, if `SECRET_NAME` environment variable is set:

```bash
export SECRET_NAME="<SECRET_IDENTIFIER>"
./secretfetcher
```

**Arguments:**

-   `-secret-identifier="<SECRET_IDENTIFIER>"`: (Required if `SECRET_NAME` env var is not set) The identifier for the secret.
    -   Can be the short name of the secret (e.g., `my-api-key`). In this case, `GOOGLE_CLOUD_PROJECT` environment variable must be set, or the project ID must be discoverable via ADC.
    -   Can be the full resource path of the secret (e.g., `projects/my-gcp-project-id/secrets/my-api-key`).
    -   If the path includes a version (e.g., `projects/my-gcp-project-id/secrets/my-api-key/versions/2`), the program will still attempt to fetch the `latest` version of the base secret.

**Environment Variables:**

-   `SECRET_NAME`: If `-secret-identifier` is not provided, this environment variable will be used as the secret identifier.
-   `GOOGLE_CLOUD_PROJECT`: Used to determine the project ID if a short secret name is provided and the project cannot be inferred from ADC.

**Example:**

```bash
# Using a short secret name (GOOGLE_CLOUD_PROJECT must be set or discoverable by ADC)
export GOOGLE_CLOUD_PROJECT="your-actual-project-id"
./secretfetcher -secret-identifier="my-database-password" > /tmp/db_pass.txt

# Using a full secret path
./secretfetcher -secret-identifier="projects/your-actual-project-id/secrets/another-api-key"

# Using the environment variable for the secret name
export SECRET_NAME="projects/your-actual-project-id/secrets/yet-another-secret"
export GOOGLE_CLOUD_PROJECT="your-actual-project-id" # May not be needed if SECRET_NAME is full path
./secretfetcher
```

## Running Tests

To run the unit tests:

```bash
go test -v ./...
```
The tests primarily cover the logic for parsing secret identifiers and determining project IDs. Testing the actual interaction with GCP Secret Manager (`accessSecret` function) would require mocking the GCP client or setting up integration tests against a live GCP environment or emulator.

## Dependencies

This tool uses the following Go modules:
-   `cloud.google.com/go/secretmanager/apiv1`
-   `golang.org/x/oauth2/google`

These are managed by Go modules and will be downloaded automatically when you run `go mod tidy` or `go build`.

## Exit Codes

-   `0`: Success.
-   `1`: Missing required input (`-secret-identifier` flag or `SECRET_NAME` environment variable).
-   `2`: Error in configuration or constructing the secret name (e.g., invalid path, project ID not found).
-   `3`: Error accessing the secret from GCP Secret Manager (e.g., permission denied, secret not found).
-   `4`: Error writing the secret payload to standard output.

## How it Works in a Codefresh Step

This tool can replace a `gcloud` command in a Codefresh step.

Original `gcloud` command in `commands` section of a step:

```yaml
# ...
commands:
  - gcloud secrets versions access latest --secret="$SECRET_NAME_VAR" --quiet
# ...
```

Using `secretfetcher` (assuming the binary is in PATH or `./secretfetcher`):
1.  Ensure `secretfetcher` binary is available in your step's image or build it within the step.
2.  Set `SECRET_NAME` environment variable or pass `-secret-identifier`.

```yaml
# ...
commands:
  # Option 1: Using environment variable (if SECRET_NAME_VAR is the identifier)
  - export SECRET_NAME="$SECRET_NAME_VAR"
  - ./secretfetcher # Assumes GOOGLE_CLOUD_PROJECT is set if SECRET_NAME_VAR is a short name
  # Option 2: Using command-line flag
  - ./secretfetcher -secret-identifier="$SECRET_NAME_VAR" # Assumes GOOGLE_CLOUD_PROJECT is set if SECRET_NAME_VAR is a short name
# ...
```

The output (the secret) will be on `stdout`, which can then be piped or redirected as needed, just like with the `gcloud` command.
For example, to use in the Codefresh step from the original prompt:

```yaml
# ...
fetch_secret_from_gcp() {
  local i
  for i in {1..5}; do
    # Using the compiled Go binary:
    if ./secretfetcher -secret-identifier="$SECRET_NAME"; then # SECRET_NAME is the input to the Codefresh step
      return 0
    else
      # secretfetcher prints errors to stderr, which will be visible in logs
      if [ "$i" -lt 5 ]; then
        echo "Warning: Attempt $i to fetch secret '$SECRET_NAME' failed. Retrying in $((i * 2)) seconds..." >&2
        sleep $((i * 2))
      else
        echo "Error: Attempt $i (final) to fetch secret '$SECRET_NAME' failed." >&2
      fi
    fi
  done
  echo "Error: All attempts to fetch secret '$SECRET_NAME' from GCP Secret Manager failed." >&2
  return 1
}

bash -c "$USER_COMMAND_SCRIPT" < <(fetch_secret_from_gcp)
# ...
```

Make sure the environment where `secretfetcher` runs has Application Default Credentials configured with the necessary permissions.
