package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"         // 1. เพิ่ม: สำหรับจัดการไฟล์และโฟลเดอร์
	"os/exec"    // 2. เพิ่ม: สำหรับรันคำสั่ง (Terraform)
	"path/filepath" // 3. เพิ่ม: สำหรับจัดการ Path ของไฟล์
	"strings"    // 4. เพิ่ม: สำหรับรวมข้อความ
	"text/template" // 5. เพิ่ม: สำหรับสร้าง HCL Template
)

// 6. [ใหม่] สร้าง "แบบฟอร์ม" รับข้อมูล (Struct)
// เราสร้าง Struct ที่มีทุก field จากทุก platform
// `json:"..."` คือการบอก Go ว่า field นี้ใน JSON ชื่ออะไร
type TerraformRequest struct {
	Platform string `json:"platform"`

	// AWS Fields
	AwsAccessKey     string `json:"aws_access_key"`
	AwsSecretKey     string `json:"aws_secret_key"`
	AwsInstanceName  string `json:"aws_instance_name"`
	AwsRegion        string `json:"aws_region"`
	AwsInstanceType  string `json:"aws_instance_type"`

	// Azure Fields (เราจะยังไม่ใช้ แต่ประกาศไว้)
	AzureClientID       string `json:"azure_client_id"`
	AzureClientSecret string `json:"azure_client_secret"`
	AzureTenantID       string `json:"azure_tenant_id"`
	AzureSubscriptionID string `json:"azure_subscription_id"`
	AzureLocation       string `json:"azure_location"`
	AzureVmSize         string `json:"azure_vm_size"`
	
	// GCP Fields (เราจะยังไม่ใช้ แต่ประกาศไว้)
	GcpKeyJson     string `json:"gcp_key_json"`
	GcpProject     string `json:"gcp_project"`
	GcpMachineType string `json:"gcp_machine_type"`
}

// 7. [ใหม่] HCL Template สำหรับ AWS
// นี่คือ "แม่พิมพ์" HCL ของเรา
// {{.AwsInstanceName}} คือจุดที่ Go จะ "ยัดไส้" ข้อมูลเข้ามา
//
// !! หมายเหตุ: เราไม่ได้ใส่ Key ลงในนี้
// เราจะส่ง Key ผ่าน Environment Variables ตอนรัน `exec` ซึ่งปลอดภัยกว่า
const awsTemplate = `
provider "aws" {
  region = "{{.AwsRegion}}"
}

resource "aws_instance" "web" {
  // นี่คือ AMI ของ Amazon Linux 2 ใน us-east-1 (ตัวอย่าง)
  // ในระบบจริง คุณอาจต้องมี Dropdown ให้เลือก AMI ด้วย
  ami           = "ami-0c55b159cbfafe1f0" 
  instance_type = "{{.AwsInstanceType}}"

  tags = {
    Name = "{{.AwsInstanceName}}"
  }
}

output "instance_ip" {
  value = aws_instance.web.public_ip
}
`

// 8. [อัปเกรด] handlePlan กลายเป็น "ตัวแจกจ่ายงาน"
func handlePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// 9. [ใหม่] Decode JSON เข้า Struct ของเรา (ไม่ใช่ map)
	var req TerraformRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 10. [ใหม่] พิมพ์ Log ที่เป็นระเบียบมากขึ้น
	log.Printf("Received request for platform: %s", req.Platform)

	var output string // ตัวแปรเก็บผลลัพธ์
	
	// 11. [ใหม่] ใช้ switch เพื่อ "แยก" การทำงาน
	switch req.Platform {
	case "aws":
		output, err = handleAWSPlan(req)
	case "azure":
		output, err = handleAzurePlan(req)
	case "gcp":
		output, err = handleGCPPlan(req)
	default:
		output = "Error: Unknown platform"
		err = fmt.Errorf("unknown platform: %s", req.Platform)
	}

	// 12. [ใหม่] ส่งผลลัพธ์ "จริง" (จาก Terraform) กลับไป
	if err != nil {
		log.Printf("Error processing request: %v", err)
		// ส่งข้อความ Error (ซึ่งก็คือ output จาก terraform)
		http.Error(w, output, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, output) // ส่ง Output สำเร็จกลับไป
}

// 13. [ใหม่] ฟังก์ชันสำหรับรันคำสั่ง Terraform
// นี่คือ "แขนขา" ที่จะไปรัน `terraform init`, `plan`
func runCommand(dir string, env []string, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir // สั่งให้รันในโฟลเดอร์ชั่วคราว
	cmd.Env = env // ใส่ Env (เช่น AWS Keys)

	// รันคำสั่ง และ "ดักจับ" Output ทั้งหมด (ทั้ง stdout และ stderr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// ถ้า Error, ให้ส่ง Output กลับไปด้วย (เพราะมันคือข้อความ Error)
		return string(output), err
	}
	return string(output), nil
}

// 14. [ใหม่] ฟังก์ชันจัดการ AWS โดยเฉพาะ
func handleAWSPlan(req TerraformRequest) (string, error) {
	// 14.1 สร้างโฟลเดอร์ชั่วคราว (นี่คือ Sandboxing ขั้นพื้นฐาน)
	tempDir, err := os.MkdirTemp("", "tf-run-")
	if err != nil {
		return "Error creating temp dir", err
	}
	defer os.RemoveAll(tempDir) // สั่งลบโฟลเดอร์นี้เสมอ ไม่ว่าจะสำเร็จหรือล่ม

	log.Printf("Created temp dir: %s", tempDir)

	// 14.2 สร้างไฟล์ main.tf จาก Template
	tmpl, err := template.New("hcl").Parse(awsTemplate)
	if err != nil {
		return "Error parsing template", err
	}

	tfFilePath := filepath.Join(tempDir, "main.tf")
	f, err := os.Create(tfFilePath)
	if err != nil {
		return "Error creating main.tf", err
	}

	// "ยัดไส้" ข้อมูลจาก Struct (req) ลงใน Template
	err = tmpl.Execute(f, req)
	if err != nil {
		return "Error executing template", err
	}
	f.Close()

	// 14.3 เตรียม Environment Variables สำหรับ AWS
	// นี่คือวิธีที่ปลอดภัยในการส่ง Keys (แทนที่จะเขียนลงไฟล์)
	env := os.Environ() // เอา Env ปัจจุบันมา
	env = append(env, "AWS_ACCESS_KEY_ID="+req.AwsAccessKey)
	env = append(env, "AWS_SECRET_ACCESS_KEY="+req.AwsSecretKey)

	var outputLog strings.Builder // ตัวแปรไว้ "สะสม" Log

	// 14.4 รัน `terraform init`
	log.Printf("Running 'terraform init' in %s", tempDir)
	initOutput, err := runCommand(tempDir, env, "terraform", "init")
	outputLog.WriteString("--- Terraform Init ---\n")
	outputLog.WriteString(initOutput)
	outputLog.WriteString("\n")
	if err != nil {
		log.Printf("Terraform init failed: %v", err)
		return outputLog.String(), err // ส่ง Log ที่มี Error กลับไป
	}
	
	// 14.5 รัน `terraform plan`
	log.Printf("Running 'terraform plan' in %s", tempDir)
	planOutput, err := runCommand(tempDir, env, "terraform", "plan", "-no-color")
	outputLog.WriteString("--- Terraform Plan ---\n")
	outputLog.WriteString(planOutput)
	outputLog.WriteString("\n")
	if err != nil {
		log.Printf("Terraform plan failed: %v", err)
		return outputLog.String(), err // ส่ง Log ที่มี Error กลับไป
	}

	// 14.6 ถ้าสำเร็จหมด
	return outputLog.String(), nil
}

// 15. [ใหม่] ฟังก์ชัน Azure (ยังไม่ทำ)
func handleAzurePlan(req TerraformRequest) (string, error) {
	log.Println("Azure platform is not implemented yet")
	return "Azure platform is not implemented yet.", nil
}

// 16. [ใหม่] ฟังก์ชัน GCP (ยังไม่ทำ)
func handleGCPPlan(req TerraformRequest) (string, error) {
	log.Println("GCP platform is not implemented yet")
	return "GCP platform is not implemented yet.", nil
}


// 17. [เหมือนเดิม] main()
func main() {
	fs := http.FileServer(http.Dir("./"))
	http.Handle("/", fs)
	http.HandleFunc("/api/plan", handlePlan)

	log.Println("Starting server on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}