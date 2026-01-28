package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

var (
	gpgDir      string
	pubKeyFile  string // สำหรับ Encrypt
	privKeyFile string // สำหรับ Decrypt
	passphrase  string // รหัสผ่านของ Private Key
)

func getRecipientFromFile(path string) (string, error) {
	fullPath := "/app/" + path
	out, err := exec.Command("gpg", "--show-keys", "--with-colons", fullPath).Output()
	if err != nil {
		return "", fmt.Errorf("gpg show-keys error: %v (path: %s)", err, fullPath)
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "pub:") {
			parts := strings.Split(line, ":")
			if len(parts) > 4 {
				return parts[4], nil
			}
		}
	}
	return "", fmt.Errorf("recipient not found in key file")
}

func encryptHandler(w http.ResponseWriter, r *http.Request) {
	if pubKeyFile == "" {
		http.Error(w, "GPG_KEY_FILE environment variable not set", 500)
		return
	}

	// 1. ดึง Recipient ID อัตโนมัติ
	recipient, err := getRecipientFromFile(pubKeyFile)
	if err != nil {
		http.Error(w, "Identify key error: "+err.Error(), 500)
		return
	}

	// 2. Import กุญแจ (ใช้ Path เต็ม /app/)
	fullPubKeyPath := "/app/" + pubKeyFile
	importCmd := exec.Command("gpg", "--homedir", gpgDir, "--batch", "--import", fullPubKeyPath)
	importCmd.Run()

	// 3. รับไฟล์จาก Form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", 400)
		return
	}
	defer file.Close()

	// 4. Encrypt
	// เพิ่ม --always-trust เพื่อให้ใช้งานคีย์ที่เพิ่ง import ได้ทันทีโดยไม่ต้องรอการยืนยัน trust db
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("gpg",
		"--homedir", gpgDir,
		"--batch",
		"--always-trust",
		"--encrypt",
		"--recipient", recipient,
		"--output", "-",
	)
	cmd.Stdin = file
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		http.Error(w, "Encryption Error: "+stderr.String(), 500)
		return
	}

	/* w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="encrypted_file.gpg"`)
	w.Write(stdout.Bytes()) */
	downloadName := header.Filename + ".gpg"

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	w.Write(stdout.Bytes())
}

func decryptHandler(w http.ResponseWriter, r *http.Request) {
	if privKeyFile == "" {
		http.Error(w, "GPG_PRIV_KEY_FILE not set", 500)
		return
	}

	// 1. Import Private Key (ใช้ Path เต็ม /app/)
	fullPrivKeyPath := "/app/" + privKeyFile
	importCmd := exec.Command("gpg", "--homedir", gpgDir, "--batch", "--import", fullPrivKeyPath)
	importCmd.Run()

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", 400)
		return
	}
	defer file.Close()

	var stdout, stderr bytes.Buffer

	// 2. ตั้งค่าคำสั่ง Decrypt
	args := []string{
		"--homedir", gpgDir,
		"--batch",
		"--pinentry-mode", "loopback",
		"--decrypt",
	}

	if passphrase != "" {
		args = append(args, "--passphrase", passphrase)
	}

	cmd := exec.Command("gpg", args...)
	cmd.Stdin = file
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		http.Error(w, "Decrypt Error: "+stderr.String(), 500)
		return
	}

	/* w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(stdout.Bytes()) */
	originalName := header.Filename
	downloadName := strings.TrimSuffix(originalName, ".gpg")
	downloadName = "decrypted_" + originalName

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	w.Write(stdout.Bytes())
}

func exportKey(gpgDir, email, keyType string) (string, error) {
	// keyType: "--export" (Public) หรือ "--export-secret-keys" (Private)
	cmd := exec.Command("gpg", "--homedir", gpgDir, "--armor", keyType, email)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func generateKeyHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	email := r.URL.Query().Get("email")

	if name == "" || email == "" {
		http.Error(w, "name and email are required", 400)
		return
	}

	// สร้าง Batch Config สำหรับ RSA 2048 แบบไม่ใส่ Passphrase
	batchConfig := fmt.Sprintf(`
		Key-Type: RSA
		Key-Length: 2048
		Subkey-Type: RSA
		Subkey-Length: 2048
		Name-Real: %s
		Name-Email: %s
		Expire-Date: 0
		%%no-protection
		%%commit
	`, name, email)

	cmd := exec.Command("gpg", "--homedir", gpgDir, "--batch", "--generate-key")
	cmd.Stdin = strings.NewReader(batchConfig)

	if out, err := cmd.CombinedOutput(); err != nil {
		http.Error(w, "Generation Error: "+string(out), 500)
		return
	}

	// Export ทั้ง 2 คีย์ออกมา
	pubKey, err := exportKey(gpgDir, email, "--export")
	if err != nil {
		http.Error(w, "Export Public Key Error", 500)
		return
	}

	privKey, err := exportKey(gpgDir, email, "--export-secret-keys")
	if err != nil {
		http.Error(w, "Export Private Key Error", 500)
		return
	}

	// ส่งกลับเป็น Text/Plain หรือ JSON ในที่นี้ส่งเป็น Text
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "### PUBLIC KEY ###\n%s\n\n### PRIVATE KEY ###\n%s", pubKey, privKey)
}

func main() {
	_ = godotenv.Load()

	gpgDir = os.Getenv("GNUPGHOME")
	if gpgDir == "" {
		gpgDir = "/home/vault/.gnupg"
	}

	pubKeyFile = os.Getenv("GPG_PUB_KEY_FILE")
	privKeyFile = os.Getenv("GPG_PRIV_KEY_FILE")
	passphrase = os.Getenv("GPG_PASSPHRASE")

	http.HandleFunc("/encrypt", encryptHandler)
	http.HandleFunc("/decrypt", decryptHandler)
	http.HandleFunc("/keys/generate", generateKeyHandler)

	log.Printf("Server starting on :8080 with Key: %s", pubKeyFile)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
