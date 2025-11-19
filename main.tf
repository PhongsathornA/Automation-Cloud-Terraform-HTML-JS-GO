terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025" # <--- ⚠️ อย่าลืมแก้ชื่อ Bucket เป็นของคุณ!
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

resource "aws_subnet" "user_selected_subnet" {
  vpc_id            = data.aws_vpc.default.id
  cidr_block        = "172.31.250.0/24"
  availability_zone = "ap-southeast-1a"

  tags = {
    Name = "Subnet-For-Output-Test"
  }
}

resource "aws_security_group" "user_custom_sg" {
  name        = "Output-test"
  description = "Security Group managed by Terraform Web Portal"
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

  tags = {
    Name = "Output-test"
  }
}

resource "aws_instance" "web_server" {
  ami           = "ami-0b3eb051c6c7936e9"
  instance_type = "t3.micro"

  subnet_id                   = aws_subnet.user_selected_subnet.id
  vpc_security_group_ids      = [aws_security_group.user_custom_sg.id]
  associate_public_ip_address = true

  tags = {
    Name    = "Output-Test"
    Project = "Cloud-Automation-Web-Generated"
  }
}

# 👇👇👇 ส่วนที่เพิ่มเข้ามา (Outputs) 👇👇👇

output "server_public_ip" {
  description = "IP Address ของ Server ที่สร้างเสร็จ"
  value       = aws_instance.web_server.public_ip
}

output "website_url" {
  description = "ลิงก์สำหรับเข้าเว็บ (ถ้าลง Web Server แล้ว)"
  value       = "http://${aws_instance.web_server.public_ip}"
}
