// --- 1. ส่วนของการ ซ่อน/แสดง ฟอร์ม ---

// 1.1 ดึง Element ที่เราต้องใช้
const platformSelector = document.getElementById('platform');
const awsFields = document.getElementById('aws-fields');
const azureFields = document.getElementById('azure-fields');
const gcpFields = document.getElementById('gcp-fields');

// 1.2 สร้างฟังก์ชันสำหรับซ่อนฟอร์มทั้งหมด (เพื่อรีเซ็ต)
function hideAllFields() {
    if (awsFields) awsFields.style.display = 'none';
    if (azureFields) azureFields.style.display = 'none';
    if (gcpFields) gcpFields.style.display = 'none';
}

// 1.3 "ดักฟัง" เหตุการณ์เมื่อ User เปลี่ยนค่าใน Dropdown
platformSelector.addEventListener('change', function() {
    hideAllFields(); // ซ่อนทุกอย่างก่อน
    
    const selectedPlatform = this.value; // ดึงค่าที่ User เลือก
    
    // ใช้ if-else เพื่อแสดงเฉพาะฟอร์มที่ถูกต้อง
    if (selectedPlatform === 'aws') {
        awsFields.style.display = 'block';
    } else if (selectedPlatform === 'azure') {
        azureFields.style.display = 'block';
    } else if (selectedPlatform === 'gcp') {
        gcpFields.style.display = 'block';
    }
});

// --- 2. ส่วนของการเชื่อมต่อ API ---

// 2.1 ดึง Element ที่เราต้องใช้
const planButton = document.getElementById('btn-plan');
const outputLog = document.getElementById('output');
const mainForm = document.getElementById('tf-form');

// 2.2 ดักฟังเหตุการณ์เมื่อปุ่ม "Plan" ถูกคลิก
planButton.addEventListener('click', async function(event) {
    
    // 2.3 หยุดการทำงานปกติของฟอร์ม (กันหน้าเว็บโหลดใหม่)
    event.preventDefault(); 
    
    // 2.4 แสดงสถานะ "กำลังโหลด"
    outputLog.textContent = "Sending request to Go server...\n";

    // 2.5 รวบรวมข้อมูลจากฟอร์มทั้งหมด
    const formData = new FormData(mainForm);
    
    // 2.6 แปลง FormData เป็น Object ธรรมดา
    const data = {};
    formData.forEach((value, key) => {
        data[key] = value;
    });

    // 2.7 ใช้ fetch() เพื่อ "ยิง" ข้อมูลไปที่ /api/plan
    try {
        const response = await fetch('/api/plan', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            // แปลง Object เป็น JSON string ก่อนส่ง
            body: JSON.stringify(data) 
        });

        // 2.8 รอรับคำตอบกลับมา (เป็น Text)
        const resultText = await response.text();

        if (response.ok) {
            // ถ้าสำเร็จ (Go ตอบกลับมา)
            outputLog.textContent = "[SUCCESS]\n" + resultText;
        } else {
            // ถ้า Go ตอบ Error
            outputLog.textContent = "[ERROR]\n" + resultText;
        }

    } catch (error) {
        // ถ้ามีปัญหา Network (เช่น Go server ล่ม)
        outputLog.textContent = "[NETWORK ERROR]\n" + error.message;
    }
});

// 3. (เริ่มต้น) ซ่อนฟอร์มทั้งหมดเมื่อหน้าเว็บโหลดเสร็จ
//    เราใช้ DOMContentLoaded เพื่อให้แน่ใจว่า HTML โหลดเสร็จก่อน
document.addEventListener('DOMContentLoaded', hideAllFields);