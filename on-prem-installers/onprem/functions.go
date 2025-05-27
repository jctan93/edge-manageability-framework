// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

//go:build mage

package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"time"

	"bufio"

	"path/filepath"
	"regexp"

	"github.com/magefile/mage/mg"
)

var orchNamespaceList = []string{
	"onprem",
	"orch-boots",
	"orch-database",
	"orch-platform",
	"orch-app",
	"orch-cluster",
	"orch-infra",
	"orch-sre",
	"orch-ui",
	"orch-secret",
	"orch-gateway",
	"orch-harbor",
	"cattle-system",
}

type OnPrem mg.Namespace

// Create a harbor admin credential secret
func (OnPrem) CreateHarborSecret(namespace, password string) error {
	exec.Command("kubectl", "-n", namespace, "delete", "secret", "harbor-admin-credential", "--ignore-not-found").Run()
	secret := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: harbor-admin-credential
  namespace: %s
stringData:
  credential: "admin:%s"
`, namespace, password)
	return applySecret(secret)
}

// Create a harbor admin password secret
func (OnPrem) CreateHarborPassword(namespace, password string) error {
	exec.Command("kubectl", "-n", namespace, "delete", "secret", "harbor-admin-password", "--ignore-not-found").Run()
	secret := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: harbor-admin-password
  namespace: %s
stringData:
  HARBOR_ADMIN_PASSWORD: "%s"
`, namespace, password)
	return applySecret(secret)
}

// Create a keycloak admin password secret
func (OnPrem) CreateKeycloakPassword(namespace, password string) error {
	exec.Command("kubectl", "-n", namespace, "delete", "secret", "platform-keycloak", "--ignore-not-found").Run()
	secret := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: platform-keycloak
  namespace: %s
stringData:
  admin-password: "%s"
`, namespace, password)
	return applySecret(secret)
}

// Create a postgres password secret
func (OnPrem) CreatePostgresPassword(namespace, password string) error {
	exec.Command("kubectl", "-n", namespace, "delete", "secret", "postgresql", "--ignore-not-found").Run()
	secret := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: postgresql
  namespace: %s
stringData:
  postgres-password: "%s"
`, namespace, password)
	return applySecret(secret)
}

// Generate a random password with requirements
func (OnPrem) GeneratePassword() (string, error) {
	lower := randomChars("abcdefghijklmnopqrstuvwxyz", 1)
	upper := randomChars("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1)
	digit := randomChars("0123456789", 1)
	special := randomChars("!@#$%^&*()_+{}|:<>?", 1)
	remaining := randomChars("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+{}|:<>?", 21)
	password := lower + upper + digit + special + remaining
	shuffled := shuffleString(password)
	fmt.Println(shuffled)
	return shuffled, nil
}

// Check if oras is installed
func (OnPrem) CheckOras() error {
	_, err := exec.LookPath("oras")
	if err != nil {
		return fmt.Errorf("Oras is not installed, install oras, exiting...")
	}
	return nil
}

// Install jq tool
func (OnPrem) InstallJq() error {
	_, err := exec.LookPath("jq")
	if err == nil {
		fmt.Println("jq tool found in the path")
		return nil
	}
	cmd := exec.Command("sudo", "NEEDRESTART_MODE=a", "apt-get", "install", "-y", "jq")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Install yq tool
func (OnPrem) InstallYq() error {
	_, err := exec.LookPath("yq")
	if err == nil {
		fmt.Println("yq tool found in the path")
		return nil
	}
	cmd := exec.Command("bash", "-c", "curl -jL https://github.com/mikefarah/yq/releases/download/v4.42.1/yq_linux_amd64 -o /tmp/yq && sudo mv /tmp/yq /usr/bin/yq && sudo chmod 755 /usr/bin/yq")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Download artifacts from OCI registry in Release Service
func (OnPrem) DownloadArtifacts(cwd, dirName, rsURL, rsPath string, artifacts ...string) error {
	os.MkdirAll(fmt.Sprintf("%s/%s", cwd, dirName), 0755)
	os.Chdir(fmt.Sprintf("%s/%s", cwd, dirName))
	for _, artifact := range artifacts {
		cmd := exec.Command("sudo", "oras", "pull", "-v", fmt.Sprintf("%s/%s/%s", rsURL, rsPath, artifact))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return os.Chdir(cwd)
}

// Get JWT token from Azure
func (OnPrem) GetJWTToken(refreshToken, rsURL string) (string, error) {
	cmd := exec.Command("curl", "-X", "POST", "-d", fmt.Sprintf("refresh_token=%s&grant_type=refresh_token", refreshToken), fmt.Sprintf("https://%s/oauth/token", rsURL))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	jq := exec.Command("jq", "-r", ".id_token")
	jq.Stdin = bytes.NewReader(out)
	token, err := jq.Output()
	return strings.TrimSpace(string(token)), err
}

// Wait for pods in namespace to be in Ready state
func (OnPrem) WaitForPodsRunning(namespace string) error {
	cmd := exec.Command("kubectl", "wait", "pod", "--selector=!job-name", "--all", "--for=condition=Ready", "--namespace="+namespace, "--timeout=600s")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Wait for deployment to be in Ready state
func (OnPrem) WaitForDeploy(deployment, namespace string) error {
	cmd := exec.Command("kubectl", "rollout", "status", "deploy/"+deployment, "-n", namespace, "--timeout=30m")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Wait for namespace to be created
func (OnPrem) WaitForNamespaceCreation(namespace string) error {
	for {
		cmd := exec.Command("kubectl", "get", "ns", namespace, "-o", "json")
		out, err := cmd.Output()
		if err != nil {
			return err
		}
		jq := exec.Command("jq", ".status.phase", "-r")
		jq.Stdin = bytes.NewReader(out)
		phase, err := jq.Output()
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(phase)) == "Active" {
			break
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

// --- Helper functions ---

func applySecret(secret string) error {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(secret)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func randomChars(charset string, length int) string {
	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}
	return string(result)
}

func shuffleString(input string) string {
	r := []rune(input)
	for i := len(r) - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		r[i], r[j.Int64()] = r[j.Int64()], r[i]
	}
	return string(r)
}

func (OnPrem) CreateNamespaces() error {
	for _, ns := range orchNamespaceList {
		cmd := exec.Command("kubectl", "create", "ns", ns, "--dry-run=client", "-o", "yaml")
		apply := exec.Command("kubectl", "apply", "-f", "-")
		pipe, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to get stdout pipe: %w", err)
		}
		apply.Stdin = pipe
		apply.Stdout = nil
		apply.Stderr = nil
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start kubectl create: %w", err)
		}
		if err := apply.Run(); err != nil {
			return fmt.Errorf("failed to apply namespace: %w", err)
		}
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("kubectl create wait failed: %w", err)
		}
	}
	return nil
}

func (OnPrem) CreateSreSecrets() error {
	namespace := "orch-sre"
	sreUsername := os.Getenv("SRE_USERNAME")
	srePassword := os.Getenv("SRE_PASSWORD")
	sreDestURL := os.Getenv("SRE_DEST_URL")
	sreDestCACert := os.Getenv("SRE_DEST_CA_CERT")

	// Delete existing secrets
	secrets := []string{
		"basic-auth-username",
		"basic-auth-password",
		"destination-secret-url",
		"destination-secret-ca",
	}
	for _, secret := range secrets {
		exec.Command("kubectl", "-n", namespace, "delete", "secret", secret, "--ignore-not-found").Run()
	}

	// Create basic-auth-username secret
	secret1 := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: basic-auth-username
  namespace: %s
stringData:
  username: %s
`, namespace, sreUsername)
	if err := applySecret(secret1); err != nil {
		return err
	}

	// Create basic-auth-password secret
	secret2 := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: basic-auth-password
  namespace: %s
stringData:
  password: "%s"
`, namespace, srePassword)
	if err := applySecret(secret2); err != nil {
		return err
	}

	// Create destination-secret-url secret
	secret3 := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: destination-secret-url
  namespace: %s
stringData:
  url: %s
`, namespace, sreDestURL)
	if err := applySecret(secret3); err != nil {
		return err
	}

	// Create destination-secret-ca secret if SRE_DEST_CA_CERT is set
	if sreDestCACert != "" {
		// Indent each line of the CA cert by 4 spaces
		indented := "    " + strings.ReplaceAll(sreDestCACert, "\n", "\n    ")
		secret4 := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: destination-secret-ca
  namespace: %s
stringData:
  ca.crt: |
%s
`, namespace, indented)
		if err := applySecret(secret4); err != nil {
			return err
		}
	}

	return nil
}

// // Helper function to apply a secret using kubectl
// func applySecret(secret string) error {
// 	cmd := exec.Command("kubectl", "apply", "-f", "-")
// 	cmd.Stdin = strings.NewReader(secret)
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	return cmd.Run()
// }

func (OnPrem) PrintEnvVariables() {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("         Environment Variables")
	fmt.Println("========================================")
	fmt.Printf("%-25s: %s\n", "RELEASE_SERVICE_URL", os.Getenv("RELEASE_SERVICE_URL"))
	fmt.Printf("%-25s: %s\n", "ORCH_INSTALLER_PROFILE", os.Getenv("ORCH_INSTALLER_PROFILE"))
	fmt.Printf("%-25s: %s\n", "DEPLOY_VERSION", os.Getenv("DEPLOY_VERSION"))
	fmt.Println("========================================")
	fmt.Println()
}

func (OnPrem) AllowConfigInRuntime() error {
	enableTrace := os.Getenv("ENABLE_TRACE") == "true"
	cwd, _ := os.Getwd()
	gitArchName := os.Getenv("git_arch_name")
	siConfigRepo := os.Getenv("si_config_repo")
	assumeYes := os.Getenv("ASSUME_YES") == "true"

	tmpDir := filepath.Join(cwd, gitArchName, "tmp")
	configRepoPath := filepath.Join(tmpDir, siConfigRepo)

	// Disable tracing if enabled (not implemented in Go, just print)
	if enableTrace {
		fmt.Println("Tracing is enabled. Temporarily disabling tracing")
	}

	// Check if config already exists
	if _, err := os.Stat(configRepoPath); err == nil {
		fmt.Printf("Configuration already exists at %s.\n", configRepoPath)
		if assumeYes {
			fmt.Println("Assuming yes to use existing configuration.")
			return nil
		}
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("Do you want to overwrite the existing configuration? (yes/no): ")
			yn, _ := reader.ReadString('\n')
			yn = strings.TrimSpace(yn)
			switch strings.ToLower(yn) {
			case "y", "yes":
				os.RemoveAll(configRepoPath)
				break
			case "n", "no":
				fmt.Println("Using existing configuration.")
				return nil
			default:
				fmt.Println("Please answer yes or no.")
			}
		}
	}

	// Untar edge-manageability-framework repo
	repoFile := ""
	files, _ := filepath.Glob(filepath.Join(cwd, gitArchName, fmt.Sprintf("*%s*.tgz", siConfigRepo)))
	if len(files) > 0 {
		repoFile = filepath.Base(files[0])
	}
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	cmd := exec.Command("tar", "-xf", filepath.Join(cwd, gitArchName, repoFile), "-C", tmpDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Prompt for Docker.io credentials
	reader := bufio.NewReader(os.Stdin)
	dockerUsername := os.Getenv("DOCKER_USERNAME")
	dockerPassword := os.Getenv("DOCKER_PASSWORD")
	for {
		if dockerUsername == "" && dockerPassword == "" {
			fmt.Print("Would you like to provide Docker credentials? (Y/n): ")
			yn, _ := reader.ReadString('\n')
			yn = strings.TrimSpace(yn)
			if strings.ToLower(yn) == "y" || yn == "" {
				fmt.Print("Enter Docker Username: ")
				dockerUsername, _ = reader.ReadString('\n')
				dockerUsername = strings.TrimSpace(dockerUsername)
				os.Setenv("DOCKER_USERNAME", dockerUsername)
				fmt.Print("Enter Docker Password: ")
				dockerPassword, _ = reader.ReadString('\n')
				dockerPassword = strings.TrimSpace(dockerPassword)
				os.Setenv("DOCKER_PASSWORD", dockerPassword)
				break
			} else if strings.ToLower(yn) == "n" {
				fmt.Println("The installation will proceed without using Docker credentials.")
				os.Unsetenv("DOCKER_USERNAME")
				os.Unsetenv("DOCKER_PASSWORD")
				break
			} else {
				fmt.Println("Please answer yes or no.")
			}
		} else {
			fmt.Println("Setting Docker credentials.")
			os.Setenv("DOCKER_USERNAME", dockerUsername)
			os.Setenv("DOCKER_PASSWORD", dockerPassword)
			break
		}
	}

	if dockerUsername != "" && dockerPassword != "" {
		fmt.Println("Docker credentials are set.")
	} else {
		fmt.Println("Docker credentials are not valid. The installation will proceed without using Docker credentials.")
		os.Unsetenv("DOCKER_USERNAME")
		os.Unsetenv("DOCKER_PASSWORD")
	}

	// Prompt for IP addresses for Argo, Traefik and Nginx services
	fmt.Println("Provide IP addresses for Argo, Traefik and Nginx services.")
	ipRegex := regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}$`)
	var argoIP, traefikIP, nginxIP string
	for {
		if argoIP == "" {
			fmt.Print("Enter Argo IP: ")
			argoIP, _ = reader.ReadString('\n')
			argoIP = strings.TrimSpace(argoIP)
			os.Setenv("ARGO_IP", argoIP)
		}
		if traefikIP == "" {
			fmt.Print("Enter Traefik IP: ")
			traefikIP, _ = reader.ReadString('\n')
			traefikIP = strings.TrimSpace(traefikIP)
			os.Setenv("TRAEFIK_IP", traefikIP)
		}
		if nginxIP == "" {
			fmt.Print("Enter Nginx IP: ")
			nginxIP, _ = reader.ReadString('\n')
			nginxIP = strings.TrimSpace(nginxIP)
			os.Setenv("NGINX_IP", nginxIP)
		}
		if ipRegex.MatchString(argoIP) && ipRegex.MatchString(traefikIP) && ipRegex.MatchString(nginxIP) {
			fmt.Println("IP addresses are valid.")
			break
		} else {
			fmt.Println("Inputted values are not valid IPs. Please input correct IPs without any masks.")
			argoIP, traefikIP, nginxIP = "", "", ""
			os.Unsetenv("ARGO_IP")
			os.Unsetenv("TRAEFIK_IP")
			os.Unsetenv("NGINX_IP")
		}
	}

	// Wait for SI to confirm that they have made changes
	for {
		proceed := os.Getenv("PROCEED")
		if proceed != "" {
			break
		}
		fmt.Printf(`Edit config values.yaml files with custom configurations if necessary!!!
The files are located at:
%s/%s/orch-configs/profiles/<profile>.yaml
%s/%s/orch-configs/clusters/$ORCH_INSTALLER_PROFILE.yaml
Enter 'yes' to confirm that configuration is done in order to progress with installation
('no' will exit the script) !!!

Ready to proceed with installation? `, tmpDir, siConfigRepo, tmpDir, siConfigRepo)
		yn, _ := reader.ReadString('\n')
		yn = strings.TrimSpace(yn)
		switch strings.ToLower(yn) {
		case "y", "yes":
			break
		case "n", "no":
			os.Exit(1)
		default:
			fmt.Println("Please answer yes or no.")
			continue
		}
		break
	}

	// Re-enable tracing if needed
	if enableTrace {
		fmt.Println("Tracing is enabled. Re-enabling tracing")
	}

	return nil
}
