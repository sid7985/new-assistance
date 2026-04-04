package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"nanoclaw-orchestrator/config"
	"nanoclaw-orchestrator/internal"
	"nanoclaw-orchestrator/internal/api"
	"nanoclaw-orchestrator/internal/computer"
	"nanoclaw-orchestrator/internal/minimax"
	"nanoclaw-orchestrator/internal/orchestrator"
	"nanoclaw-orchestrator/internal/steps"
	"nanoclaw-orchestrator/internal/telegram"
	"nanoclaw-orchestrator/internal/venice"
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
	useVenice := flag.Bool("venice", false, "Use Venice/Mithril instead of MiniMax")
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

	db, err := internal.NewDatabase(filepath.Join(cfg.ProjectDir, "nanoclaw.db"))
	if err != nil {
		fmt.Printf("⚠️  Database initialization failed: %v. Running in memory-only mode.\n", err)
	} else {
		// Boot API Server for Dashboard Dashboard
		go api.StartServer("8080", db)
	}

	minimaxClient := minimax.NewClient(cfg.MiniMaxAPIKey, cfg.MiniMaxGroupID)
	veniceClient := venice.NewClient(cfg.VeniceAPIKey, cfg.VeniceModel)

	openCodeRunner := &steps.OpenCodeRunner{
		WorkDir: cfg.ProjectDir,
	}

	orch := &orchestrator.Orchestrator{
		DB:        db,
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
		tgBot = telegram.NewBot(cfg.TelegramBotToken, cfg.TelegramChatID, cfg.ProjectDir, cfg.MachineName, func(req string) (string, error) {
			// ── Budget Gate (Paperclip-inspired) ──
			if db != nil && db.IsBudgetExceeded() {
				limit, spent, _ := db.GetBudgetStatus()
				return "", fmt.Errorf("🚫 Monthly budget exceeded ($%.2f / $%.2f). Use /budget to adjust", float64(spent)/100, float64(limit)/100)
			}

			// ── Create a mission for tracking ──
			var missionID int64
			if db != nil {
				missionID, _ = db.CreateMission(req)
			}

			// ── Mem0 Context Injection ──
			var memoryContext string
			if db != nil {
				mems, err := db.GetEntityMemories("user", "telegram_user")
				if err == nil && len(mems) > 0 {
					memoryContext = "User Preferences Context:\n" + strings.Join(mems, "\n") + "\n\n"
				}
			}

			// Add the context to the request
			augmentedReq := req
			if memoryContext != "" {
				augmentedReq = memoryContext + "Current Request: " + req
			}

			// ── Execute ──
			var reply string
			var execErr error
			if *useVenice {
				reply, execErr = executeActionWithVenice(missionID, db, augmentedReq, veniceClient, openCodeRunner, cfg.ProjectDir)
			} else {
				reply, execErr = executeActionFromPrompt(missionID, db, augmentedReq, minimaxClient, openCodeRunner, cfg.ProjectDir)
			}

			// ── Audit Trail & Budget ──
			if db != nil {
				source := "minimax"
				if *useVenice {
					source = "venice"
				}
				if execErr != nil {
					db.LogAction(missionID, "ERROR", execErr.Error(), source, 0)
					db.FailMission(missionID, execErr.Error())
				} else {
					db.LogAction(missionID, "COMPLETED", reply, source, 500) // Estimate ~500 tokens per action
					db.CompleteMission(missionID)
					db.AddMissionTokens(missionID, 500)
					db.RecordSpend(1) // ~$0.01 per MiniMax call estimate
				}
			}

			return reply, execErr
		})
		tgStopChan := make(chan struct{})
		go tgBot.PollForRepoURLs(tgStopChan)
		defer close(tgStopChan)

		// Start Proactive Heartbeat (System Insight Scan)
		hb := orchestrator.NewHeartbeat(2 * time.Minute) // Shortened for 'Agency' responsiveness
		hb.Start(func() {
			fmt.Println("\n💓 Heartbeat: Agency Performance Scan...")
			screenPath := "/tmp/nanoclaw_heartbeat_screen.png"
			if err := computer.TakeScreenshot(screenPath); err == nil {
				prompt := "Role: CEO. Analyze the screen. If you see a code error, terminal crash, or something obviously broken, reply with 'MISSION: [RECOVERY]: <fix description>'. If everything looks good, reply 'Status: Nominal'."
				
				var analysis string
				var err error
				if *useVenice {
					analysis, err = veniceClient.GenerateAction(prompt, "System Monitor Mode")
				} else {
					analysis, mmErr := minimaxClient.AnalyzeScreen(screenPath, prompt)
					analysis = analysis // MiniMax returns string directly
					err = mmErr
				}

				if err == nil && strings.Contains(analysis, "MISSION:") {
					// ── Autonomous Recovery Trigger (Claw-Code Power) ──
					re := regexp.MustCompile(`MISSION: \[(.*)\]: (.*)`)
					matches := re.FindSubmatch([]byte(analysis))
					if len(matches) >= 3 {
						fixRef := string(matches[2])
						tgBot.SendMessage("🚨 [PROACTIVE] Crisis Detected: " + fixRef + "\n⚡️ Agency is auto-fixing now...")
						
						// Create mission
						missionID, _ := db.CreateMission("Autonomous Recovery: " + fixRef)
						go executeActionFromPrompt(missionID, db, fixRef, minimaxClient, openCodeRunner, cfg.ProjectDir)
					}
				}
			}
		})
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
		shouldExit := false
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "exit" {
				fmt.Println("Exiting NanoClaw...")
				shouldExit = true
				break
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

		if shouldExit {
			break
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

		// ── Create a mission for tracking ──
		var missionID int64
		if db != nil {
			missionID, _ = db.CreateMission(prompt)
		}

		_, err = orch.RunStep(orchestrator.WorkflowStep{
			Name: fmt.Sprintf("prompt_%d", i),
			Action: func() (string, error) {
				if *useVenice {
					return executeActionWithVenice(missionID, db, prompt, veniceClient, openCodeRunner, cfg.ProjectDir)
				}
				return executeActionFromPrompt(missionID, db, prompt, minimaxClient, openCodeRunner, cfg.ProjectDir)
			},
		})
		if err != nil {
			fmt.Printf("Error running prompt: %v\n", err)
		}
	}

	fmt.Printf("\n✅ Project Complete: %s\n", *project)
}

func printBanner() {
	fmt.Print(`
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
func executeActionFromPrompt(missionID int64, db *internal.Database, prompt string, minimaxClient *minimax.Client, openCodeRunner *steps.OpenCodeRunner, projectDir string) (string, error) {
	fmt.Println("🧠 MiniMax is analyzing the mission...")

	plan, err := minimaxClient.GetManagerPlan(prompt)
	if err != nil {
		return "", fmt.Errorf("MiniMax plan error: %v", err)
	}

	tasks := orchestrator.ParseDelegations(plan)
	if len(tasks) == 0 {
		return executeSingleAction(prompt, minimaxClient, openCodeRunner, projectDir)
	}

	orch := &orchestrator.Orchestrator{DB: db}
	handler := func(taskPrompt string, role string) (string, error) {
		return executeSingleAction(taskPrompt, minimaxClient, openCodeRunner, projectDir)
	}

	return orch.ExecuteAutonomousMission(missionID, prompt, minimaxClient.GetManagerPlan, handler)
}

// executeSingleAction performs a standalone atomic action.
func executeSingleAction(prompt string, minimaxClient *minimax.Client, openCodeRunner *steps.OpenCodeRunner, projectDir string) (string, error) {
	actionData, err := minimaxClient.GetDesktopAction(prompt)
	if err != nil {
		return "", fmt.Errorf("MiniMax action error: %v", err)
	}

	return executeSingleActionInternal(actionData, minimaxClient, openCodeRunner, projectDir)
}

// executeActionWithVenice uses the Venice/Mithril model to plan and execute tasks.
func executeActionWithVenice(missionID int64, db *internal.Database, prompt string, veniceClient *venice.Client, openCodeRunner *steps.OpenCodeRunner, projectDir string) (string, error) {
	fmt.Println("⚪ Venice/Mithril is creating a mission plan...")
	
	systemPrompt := `You are the ARCHITECT AGENT for NanoClaw.
Receive the directive and break it into a logical sequence of subtasks.
Format for delegations: -> [ROLE]: task description
Roles: CODER, RESEARCHER, DESIGNER, TESTER.`

	plan, err := veniceClient.GenerateAction(prompt, systemPrompt)
	if err != nil {
		return "", fmt.Errorf("venice plan error: %v", err)
	}

	tasks := orchestrator.ParseDelegations(plan)
	if len(tasks) == 0 {
		return executeSingleActionWithVenice(prompt, veniceClient, openCodeRunner, projectDir)
	}

	orch := &orchestrator.Orchestrator{DB: db}
	handler := func(taskPrompt string, role string) (string, error) {
		return executeSingleActionWithVenice(taskPrompt, veniceClient, openCodeRunner, projectDir)
	}

	return orch.ExecuteAutonomousMission(missionID, prompt, func(ctx string) (string, error) {
		return veniceClient.GenerateAction(ctx, systemPrompt)
	}, handler)
}

// executeSingleActionWithVenice maps a prompt to a specific tool using Venice.
func executeSingleActionWithVenice(prompt string, veniceClient *venice.Client, openCodeRunner *steps.OpenCodeRunner, projectDir string) (string, error) {
	systemPrompt := `You are an AI desktop agent. Map the request to a tool.
Tools: OpenDocument, CreateDocument, WebSearch, YouTubeSearch, TerminalCommand, OpenCode, AgentDesktopControl, ChatResponse
Respond with ONLY ONE valid JSON object: {"action": "ActionName", ...args}`

	content, err := veniceClient.GenerateAction(prompt, systemPrompt)
	if err != nil {
		return "", err
	}

	var actionData map[string]string
	if err := minimax.DecodeFirstJSON(content, &actionData); err != nil {
		return "", err
	}

	return executeSingleActionInternal(actionData, nil, openCodeRunner, projectDir)
}

// executeSingleActionInternal is a helper that wraps the common action switch
func executeSingleActionInternal(actionData map[string]string, minimaxClient *minimax.Client, openCodeRunner *steps.OpenCodeRunner, projectDir string) (string, error) {
	actionName := actionData["action"]
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
	case "WhatsApp":
		cmdErr = computer.OpenWhatsApp(actionData["phone"], actionData["text"])
	case "Paint":
		cmdErr = computer.OpenPaint()
	case "RemoteCommand":
		res, err := computer.ExecuteRemoteCommand(actionData["host"], actionData["user"], actionData["command"])
		if err != nil {
			return res, err
		}
		return fmt.Sprintf("Remote result from %s:\n%s", actionData["host"], res), nil
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
		if minimaxClient != nil {
			out, errCmd := executeAutonomousLoop(actionData["prompt"], minimaxClient)
			return out, errCmd
		}
		return "AgentDesktopControl (Vision) currently requires MiniMax for visual reasoning.", nil
	case "ChatResponse":
		return actionData["reply"], nil
	default:
		return "", fmt.Errorf("unknown action: %s", actionName)
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
