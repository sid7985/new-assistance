package orchestrator

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

type WorkflowStep struct {
	Name   string
	Action func() (string, error)
}

type Orchestrator struct {
	Steps  []WorkflowStep
	Memory map[string]string
}

type Task struct {
	AssignedTo  string
	Description string
	Status      string
}

// ParseDelegations extracts tasks from the manager's plan string.
func ParseDelegations(plan string) []Task {
	var tasks []Task
	// Regex matches "-> [ROLE]: description"
	re := regexp.MustCompile(`(?m)^->\s*\[(.*?)]:?\s*(.*)$`)
	matches := re.FindAllStringSubmatch(plan, -1)

	for _, m := range matches {
		if len(m) >= 3 {
			tasks = append(tasks, Task{
				AssignedTo:  strings.ToUpper(strings.TrimSpace(m[1])),
				Description: strings.TrimSpace(m[2]),
				Status:      "pending",
			})
		}
	}
	return tasks
}

func (o *Orchestrator) AskForConfirmation(actionDesc string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\n⚠️  NanoClaw wants to perform: %s\n", actionDesc)
		fmt.Printf("Do you allow this action? [y/N]: ")
		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		
		response = strings.TrimSpace(strings.ToLower(response))
		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" || response == "" {
			return false
		} else {
			fmt.Println("Please type 'y' for yes or 'n' for no.")
		}
	}
}

func (o *Orchestrator) RunStep(step WorkflowStep) (string, error) {
	var lastErr error
	
	waitTimes := []time.Duration{
		1 * time.Minute,
		3 * time.Minute,
		5 * time.Minute,
	}

	maxAttempts := len(waitTimes) + 1 // Initial attempt + 3 retries = 4 limits total

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Printf("[%s] Running step '%s' (attempt %d/%d)\n", time.Now().Format(time.Kitchen), step.Name, attempt, maxAttempts)

		response, err := step.Action()
		if err != nil {
			lastErr = err
			fmt.Printf("[%s] Step '%s' returned error: %v\n", time.Now().Format(time.Kitchen), step.Name, err)
		}

		if response != "" && err == nil {
			fmt.Printf("[%s] Step '%s' completed successfully\n", time.Now().Format(time.Kitchen), step.Name)
			return response, nil
		}

		// Wait logic if attempt failed
		if attempt < maxAttempts {
			waitTime := waitTimes[attempt-1]
			fmt.Printf("[%s] Step '%s' pending/failed. Waiting %v before retry...\n", time.Now().Format(time.Kitchen), step.Name, waitTime)
			time.Sleep(waitTime)
		}
	}

	return "", fmt.Errorf("step '%s' failed after %d attempts: %w", step.Name, maxAttempts, lastErr)
}

func (o *Orchestrator) RunAll() {
	for _, step := range o.Steps {
		o.RunStep(step)
	}
}

// RunWebStep runs a browser-based action with a fixed 30-second wait on failure.
// Web actions are fast, so no 1m/3m/5m escalation is needed.
func (o *Orchestrator) RunWebStep(step WorkflowStep) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= 2; attempt++ {
		fmt.Printf("[%s] Running web step '%s' (attempt %d/2)\n", time.Now().Format(time.Kitchen), step.Name, attempt)

		response, err := step.Action()
		if err != nil {
			lastErr = err
			fmt.Printf("[%s] Web step '%s' error: %v\n", time.Now().Format(time.Kitchen), step.Name, err)
		}

		if response != "" && err == nil {
			fmt.Printf("[%s] Web step '%s' completed\n", time.Now().Format(time.Kitchen), step.Name)
			return response, nil
		}

		if attempt < 2 {
			fmt.Printf("[%s] Web step '%s' pending. Waiting 30s...\n", time.Now().Format(time.Kitchen), step.Name)
			time.Sleep(30 * time.Second)
		}
	}

	return "", fmt.Errorf("web step '%s' failed after 2 attempts: %w", step.Name, lastErr)
}

func (o *Orchestrator) NewProject(name string) {
	o.Memory = make(map[string]string)
	o.Memory["project"] = name
	fmt.Printf("[%s] Switched to new project: %s\n", time.Now().Format(time.Kitchen), name)
}

// ExecuteAutonomousMission runs a goal-driven loop where MiniMax decides the next step iteratively.
func (o *Orchestrator) ExecuteAutonomousMission(goal string, planFetcher func(string) (string, error), taskHandler func(string, string) (string, error)) (string, error) {
	fmt.Printf("\n🚀 Starting Autonomous MiniMax 2.7 Mission: %s\n", goal)
	
	currentContext := goal
	maxSteps := 15
	var missionHistory []string

	for i := 1; i <= maxSteps; i++ {
		fmt.Printf("\n🧠 [Step %d/%d] Planning next logical action...\n", i, maxSteps)
		
		// Use MiniMax to generate the next sub-plan based on current progress
		plan, err := planFetcher(currentContext)
		if err != nil {
			return "", fmt.Errorf("autonomous planning error: %v", err)
		}

		tasks := ParseDelegations(plan)
		if len(tasks) == 0 {
			fmt.Println("🏁 No more tasks in plan. Mission complete?")
			return strings.Join(missionHistory, "\n"), nil
		}

		// Execute the first task from the plan and then re-evaluate
		task := tasks[0]
		fmt.Printf("📋 Role: %s | Task: %s\n", task.AssignedTo, task.Description)
		
		rolePrompt := fmt.Sprintf("Role: %s. Task: %s. Previous Context: %s", task.AssignedTo, task.Description, strings.Join(missionHistory, " | "))
		result, err := taskHandler(rolePrompt, task.AssignedTo)
		
		if err != nil {
			fmt.Printf("⚠️ Task evaluation: %v\n", err)
			currentContext = fmt.Sprintf("Goal: %s. TASK FAILED: %s. Error: %v. Adjust plan accordingly.", goal, task.Description, err)
		} else {
			fmt.Printf("✅ Task progress: %s\n", result)
			missionHistory = append(missionHistory, fmt.Sprintf("Step %d (%s): %s", i, task.AssignedTo, result))
			currentContext = fmt.Sprintf("Goal: %s. PROGRESS: %s. Decide the FINAL or NEXT step.", goal, strings.Join(missionHistory, " | "))
		}

		// If the agent indicates the mission is complete in its summary or if we see a natural end
		if strings.Contains(strings.ToLower(plan), "mission complete") || strings.Contains(strings.ToLower(plan), "all tasks finished") {
			fmt.Println("🏁 MiniMax signaled mission completion.")
			break
		}
	}

	return strings.Join(missionHistory, "\n"), nil
}

