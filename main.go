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

func main() {
	http.HandleFunc("/keys/generate", generateKeyHandler)
	http.HandleFunc("/encrypt", encryptHandler)
	http.HandleFunc("/decrypt", decryptHandler)
	http.ListenAndServe(":8080", nil)
}
