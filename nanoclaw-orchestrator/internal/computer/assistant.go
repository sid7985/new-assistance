package computer

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
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
