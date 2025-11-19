package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
)

type FormData struct {
	ServerName   string
	InstanceType string
	Region       string
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/generate", handleGenerate)

	fmt.Println("Server started at http://localhost:8080")
	fmt.Println("Opening browser...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := FormData{
		ServerName:   r.FormValue("serverName"),
		InstanceType: r.FormValue("instanceType"),
		Region:       r.FormValue("region"),
	}

	const tfTemplate = `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025"  # <--- ⚠️ เช็กชื่อ Bucket ตรงนี้ให้ถูกต้อง!
    key    = "terraform.tfstate"
    region = "{{.Region}}"
  }
}

provider "aws" {
  region = "{{.Region}}"
}

data "aws_vpc" "default" {
  default = true
}

resource "aws_subnet" "my_custom_subnet" {
  vpc_id            = data.aws_vpc.default.id
  cidr_block        = "172.31.200.0/24"
  availability_zone = "{{.Region}}a"
  
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
  instance_type = "{{.InstanceType}}"
  
  subnet_id              = aws_subnet.my_custom_subnet.id
  vpc_security_group_ids = [aws_security_group.allow_web_ssh.id]
  associate_public_ip_address = true

  tags = {
    Name    = "{{.ServerName}}"
    Project = "Cloud-Automation-Full-Stack"
  }
}
`

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

	// 👇 ปรับแก้ HTML ตรงนี้ครับ ใส่คำเตือนและคำสั่งให้ครบชุด
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<div style="font-family: sans-serif; text-align: center; padding: 40px; max-width: 600px; margin: auto;">
			<h1 style="color: #28a745;">✅ สร้างไฟล์สำเร็จ!</h1>
			
			<div style="background: #fff3cd; color: #856404; padding: 15px; border: 1px solid #ffeeba; border-radius: 5px; margin-bottom: 20px; text-align: left;">
				<strong>⚠️ สำคัญมาก:</strong> เพื่อป้องกัน GitHub Actions แจ้ง Error <br>
				กรุณารันคำสั่ง <code>terraform fmt</code> ทุกครั้งก่อนส่งงาน!
			</div>

			<p>ก๊อปปี้คำสั่งด้านล่างไปวางใน Terminal ของ VS Code:</p>
			
			<div style="background: #2d2d2d; color: #f8f8f2; padding: 20px; border-radius: 10px; text-align: left; font-family: monospace; font-size: 1.1em;">
				<span style="color: #a6e22e;">terraform fmt</span> <span style="color: #75715e;"># จัดระเบียบโค้ดให้สวย</span><br>
				<span style="color: #a6e22e;">git add .</span><br>
				<span style="color: #a6e22e;">git commit -m "Update infrastructure"</span><br>
				<span style="color: #a6e22e;">git push</span>
			</div>

			<br><br>
			<a href="/" style="padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 5px; font-weight: bold;">⬅️ กลับหน้าแรก</a>
		</div>
	`)
	
	fmt.Printf("Generated Terraform for: %s\n", data.ServerName)
}