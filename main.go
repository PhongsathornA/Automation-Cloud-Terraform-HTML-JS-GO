package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
)

type FormData struct {
	Provider       string
	ResourceName   string
	
	// AWS Fields
	AWSInstanceType string
	AWSCapacity     string
	AWSSgName       string
	InstallNginx    bool
	InstallDb       bool
	
	// Azure Fields
	AzureLocation   string
	AzureVmSize     string
	AzureRgName     string
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/generate", handleGenerate)

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := FormData{
		Provider:        r.FormValue("provider"),
		ResourceName:    r.FormValue("resourceName"),
		
		AWSInstanceType: r.FormValue("awsInstanceType"),
		AWSCapacity:     r.FormValue("awsCapacity"),
		AWSSgName:       r.FormValue("awsSgName"),
		InstallNginx:    r.FormValue("installNginx") == "yes",
		InstallDb:       r.FormValue("installDb") == "yes",

		AzureLocation:   r.FormValue("azureLocation"),
		AzureVmSize:     r.FormValue("azureVmSize"),
		AzureRgName:     r.FormValue("azureRgName"),
	}

	var tfTemplate string
	if data.Provider == "aws" {
		tfTemplate = awsClusterTemplate
	} else {
		tfTemplate = azureVmTemplate
	}

	tmpl, err := template.New("terraform").Parse(tfTemplate)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	file, err := os.Create("main.tf")
	if err != nil {
		http.Error(w, "Error creating file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	err = tmpl.Execute(file, data)
	if err != nil {
		http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<div style="font-family: sans-serif; text-align: center; padding: 50px;">
			<h1 style="color: #28a745;">✅ Generated %s Config Success!</h1>
			<p>สร้างไฟล์ <strong>main.tf</strong> เรียบร้อยแล้ว</p>
            <p><strong>Features:</strong> Nginx=%t, Database=%t</p>
			<div style="background: #f1f1f1; padding: 20px; border-radius: 10px; display: inline-block; text-align: left;">
				<code>
				terraform fmt<br>
				git add .<br>
				git commit -m "Update infrastructure for %s"<br>
				git push
				</code>
			</div>
			<br><br>
			<a href="/">⬅️ Back to Home</a>
		</div>
	`, data.Provider, data.InstallNginx, data.InstallDb, data.Provider)
	
	fmt.Printf("Generated for %s: %s (DB=%t)\n", data.Provider, data.ResourceName, data.InstallDb)
}

// --- 1. แม่พิมพ์ AWS (HA Cluster + DB + Show ID) ---
const awsClusterTemplate = `
terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.0" }
  }
  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025" # <--- ⚠️ แก้ชื่อ Bucket ให้ถูก
    key    = "dev-terraform.tfstate"
    region = "ap-southeast-1"
  }
}

provider "aws" { region = "ap-southeast-1" }
data "aws_vpc" "default" { default = true }

resource "aws_subnet" "sub_a" {
  vpc_id = data.aws_vpc.default.id
  cidr_block = "172.31.201.0/24"
  availability_zone = "ap-southeast-1a"
  tags = { Name = "Subnet-A-{{.ResourceName}}" }
}
resource "aws_subnet" "sub_b" {
  vpc_id = data.aws_vpc.default.id
  cidr_block = "172.31.202.0/24"
  availability_zone = "ap-southeast-1b"
  tags = { Name = "Subnet-B-{{.ResourceName}}" }
}

resource "aws_security_group" "alb_sg" {
  name = "{{.AWSSgName}}"
  vpc_id = data.aws_vpc.default.id

  ingress {
    from_port = 80
    to_port = 80
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  {{if .InstallDb}}
  ingress {
    description = "Database Port"
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  {{end}}

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_lb" "app_lb" {
  name = "alb-{{.ResourceName}}"
  load_balancer_type = "application"
  security_groups = [aws_security_group.alb_sg.id]
  subnets = [aws_subnet.sub_a.id, aws_subnet.sub_b.id]
}

resource "aws_lb_target_group" "app_tg" {
  name = "tg-{{.ResourceName}}"
  port = 80
  protocol = "HTTP"
  vpc_id = data.aws_vpc.default.id
}

resource "aws_lb_listener" "front_end" {
  load_balancer_arn = aws_lb.app_lb.arn
  port = "80"
  protocol = "HTTP"
  default_action {
    type = "forward"
    target_group_arn = aws_lb_target_group.app_tg.arn
  }
}

resource "aws_launch_template" "app_lt" {
  name_prefix = "lt-{{.ResourceName}}"
  image_id = "ami-0b3eb051c6c7936e9"
  instance_type = "{{.AWSInstanceType}}"
  
  network_interfaces {
    associate_public_ip_address = true
    security_groups = [aws_security_group.alb_sg.id]
  }

  user_data = base64encode(<<-EOF
              #!/bin/bash
              dnf update -y
              
              {{if .InstallNginx}}
              # Install Nginx
              dnf install -y nginx
              systemctl start nginx
              systemctl enable nginx
              
              # 👇 ส่วนนี้แหละครับที่ดึงเลข ID มาโชว์ 👇
              TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
              INSTANCE_ID=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" -s http://169.254.169.254/latest/meta-data/instance-id)
              AZ=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" -s http://169.254.169.254/latest/meta-data/placement/availability-zone)
              
              # เขียนไฟล์ HTML
              cat <<HTML > /usr/share/nginx/html/index.html
              <!DOCTYPE html>
              <html>
              <head>
                  <style>
                      body { font-family: sans-serif; text-align: center; padding-top: 50px; background: #f4f4f4; }
                      .container { background: white; padding: 40px; border-radius: 10px; display: inline-block; box-shadow: 0 4px 15px rgba(0,0,0,0.1); }
                      h1 { color: #2c3e50; }
                      .info { color: #e67e22; font-weight: bold; font-size: 1.2em; }
                      .zone { color: #2980b9; font-weight: bold; }
                  </style>
              </head>
              <body>
                  <div class="container">
                      <h1>Hello from {{.ResourceName}}</h1>
                      <p>Served by Instance ID: <span class="info">$INSTANCE_ID</span></p>
                      <p>Availability Zone: <span class="zone">$AZ</span></p>
                      <hr>
                      <small>Deployed via Terraform & Go</small>
                  </div>
              </body>
              </html>
              HTML
              {{end}}

              {{if .InstallDb}}
              # Install MariaDB
              dnf install -y mariadb105-server
              systemctl start mariadb
              systemctl enable mariadb
              mysql -e "CREATE DATABASE my_app_db;"
              mysql -e "CREATE USER 'admin'@'%' IDENTIFIED BY 'Pass1234!';"
              mysql -e "GRANT ALL PRIVILEGES ON *.* TO 'admin'@'%';"
              {{end}}
              EOF
  )
}

resource "aws_autoscaling_group" "app_asg" {
  desired_capacity = {{.AWSCapacity}}
  max_size = {{.AWSCapacity}}
  min_size = {{.AWSCapacity}}
  vpc_zone_identifier = [aws_subnet.sub_a.id, aws_subnet.sub_b.id]
  target_group_arns = [aws_lb_target_group.app_tg.arn]
  launch_template {
    id = aws_launch_template.app_lt.id
    version = "$Latest"
  }
}

output "alb_dns_name" {
  value = "http://${aws_lb.app_lb.dns_name}"
}
`

// --- 2. แม่พิมพ์ Azure (เหมือนเดิม) ---
const azureVmTemplate = `
terraform {
  required_providers {
    azurerm = { source = "hashicorp/azurerm", version = "~> 3.0" }
  }
  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025"
    key    = "dev-azure.tfstate"
    region = "ap-southeast-1"
  }
}

provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "rg" {
  name     = "{{.AzureRgName}}"
  location = "{{.AzureLocation}}"
}

resource "azurerm_virtual_network" "vnet" {
  name                = "vnet-{{.ResourceName}}"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
}

resource "azurerm_subnet" "subnet" {
  name                 = "internal"
  resource_group_name  = azurerm_resource_group.rg.name
  virtual_network_name = azurerm_virtual_network.vnet.name
  address_prefixes     = ["10.0.2.0/24"]
}

resource "azurerm_network_interface" "nic" {
  name                = "nic-{{.ResourceName}}"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_linux_virtual_machine" "vm" {
  name                = "{{.ResourceName}}"
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  size                = "{{.AzureVmSize}}"
  
  admin_username                  = "adminuser"
  admin_password                  = "P@ssw0rd1234!" 
  disable_password_authentication = false

  network_interface_ids = [azurerm_network_interface.nic.id]

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "18.04-LTS"
    version   = "latest"
  }
}
`