package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"nanoclaw-orchestrator/config"
	"nanoclaw-orchestrator/internal/computer"
	"nanoclaw-orchestrator/internal/minimax"
	"nanoclaw-orchestrator/internal/orchestrator"
	"nanoclaw-orchestrator/internal/steps"
	"nanoclaw-orchestrator/internal/telegram"
)

var memory = struct {
	Prompts    []string
	Iterations []iteration
	Project    string
}{}

type iteration struct {
	Index     int
	Prompt    string
	Response  string
	Timestamp time.Time
}

func main() {
	project := flag.String("project", "", "Project name")
	repoURL := flag.String("repo", "", "Git repo URL to clone into project folder")
	newFlag := flag.Bool("new", false, "Start fresh (clears memory)")
	calibrate := flag.Bool("calibrate", false, "Run calibration mode")
	flag.Parse()

	printBanner()

	if *calibrate {
		computer.PrintMousePosition()
		return
	}

	cfg := config.Load()

	if *newFlag {
		memory.Prompts = []string{}
		memory.Iterations = []iteration{}
		memory.Project = ""
		fmt.Println("Memory cleared")
	}

	minimaxClient := minimax.NewClient(cfg.MiniMaxAPIKey, cfg.MiniMaxGroupID)

	openCodeRunner := &steps.OpenCodeRunner{
		WorkDir: cfg.ProjectDir,
	}

	orch := &orchestrator.Orchestrator{
		Steps:     []orchestrator.WorkflowStep{},
		Memory:    make(map[string]string),
	}

	if *project != "" {
		orch.NewProject(*project)
		memory.Project = *project
	}

	// Clone repo if -repo flag is provided
	if *repoURL != "" {
		fmt.Printf("Cloning repo: %s\n", *repoURL)
		if err := computer.CloneRepo(*repoURL, cfg.ProjectDir); err != nil {
			fmt.Printf("Warning: repo clone failed: %v\n", err)
		}
	}

	// Start Telegram bot listener in background (if configured)
	var tgBot *telegram.Bot
	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		tgBot = telegram.NewBot(cfg.TelegramBotToken, cfg.TelegramChatID, cfg.ProjectDir, func(req string) (string, error) {
			// Proxy telegram text requests to NanoClaw engine
			return executeActionFromPrompt(req, minimaxClient, openCodeRunner, cfg.ProjectDir)
		})
		tgStopChan := make(chan struct{})
		go tgBot.PollForRepoURLs(tgStopChan)
		defer close(tgStopChan)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, saving state...")
		os.Exit(0)
	}()

	// Start Interactive Loop
	fmt.Println("\n🤖 NanoClaw Interactive Mode")
	fmt.Println("Paste your prompt from Perplexity/Comet. Type 'END' on a new line to submit, or 'exit' to quit.")

	scanner := bufio.NewScanner(os.Stdin)
	i := 0
	for {
		i++
		fmt.Printf("\n=== Prompt %d ===\n> ", i)
		
		var promptBuilder strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "exit" {
				fmt.Println("Exiting NanoClaw...")
				os.Exit(0)
			}
			if strings.TrimSpace(line) == "END" || (promptBuilder.Len() == 0 && line != "") {
				// If they typed END or they just typed a one-liner without END, we proceed.
				// For real multi-line, they paste block and type END.
				if line != "END" {
					promptBuilder.WriteString(line + "\n")
				}
				if promptBuilder.Len() > 0 {
					break
				}
			}
			// Only require END if they pasted multiple lines
			promptBuilder.WriteString(line + "\n")
		}

		prompt := strings.TrimSpace(promptBuilder.String())
		if prompt == "" {
			continue
		}

		screenPath := "/tmp/nanoclaw_screen.png"
		err := computer.TakeScreenshot(screenPath)
		if err != nil {
			fmt.Printf("Warning: Failed to capture screen: %v\n", err)
		} else {
			fmt.Println("Analyzing screen with MiniMax...")
			analysisPrompt := fmt.Sprintf("The next step in the workflow is: '%s'\nAnalyze the screen. Do you see any issues, or is it safe to proceed with the OS/code workflow?", prompt)
			assessment, mmErr := minimaxClient.AnalyzeScreen(screenPath, analysisPrompt)
			if mmErr != nil {
				fmt.Printf("MiniMax analysis error: %v\n", mmErr)
			} else {
				fmt.Printf("\n🧠 MiniMax Assessment:\n%s\n\n", assessment)
			}
		}

		if !orch.AskForConfirmation("Execute autonomous MiniMax action for prompt: " + prompt[:min(50, len(prompt))]+"...") {
			fmt.Println("Action cancelled by user. Skipping prompt.")
			continue
		}

		_, err = orch.RunStep(orchestrator.WorkflowStep{
			Name: fmt.Sprintf("prompt_%d", i),
			Action: func() (string, error) {
				return executeActionFromPrompt(prompt, minimaxClient, openCodeRunner, cfg.ProjectDir)
			},
		})
		if err != nil {
			fmt.Printf("Error running prompt: %v\n", err)
		}
	}

	fmt.Printf("\n✅ Project Complete: %s\n", *project)
}

func printBanner() {
	fmt.Println(`
   _                _   _ _         _ 
  | |    ___   __ _| |_| |_ _ __ | |
  | |   / _ \ / _' | __|  _| '_ \| |
  | |__|  __/ | (_| | |_| | | | | | |
  |_____\___| \__,_|\__|_| |_| |_|_| |
      NanoClaw Orchestrator v1.0      
`)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// executeActionFromPrompt uses the Manager Agent to decompose the prompt into a mission plan, then executes the tasks.
func executeActionFromPrompt(prompt string, minimaxClient *minimax.Client, openCodeRunner *steps.OpenCodeRunner, projectDir string) (string, error) {
	fmt.Println("🟣 Manager Agent is creating a mission plan...")
	plan, err := minimaxClient.GetManagerPlan(prompt)
	if err != nil {
		return "", fmt.Errorf("manager plan error: %v", err)
	}

	tasks := orchestrator.ParseDelegations(plan)
	if len(tasks) == 0 {
		fmt.Println("⚠️  No delegations found in plan. Falling back to direct action.")
		return executeSingleAction(prompt, minimaxClient, openCodeRunner, projectDir)
	}

	orch := &orchestrator.Orchestrator{}
	handler := func(taskPrompt string, role string) (string, error) {
		return executeSingleAction(taskPrompt, minimaxClient, openCodeRunner, projectDir)
	}

	return orch.ExecuteMission(tasks, handler)
}

// executeSingleAction performs a standalone atomic action.
func executeSingleAction(prompt string, minimaxClient *minimax.Client, openCodeRunner *steps.OpenCodeRunner, projectDir string) (string, error) {
	actionData, err := minimaxClient.GetDesktopAction(prompt)
	if err != nil {
		return "", fmt.Errorf("MiniMax action error: %v", err)
	}

	actionName := actionData["action"]
	fmt.Printf("▶️  Task mapping: %s\n", actionName)

	var cmdErr error
	switch actionName {
	case "OpenDocument":
		cmdErr = computer.OpenDocument(actionData["path"])
	case "CreateDocument":
		cmdErr = computer.CreateDocument(actionData["path"], actionData["content"])
	case "WebSearch":
		cmdErr = computer.WebSearch(actionData["query"])
	case "YouTubeSearch":
		cmdErr = computer.YouTubeSearch(actionData["query"])
	case "PlayMusic":
		cmdErr = computer.PlayMusic(actionData["query"])
	case "TerminalCommand":
		fmt.Printf("⚙️  Running command: %s\n", actionData["command"])
		cmd := exec.Command("sh", "-c", actionData["command"])
		cmd.Dir = projectDir
		out, errCmd := cmd.CombinedOutput()
		if errCmd != nil {
			return string(out), errCmd
		}
		return fmt.Sprintf("Command executed: %s\nOutput:\n%s", actionData["command"], string(out)), nil
	case "OpenCode":
		fmt.Printf("💻 Running OpenCode: %s\n", actionData["prompt"])
		out, errCmd := openCodeRunner.RunPrompt(actionData["prompt"])
		if errCmd != nil {
			return out, errCmd
		}
		return fmt.Sprintf("OpenCode completed:\n%s", out), nil
	case "AgentDesktopControl":
		fmt.Printf("🖥️  Agent taking control of desktop: %s\n", actionData["prompt"])
		out, errCmd := executeAutonomousLoop(actionData["prompt"], minimaxClient)
		if errCmd != nil {
			return out, errCmd
		}
		return out, nil
	case "ChatResponse":
		return actionData["reply"], nil
	default:
		return "", fmt.Errorf("unknown action from AI: %s", actionName)
	}

	if cmdErr != nil {
		return "", cmdErr
	}
	return fmt.Sprintf("Successfully executed %s", actionName), nil
}

// executeAutonomousLoop takes screenshots and executes atomic actions continuously until TaskComplete
func executeAutonomousLoop(prompt string, minimaxClient *minimax.Client) (string, error) {
	fmt.Printf("🤖 Starting Autonomous Desktop Control for: %s\n", prompt)
	for i := 0; i < 20; i++ {
		screenPath := fmt.Sprintf("/tmp/nanoclaw_auto_screen_%d.png", i)
		err := computer.TakeScreenshot(screenPath)
		if err != nil {
			return "", fmt.Errorf("failed to capture screen: %v", err)
		}

		action, err := minimaxClient.GetVisionDesktopAction(prompt, screenPath)
		if err != nil {
			return "", fmt.Errorf("vision action error: %v", err)
		}

		actionName, _ := action["action"].(string)
		fmt.Printf("   -> Next Action: %s\n", actionName)

		switch actionName {
		case "MouseClick":
			x, _ := action["x"].(float64)
			y, _ := action["y"].(float64)
			computer.ClickAt(int(x), int(y))
		case "MouseDoubleClick":
			x, _ := action["x"].(float64)
			y, _ := action["y"].(float64)
			computer.DoubleClickAt(int(x), int(y))
		case "KeyboardType":
			text, _ := action["text"].(string)
			computer.TypeText(text)
		case "KeyboardPress":
			key, _ := action["key"].(string)
			computer.KeyboardPress(key)
		case "MouseScroll":
			x, _ := action["x"].(float64)
			y, _ := action["y"].(float64)
			direction, _ := action["direction"].(string)
			computer.MouseScroll(int(x), int(y), direction)
		case "TaskComplete":
			summary, _ := action["summary"].(string)
			fmt.Printf("✅ Autonomous Task Complete: %s\n", summary)
			return summary, nil
		default:
			fmt.Printf("   ⚠️ Unknown action %s, stopping loop.\n", actionName)
			return "", fmt.Errorf("unknown primitive action: %s", actionName)
		}

		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("max iterations reached for autonomous control")
}
