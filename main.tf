
terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.0" }
  }
  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025" # <--- ⚠️ แก้ชื่อ Bucket ให้ถูก
    key    = "dev-terraform.tfstate"             # ใช้ Dev State
    region = "ap-southeast-1"
  }
}

provider "aws" { region = "ap-southeast-1" }
data "aws_vpc" "default" { default = true }

# Network
resource "aws_subnet" "sub_a" {
  vpc_id            = data.aws_vpc.default.id
  cidr_block        = "172.31.201.0/24"
  availability_zone = "ap-southeast-1a"
  tags              = { Name = "Subnet-A-Test-DB" }
}
resource "aws_subnet" "sub_b" {
  vpc_id            = data.aws_vpc.default.id
  cidr_block        = "172.31.202.0/24"
  availability_zone = "ap-southeast-1b"
  tags              = { Name = "Subnet-B-Test-DB" }
}

# Security Group
resource "aws_security_group" "alb_sg" {
  name   = "Test-DB"
  vpc_id = data.aws_vpc.default.id

  # HTTP
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }


  # MySQL / MariaDB Port (เพิ่มเฉพาะตอนเลือก DB)
  ingress {
    description = "Database Port"
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # ใน Production ควรระบุ IP
  }


  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# Load Balancer
resource "aws_lb" "app_lb" {
  name               = "alb-Test-DB"
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb_sg.id]
  subnets            = [aws_subnet.sub_a.id, aws_subnet.sub_b.id]
}
resource "aws_lb_target_group" "app_tg" {
  name     = "tg-Test-DB"
  port     = 80
  protocol = "HTTP"
  vpc_id   = data.aws_vpc.default.id
}
resource "aws_lb_listener" "front_end" {
  load_balancer_arn = aws_lb.app_lb.arn
  port              = "80"
  protocol          = "HTTP"
  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app_tg.arn
  }
}

# Launch Template & ASG
resource "aws_launch_template" "app_lt" {
  name_prefix   = "lt-Test-DB"
  image_id      = "ami-0b3eb051c6c7936e9" # Amazon Linux 2023
  instance_type = "t3.micro"

  network_interfaces {
    associate_public_ip_address = true
    security_groups             = [aws_security_group.alb_sg.id]
  }

  user_data = base64encode(<<-EOF
              #!/bin/bash
              dnf update -y
              
              
              # Install Nginx
              dnf install -y nginx
              systemctl start nginx
              systemctl enable nginx
              echo "<h1>Hello from Test-DB</h1>" > /usr/share/nginx/html/index.html
              

              
              # Install MariaDB (MySQL)
              dnf install -y mariadb105-server
              systemctl start mariadb
              systemctl enable mariadb
              
              # สร้าง Database ทดสอบ (User: admin / Pass: Pass1234!)
              mysql -e "CREATE DATABASE my_app_db;"
              mysql -e "CREATE USER 'admin'@'%' IDENTIFIED BY 'Pass1234!';"
              mysql -e "GRANT ALL PRIVILEGES ON *.* TO 'admin'@'%';"
              
              EOF
  )
}

resource "aws_autoscaling_group" "app_asg" {
  desired_capacity    = 2
  max_size            = 2
  min_size            = 2
  vpc_zone_identifier = [aws_subnet.sub_a.id, aws_subnet.sub_b.id]
  target_group_arns   = [aws_lb_target_group.app_tg.arn]
  launch_template {
    id      = aws_launch_template.app_lt.id
    version = "$Latest"
  }
}

output "alb_dns_name" {
  value = "http://${aws_lb.app_lb.dns_name}"
}
