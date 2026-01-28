package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

const gpgDir = "/home/vault/.gnupg"

func generateKeyHandler(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email required", 400)
		return
	}

	config := []string{
		"Key-Type: 1", // RSA
		"Key-Length: 2048",
		"Subkey-Type: 1",
		"Subkey-Length: 2048",
		"Name-Real: Vault User",
		"Name-Email: " + email,
		"Expire-Date: 0",
		"%no-protection",
		"%commit",
	}
	batchConfig := strings.Join(config, "\n") + "\n"

	// ใช้ --homedir บังคับไปที่ path ของเรา
	cmd := exec.Command("gpg", "--homedir", gpgDir, "--batch", "--gen-key")
	cmd.Stdin = strings.NewReader(batchConfig)

	if out, err := cmd.CombinedOutput(); err != nil {
		http.Error(w, "Generate Error: "+string(out), 500)
		return
	}

	// บังคับให้ GPG เขียนข้อมูลลง disk ทันที
	exec.Command("gpg", "--homedir", gpgDir, "--check-trustdb").Run()

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Key created for: %s\n", email)
}

func encryptHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	file, _, _ := r.FormFile("file")
	if file != nil {
		defer file.Close()
	}

	// ใช้ --trust-model always เพื่อเลี่ยงปัญหา trust db
	cmd := exec.Command("gpg",
		"--homedir", gpgDir,
		"--batch",
		"--encrypt",
		"--recipient", email,
		"--trust-model", "always",
		"--output", "-",
	)

	cmd.Stdin = file
	out, err := cmd.CombinedOutput()
	if err != nil {
		list, _ := exec.Command("gpg", "--homedir", gpgDir, "--list-keys").CombinedOutput()
		http.Error(w, fmt.Sprintf("Error: %s\n\nKeys in %s:\n%s", string(out), gpgDir, string(list)), 500)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(out)
}

func encryptWithLocalKeyHandler(w http.ResponseWriter, r *http.Request) {
	// 1. ระบุชื่อไฟล์กุญแจที่มีอยู่ในโปรเจค
	keyPath := "KBankH2HPgpUAT.asc"

	// 2. ตรวจสอบอีเมลหรือ ID ของผู้รับ (จากไฟล์ที่ให้มาคือ PENK <demo@example.com> หรือ ID ktradeh2h02@kasikornbank.com)
	// ในที่นี้แนะนำให้ใช้ Key ID หรือ Email ที่ระบุในไฟล์
	recipient := "ktradeh2h02@kasikornbank.com"

	// 3. นำเข้ากุญแจจากไฟล์เข้าสู่ระบบก่อน (เพื่อให้ GPG รู้จักคีย์นี้)
	importCmd := exec.Command("gpg", "--homedir", gpgDir, "--batch", "--import", keyPath)
	if out, err := importCmd.CombinedOutput(); err != nil {
		http.Error(w, "Failed to import local key: "+string(out), 500)
		return
	}

	// 4. รับไฟล์ที่ต้องการเข้ารหัสจาก User
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", 400)
		return
	}
	defer file.Close()

	// 5. ทำการเข้ารหัส
	cmd := exec.Command("gpg",
		"--homedir", gpgDir,
		"--batch",
		"--encrypt",
		"--recipient", recipient,
		"--trust-model", "always",
		"--armor", // ใช้ Armor เพื่อให้ได้ไฟล์แบบ Text
		"--output", "-",
	)

	cmd.Stdin = file
	out, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, "Encryption Error: "+string(out), 500)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="encrypted_by_penk.asc"`)
	w.Write(out)
}

/*
	 func decryptHandler(w http.ResponseWriter, r *http.Request) {
		file, _, _ := r.FormFile("file")
		defer file.Close()

		cmd := exec.Command("gpg",
			"--homedir", gpgDir,
			"--batch",
			"--pinentry-mode", "loopback",
			"--decrypt",
		)
		cmd.Stdin = file

		out, err := cmd.CombinedOutput()
		if err != nil {
			http.Error(w, "Decrypt Error: "+string(out), 500)
			return
		}

		w.Write(out) // จะได้ข้อความต้นฉบับกลับมา
	}
*/
func decryptHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", 400)
		return
	}
	defer file.Close()

	// ใช้ bytes.Buffer เพื่อแยก Stdout และ Stderr
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("gpg",
		"--homedir", gpgDir,
		"--batch",
		"--pinentry-mode", "loopback",
		"--decrypt",
	)

	cmd.Stdin = file
	cmd.Stdout = &stdout // เก็บข้อมูลที่ถอดรหัสได้ที่นี่
	cmd.Stderr = &stderr // เก็บ Log ของ GPG ไว้ที่นี่

	err = cmd.Run()
	if err != nil {
		http.Error(w, "Decrypt Error: "+stderr.String(), 500)
		return
	}

	// ส่งเฉพาะข้อมูลจริง (Stdout) กลับไปให้ User
	w.Header().Set("Content-Type", "text/plain")
	w.Write(stdout.Bytes())
}

func decryptLocalHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", 400)
		return
	}
	defer file.Close()

	// แยก Stdout เพื่อเอาข้อมูลจริง และ Stderr เพื่อเอาไว้ดู Error Log
	var stdout, stderr bytes.Buffer

	// คำสั่ง gpg สำหรับถอดรหัส
	// หมายเหตุ: หากกุญแจ PENK ของคุณมีรหัสผ่าน (Passphrase) ต้องเพิ่ม --passphrase หรือใช้ --pinentry-mode loopback
	cmd := exec.Command("gpg",
		"--homedir", gpgDir,
		"--batch",
		"--pinentry-mode", "loopback", // ยอมรับรหัสผ่านจาก stdin หรือไม่ต้องใช้หน้าจอโต้ตอบ
		"--decrypt",
	)

	cmd.Stdin = file
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// หากถอดรหัสไม่ได้ (เช่น ไม่มี Private Key ของกุญแจชุดนั้นอยู่ในเครื่อง)
		http.Error(w, "Decrypt Local Error: "+stderr.String(), 500)
		return
	}

	// ส่งไฟล์ที่ถอดรหัสแล้วกลับไป (นามสกุลเดิมก่อนโดน encrypt)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(stdout.Bytes())
}

func main() {
	http.HandleFunc("/keys/generate", generateKeyHandler)
	http.HandleFunc("/encrypt", encryptHandler)
	http.HandleFunc("/encrypt/local", encryptWithLocalKeyHandler)
	http.HandleFunc("/decrypt", decryptHandler)
	http.HandleFunc("/decrypt/local", decryptLocalHandler)
	http.ListenAndServe(":8080", nil)
}
