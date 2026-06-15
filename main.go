package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
)

const gitmojiURL = "https://gitmoji.dev/api/gitmojis"

type gitmoji struct {
	Emoji       string `json:"emoji"`
	Entity      string `json:"entity"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Name        string `json:"name"`
	Semver      string `json:"semver"`
}
type gitmojiApiResponse struct {
	Gitmojis []gitmoji `json:"gitmojis"`
}

type Format string

var (
	formatList  Format = "list"
	formatEmoji Format = "emoji"
	formatCode  Format = "code"
	formatJSON  Format = "json"
)

// CLI flags
var (
	search string
	format Format
	single bool
	amend  bool
)

func main() {
	flag.StringVar(&search, "search", "", "Search for a gitmoji by name, description or code")
	flag.StringVar((*string)(&format), "format", "list", "Output format: list, emoji, code or json")
	flag.BoolVar(&single, "single", false, "Show only the first result")
	flag.BoolVar(&amend, "git-amend", false, "Amend the last commit with the selected gitmoji as prefix")
	flag.Parse()

	if format != formatList && format != formatEmoji && format != formatCode && format != formatJSON {
		fmt.Println("Invalid format. Valid formats are: list, emoji, code or json")
		return
	}

	gitmojis, err := readGitmojis()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if search != "" {
		gitmojis = applySearch(gitmojis, search)
	}

	if single && len(gitmojis) > 0 {
		gitmojis = gitmojis[:1]
	}

	if amend {
		prefix := gitmojis[0].Emoji
		if format == formatCode {
			prefix = gitmojis[0].Code
		}

		var out strings.Builder
		cmd := exec.Command("git", "log", "-1", "--pretty=%B")
		cmd.Stdout = &out
		err := cmd.Run()

		message := out.String()

		cmd = exec.Command("git", "commit", "--amend", "--message", fmt.Sprintf("%s %s", prefix, message))
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			fmt.Println("Error:", err)
		}
		return
	}
	printGitmojis(gitmojis, format)
}

func readGitmojis() ([]gitmoji, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("Config dir not found: %w", err)
	}
	gitmojiGoDir := path.Join(configDir, "gitmoji-go")
	gitmojisFile := path.Join(gitmojiGoDir, "gitmojis.json")

	bytes, err := os.ReadFile(gitmojisFile)
	if err != nil {
		fmt.Println("Downloading gitmojis")
		resp, err := http.Get(gitmojiURL)
		if err != nil {
			return nil, fmt.Errorf("Impossible to download gitmojis from API: %w", err)
		}
		if resp.Body != nil {
			defer resp.Body.Close()
		}
		bytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Impossible to read gitmojis from API: %w", err)
		}
		err = os.MkdirAll(gitmojiGoDir, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("Impossible to create gitmoji directory: %w", err)
		}
		err = os.WriteFile(gitmojisFile, bytes, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("Impossible to write gitmojis to file: %w", err)
		}
	}
	gitmojiApiResponse := gitmojiApiResponse{}
	err = json.Unmarshal(bytes, &gitmojiApiResponse)
	if err != nil {
		return nil, fmt.Errorf("Impossible to parse gitmojis from API: %w", err)
	}
	return gitmojiApiResponse.Gitmojis, nil
}

func applySearch(gitmojis []gitmoji, search string) []gitmoji {
	var filteredGitmojis []gitmoji
	for _, g := range gitmojis {
		if strings.Contains(g.Name, search) || strings.Contains(g.Description, search) || strings.Contains(g.Code, search) {
			filteredGitmojis = append(filteredGitmojis, g)
		}
	}
	return filteredGitmojis
}

func printGitmojis(gitmojis []gitmoji, format Format) {
	if format == formatJSON {
		bytes, err := json.MarshalIndent(gitmojis, "", "  ")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println(string(bytes))
		return
	}

	if format == formatList {
		for _, g := range gitmojis {
			fmt.Printf("%s %s - %s\n", g.Emoji, g.Code, g.Description)
		}
	}
	if format == formatEmoji {
		for _, g := range gitmojis {
			fmt.Printf("%s\n", g.Emoji)
		}
	}
	if format == formatCode {
		for _, g := range gitmojis {
			fmt.Printf("%s\n", g.Code)
		}
	}
}
