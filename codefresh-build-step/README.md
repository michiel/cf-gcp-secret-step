# secure-secret-run

> A Codefresh shared step that securely pulls a secret from GCP Secret Manager and passes it to a command via `stdin` — never via environment variables or disk.

## 🔐 Key Features

- ✅ GCP Secret Manager support
- ✅ Secret never placed in environment or written to disk
- ✅ Securely piped to command via `stdin`
- ✅ Includes retry logic with exponential backoff
- ✅ Safe for use in sensitive CI/CD workflows

## 📦 Inputs

| Name         | Description                               | Required |
|--------------|-------------------------------------------|----------|
| `SECRET_NAME` | Name of the secret in GCP Secret Manager  | ✅       |
| `CMD`         | Command to run, reading secret from stdin | ✅       |

## 🚀 Example Usage in Codefresh Pipeline

```yaml
steps:
  terraform_apply:
    type: your-org/secure-secret-run
    arguments:
      SECRET_NAME: my-sensitive-secret
      CMD: terraform apply -var "my_secret=\$(cat)"
```

## 🧪 Test Locally (Optional)

```bash
export SECRET_NAME=my-secret
export CMD='jq -R .'

bash step.yaml < <(gcloud secrets versions access latest --secret="$SECRET_NAME")
```

## 📜 License

MIT
