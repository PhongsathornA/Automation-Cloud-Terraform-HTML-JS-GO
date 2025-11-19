terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025" # <--- ⚠️ เช็กชื่อ Bucket ตรงนี้ให้ถูกต้อง!
    key    = "terraform.tfstate"
    region = "ap-southeast-1"
  }
}

provider "aws" {
  region = "ap-southeast-1"
}

data "aws_vpc" "default" {
  default = true
}

resource "aws_subnet" "my_custom_subnet" {
  vpc_id            = data.aws_vpc.default.id
  cidr_block        = "172.31.200.0/24"
  availability_zone = "ap-southeast-1a"

  tags = {
    Name = "My-Custom-Subnet-By-Go"
  }
}

resource "aws_security_group" "allow_web_ssh" {
  name        = "allow_web_ssh_by_go"
  description = "Allow Web (80) and SSH (22)"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "web_server" {
  ami           = "ami-0b3eb051c6c7936e9"
  instance_type = "t3.micro"

  subnet_id                   = aws_subnet.my_custom_subnet.id
  vpc_security_group_ids      = [aws_security_group.allow_web_ssh.id]
  associate_public_ip_address = true

  tags = {
    Name    = "Test2-By-Go"
    Project = "Cloud-Automation-Full-Stack"
  }
}
