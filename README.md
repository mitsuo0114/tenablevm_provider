# Tenable Vulnerability Management Terraform Provider

This repository contains a Terraform provider that integrates with the **Tenable Vulnerability Management (Tenable VM)** API. The provider is written in Go using the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework).

The provider exposes resources for managing Tenable VM users and data sources for retrieving users, roles and groups. The codebase is intended for local development and experimentation and is not an official Tenable product.

## Requirements

- Go 1.20 or later
- Terraform 1.5 or later

## Building the provider

Clone this repository and run the following command to build the provider binary:

```bash
go build -o terraform-provider-tenablevm
```

The resulting binary can be placed in your Terraform plugin directory (usually `~/.terraform.d/plugins/registry.terraform.io/tenable/tenablevm/<version>/`). For local development you can use any version string. Terraform will look for the provider binary based on the address specified in the configuration.

## Initial setup

The provider requires credentials for the Tenable VM API. These can be supplied directly in the provider block or via environment variables.

| Configuration attribute | Environment variable        | Description                                   |
|-------------------------|-----------------------------|-----------------------------------------------|
| `access_key`            | `TENABLE_ACCESS_KEY`        | API access key                                |
| `secret_key`            | `TENABLE_SECRET_KEY`        | API secret key (sensitive)                    |
| `base_url`              | N/A                         | Base URL for the API (defaults to `https://cloud.tenable.com`) |

At a minimum `access_key` and `secret_key` must be provided.

## Using the provider in Terraform

Declare the provider in your Terraform configuration:

```hcl
terraform {
  required_providers {
    tenablevm = {
      source  = "registry.terraform.io/tenable/tenablevm"
      version = "0.1.0" # or any version when developing locally
    }
  }
}

provider "tenablevm" {
  access_key = var.access_key
  secret_key = var.secret_key
  # base_url can be omitted to use the default cloud.tenable.com
}
```

### Managing users

The provider currently implements the `tenablevm_user` resource. A minimal example is shown below:

```hcl
resource "tenablevm_user" "example" {
  username    = "terraform-user"
  password    = "initialPassword123!"
  permissions = 16
  name        = "Terraform Example"
  email       = "tf@example.com"
  enabled     = true
}
```

Refer to the schema definitions in the source code for a full list of available attributes.

### Data sources

- `tenablevm_user` – Look up a user by ID or username
- `tenablevm_role` – Retrieve role details
- `tenablevm_group` – Retrieve group details

Example:

```hcl
data "tenablevm_user" "current" {
  username = "terraform-user"
}
```

## Testing

Run the Go unit tests with:

```bash
go test ./...
```

This repository uses [pre-commit](https://pre-commit.com/) to run formatting, linting and tests before each commit. Install the hooks with:

```bash
pip install pre-commit
pre-commit install
```

The hooks run `go fmt`, `go vet`, `golangci-lint`, `go mod tidy` and the unit tests with coverage enabled. Coverage results are written to `coverage.out` and uploaded in CI as a build artifact.

Note that the repository currently lacks a `go.sum` file. Running the tests may fail if the required modules cannot be downloaded.

