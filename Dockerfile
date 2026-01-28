FROM golang:1.22-bookworm

# 1. ติดตั้ง gnupg และเครื่องมือที่จำเป็น
RUN apt-get update && apt-get install -y gnupg && rm -rf /var/lib/apt/lists/*

# 2. สร้าง user 'vault' เพื่อความปลอดภัย (ไม่รันด้วย root)
RUN useradd -m -u 1000 vault

# 3. กำหนดตัวแปรสภาพแวดล้อมสำหรับ GPG Home
ENV GNUPGHOME=/home/vault/.gnupg

# 4. เตรียมไดเรกทอรีสำหรับ GPG และตั้งค่าสิทธิ์ (Permissions)
# GPG เข้มงวดเรื่องสิทธิ์ของโฟลเดอร์มาก ต้องเป็น 700 เท่านั้น
RUN mkdir -p /home/vault/.gnupg && \
    chown -R vault:vault /home/vault && \
    chmod 700 /home/vault/.gnupg

# 5. ตั้งค่าการทำงานของ GPG Agent ให้รองรับการทำงานแบบไร้หน้าจอ (Headless)
RUN echo "allow-loopback-pinentry" > /home/vault/.gnupg/gpg-agent.conf && \
    echo "pinentry-mode loopback" > /home/vault/.gnupg/gpg.conf && \
    chown vault:vault /home/vault/.gnupg/gpg-agent.conf /home/vault/.gnupg/gpg.conf

# 6. กำหนด Working Directory
WORKDIR /app

# 7. คัดลอกไฟล์ทั้งหมด (รวมถึง main.go และ KBankH2HPgpUAT.asc)
COPY . .

# 8. บิลด์แอปพลิเคชัน และจัดการสิทธิ์ไฟล์กุญแจที่มีอยู่แล้ว
RUN go build -o vaultseal && \
    chown vault:vault vaultseal KBankH2HPgpUAT.asc

# 9. สลับไปใช้ user vault
USER vault

# 10. เปิดพอร์ตสำหรับ API
EXPOSE 8080

# 11. รันแอปพลิเคชัน
CMD ["./vaultseal"]