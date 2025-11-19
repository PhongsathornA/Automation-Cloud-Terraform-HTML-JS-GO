terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025" # <--- ⚠️ แก้ชื่อ Bucket ให้ตรงกับของคุณ!
    key    = "terraform.tfstate"
    region = "ap-southeast-1"
  }
}

provider "aws" {
  region = "ap-southeast-1"
  # ไม่ต้องใส่ access_key ตรงนี้ (GitHub Actions จะจัดการให้)
}

resource "aws_instance" "web_server" {
  ami           = "ami-0b3eb051c6c7936e9" # Ubuntu 20.04 (Singapore)
  instance_type = "t3.micro"

  tags = {
    Name    = "Test-By-Go"
    Project = "Cloud-Automation-Web-Generated"
  }
}
