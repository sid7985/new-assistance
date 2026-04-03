package computer

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// OpenDocument opens any file with the default macOS application.
func OpenDocument(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}
	fmt.Printf("📄 Opening document: %s\n", path)
	return exec.Command("open", path).Run()
}

// CreateDocument creates a new file with optional content and opens it.
func CreateDocument(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create document: %v", err)
	}
	fmt.Printf("📝 Created document: %s\n", path)
	return OpenDocument(path)
}

// WebSearch opens a Google search in the default browser.
func WebSearch(query string) error {
	searchURL := "https://www.google.com/search?q=" + url.QueryEscape(query)
	fmt.Printf("🔍 Searching the web: %s\n", query)
	return exec.Command("open", searchURL).Run()
}

// YouTubeSearch opens a YouTube search in the default browser.
func YouTubeSearch(query string) error {
	ytURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query)
	fmt.Printf("▶️  Searching YouTube: %s\n", query)
	return exec.Command("open", ytURL).Run()
}

// PlayMusic searches for music. Tries Spotify first, falls back to YouTube.
func PlayMusic(query string) error {
	spotifyURL := "https://open.spotify.com/search/" + url.QueryEscape(query)
	fmt.Printf("🎵 Playing music: %s\n", query)
	return exec.Command("open", spotifyURL).Run()
}

// CloneRepo clones a git repository into the specified destination directory.
func CloneRepo(repoURL, destDir string) error {
	fmt.Printf("📦 Cloning repo %s into %s...\n", repoURL, destDir)
	cmd := exec.Command("git", "clone", repoURL, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %v", err)
	}
	fmt.Printf("✅ Repo cloned successfully into %s\n", destDir)
	return nil
}
// OpenWhatsApp opens the WhatsApp desktop application or web link.
func OpenWhatsApp(phone string, text string) error {
	messageURL := "https://web.whatsapp.com/send?phone=" + phone
	if text != "" {
		messageURL += "&text=" + url.QueryEscape(text)
	}
	fmt.Printf("💬 Messaging on WhatsApp: %s\n", phone)
	return exec.Command("open", messageURL).Run()
}

// OpenPaint opens a simple drawing canvas (macOS Freeform).
func OpenPaint() error {
	fmt.Printf("🎨 Opening Paint (Freeform)...\n")
	return exec.Command("open", "-a", "Freeform").Run()
}

// ExecuteRemoteCommand runs a shell command on a client server via SSH.
func ExecuteRemoteCommand(host, user, command string) (string, error) {
	fmt.Printf("🌐 Accessing client server: %s@%s | Command: %s\n", user, host, command)
	cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%s@%s", user, host), command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("SSH failure: %v. Output: %s", err, string(out))
	}
	fmt.Printf("✅ Remote command successful on %s\n", host)
	return string(out), nil
}

// CreateAgentWorktree creates an isolated git worktree for a specific agent persona,
// ensuring their code modifications do not interfere with the main trunk unless explicitly merged.
func CreateAgentWorktree(projectDir, agentName string) (string, error) {
	worktreePath := filepath.Join(projectDir, ".agents", agentName)
	
	// Create .agents dir if it doesn't exist
	if err := os.MkdirAll(filepath.Join(projectDir, ".agents"), 0755); err != nil {
		return "", err
	}

	// Verify the project is a git repo
	if err := exec.Command("git", "-C", projectDir, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		// If not a git repo, just return a regular directory for isolation
		err = os.MkdirAll(worktreePath, 0755)
		return worktreePath, err
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		return worktreePath, nil // Already isolated
	}

	branchName := fmt.Sprintf("agent/%s/%d", agentName, time.Now().Unix())
	fmt.Printf("🌿 Expanding Agent Worktree for %s at %s\n", agentName, worktreePath)
	
	cmd := exec.Command("git", "-C", projectDir, "worktree", "add", "-b", branchName, worktreePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree failed: %v, out: %s", err, out)
	}

	return worktreePath, nil
}
