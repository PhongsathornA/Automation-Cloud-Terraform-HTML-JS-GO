terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025" # <--- ⚠️ แก้ชื่อ Bucket ของคุณตรงนี้!
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
    Name = "Subnet-For-Test-Nginx-2"
  }
}

resource "aws_security_group" "user_custom_sg" {
  name        = "Nginx-Test-2"
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
    Name = "Nginx-Test-2"
  }
}

resource "aws_instance" "web_server" {
  ami           = "ami-0b3eb051c6c7936e9"
  instance_type = "t3.micro"

  subnet_id                   = aws_subnet.user_selected_subnet.id
  vpc_security_group_ids      = [aws_security_group.user_custom_sg.id]
  associate_public_ip_address = true

  # 👇👇👇 ปรับแก้ Script สำหรับ Amazon Linux 👇👇👇

  user_data = <<-EOF
              #!/bin/bash
              # ใช้ dnf แทน apt-get (เพราะเป็น Amazon Linux)
              dnf update -y
              dnf install -y nginx
              
              systemctl start nginx
              systemctl enable nginx
              
              # Amazon Linux เก็บหน้าเว็บไว้ที่ /usr/share/nginx/html
              echo "<h1>☁️ Hello from Amazon Linux!</h1><p>Server: Test-Nginx-2</p>" > /usr/share/nginx/html/index.html
              EOF

  user_data_replace_on_change = true


  tags = {
    Name    = "Test-Nginx-2"
    Project = "Cloud-Automation-Web-Generated"
  }
}

output "server_public_ip" {
  value = aws_instance.web_server.public_ip
}

output "website_url" {
  value = "http://${aws_instance.web_server.public_ip}"
}
