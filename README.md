<center><p style="text-align:center;"><img src="logo.png" alt="Logo", width="125"></p></center>
[![netlify status](https://api.netlify.com/api/v1/badges/20b08937-99d0-4882-b9a3-d5f09ddd29b7/deploy-status)](https://app.netlify.com/sites/tangerine-licorice-5f7cbc/deploys)

# Terraform registry proxy with prometheus

## How to setup terraform registry proxy

- Create `config.ini` file in the current directory with the following content:
```ini
[registry]
proxy_host = "your-registry.terraform-registry.example.com"
[release]
proxy_host = "your-release.terraform-registry.example.com"
path_prefix = ""
terraform_version = "1.1.7"
[server]
address = ":5000"
is_private = false # if true, only the proxy_host will be allowed to access the registry and release by certificate
cert_file = "" # optional
key_file = ""  # optional
use_tls = false # optional
```

- Build container images with podman
```shell
podman build -t registry-proxy:dev .
```

- Install container with podman/docker
```shell
podman run --name terraform-registry-proxy -itd -p 5000:5000 registry-proxy:dev
```

After you have your infrastructure setup you need to update your Terraform
configuration so it knows to pull dependencies through the proxy.

## Modify `provider.tf` switch to proxy mode
```terraform
terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/aws"
      version = "=< 4.16.0"
    }
  }
}

# Configure the Microsoft Azure Provider
provider "aws" {
  region = "cn-north-1"
  profile = "default"
}
```

You would update it to this

```terraform
terraform {
  required_providers {
    azurerm = {
      source  = "your-registry.terraform-registry.example.com/hashicorp/aws"
      version = "=< 4.16.0"
    }
  }
}

# Configure the Microsoft Azure Provider
provider "aws" {
  region = "cn-north-1"
  profile = "default"
}
```

## Supported Cloud Platform

### AWS Lambda Container
[![amplifybutton](https://cloudbriefly.com/img/post/0019/0019_00.svg)](https://console.aws.amazon.com/amplify/home#/deploy?repo=https://github.com/username/repository)
### Salesforce Heroku
[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)
