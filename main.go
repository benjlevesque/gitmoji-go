package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/manifoldco/promptui"
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
	search  string
	format  Format
	single  bool
	amend   bool
	commit  bool
	message string
	version bool
)

// Build flags
var (
	buildVersion string = "dev"
	buildSha     string = "none"
)

func main() {
	flag.BoolVar(&version, "version", false, "Display version information")
	flag.StringVar(&search, "search", "", "Search for a gitmoji by name, description or code")
	flag.StringVar((*string)(&format), "format", "list", "Output format: list, emoji, code or json")
	flag.BoolVar(&single, "single", false, "Show only the first result")
	flag.BoolVar(&amend, "git-amend", false, "Amend the last commit with the selected gitmoji as prefix")
	flag.BoolVar(&commit, "git-commit", false, "Create a new commit with the selected gitmoji as prefix")
	flag.StringVar(&message, "message", "", "Commit message for the new commit")
	flag.Parse()

	if version {
		fmt.Printf("gitmoji-go version: %s, commit: %s\n", buildVersion, buildSha)
		return
	}

	if commit && amend {
		fmt.Println("You cannot use both --git-commit and --git-amend at the same time")
		return
	}

	if amend && message != "" {
		fmt.Println("You cannot use --message when using --git-amend")
		return
	}

	if format != formatList && format != formatEmoji && format != formatCode && format != formatJSON {
		fmt.Println("Invalid format. Valid formats are: list, emoji, code or json")
		return
	}

	gitmojis, err := readGitmojis()
	if err != nil {
		log.Fatal(err)
	}

	if search == "" {
		g, err := promptForGitmoji(&gitmojis)
		if err != nil {
			log.Fatal(err)
		}
		gitmojis = []gitmoji{g}
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

		message, err := runGitCommand("log", "-1", "--pretty=%B")
		if err != nil {
			log.Fatal(err)
		}

		_, err = runGitCommand("commit", "--amend", "--message", fmt.Sprintf("%s %s", prefix, message))
		if err != nil {
			log.Fatal(err)
		}
	} else if commit {
		prefix := gitmojis[0].Emoji
		if format == formatCode {
			prefix = gitmojis[0].Code
		}

		if message == "" {
			message, err = promptForCommitMessage()
			if err != nil {
				log.Fatal(err)
			}
		}

		_, err = runGitCommand("commit", "--message", fmt.Sprintf("%s %s", prefix, message))
		if err != nil {
			log.Fatalf("Error in git commit: %v", err)
		}
		return
	} else {
		printGitmojis(gitmojis, format)
	}
}

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error running git command:\n%v", out.String())
	}

	return out.String(), nil
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

func isMatch(g gitmoji, search string) bool {
	fields := []string{g.Name, g.Description, g.Code}
	for _, field := range fields {
		if fuzzy.MatchNormalizedFold(strings.ToLower(search), strings.ToLower(field)) {
			return true
		}
	}
	return false
}

func applySearch(gitmojis []gitmoji, search string) []gitmoji {
	var filteredGitmojis []gitmoji
	for _, g := range gitmojis {
		if isMatch(g, search) {
			filteredGitmojis = append(filteredGitmojis, g)
		}
	}
	return filteredGitmojis
}

func printGitmojis(gitmojis []gitmoji, format Format) {
	if format == formatJSON {
		bytes, err := json.MarshalIndent(gitmojis, "", "  ")
		if err != nil {
			log.Fatal(err)
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

func removeGitmojiPrefix(message string) string {
	parts := strings.SplitN(message, " ", 2)
	if len(parts) < 2 {
		return message
	}
	if regexp.MustCompile(`\w`).MatchString(parts[0]) {
		return message
	}
	return strings.TrimSpace(parts[1])
}

func promptForGitmoji(gitmojis *[]gitmoji) (gitmoji, error) {
	templates := &promptui.SelectTemplates{
		Label:    "  {{ .Emoji | cyan }} {{ .Code | red }} {{ .Description | green }}",
		Active:   "▸ {{ .Emoji | cyan }} {{ .Code | red }} {{ .Description | green }}",
		Inactive: "  {{ .Emoji | cyan }} {{ .Code | red }} {{ .Description | green }}",
		Selected: "✔ {{ .Emoji | cyan }} {{ .Code | red }} {{ .Description | green }}",
	}
	prompt := promptui.Select{
		Label:             "",
		Items:             *gitmojis,
		Templates:         templates,
		HideSelected:      true,
		StartInSearchMode: true,
		Searcher: func(input string, index int) bool {
			return isMatch((*gitmojis)[index], input)
		},
	}
	i, _, err := prompt.Run()

	if err != nil {
		return gitmoji{}, err
	}

	return (*gitmojis)[i], nil
}
func promptForCommitMessage() (string, error) {
	prompt := promptui.Prompt{
		Label:       "Commit message",
		HideEntered: true,
	}
	message, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return message, nil
}
