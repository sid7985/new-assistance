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

// ExecuteMission runs a sequence of tasks until completion or failure.
func (o *Orchestrator) ExecuteMission(tasks []Task, handler func(string, string) (string, error)) (string, error) {
	fmt.Printf("\n🚀 Starting Mission with %d tasks...\n", len(tasks))
	var finalResults []string

	for i, task := range tasks {
		fmt.Printf("\n📋 [%d/%d] Role: %s | Task: %s\n", i+1, len(tasks), task.AssignedTo, task.Description)
		
		// Map Role to a refined prompt for the handler
		rolePrompt := fmt.Sprintf("Role: %s. Task: %s", task.AssignedTo, task.Description)
		result, err := handler(rolePrompt, task.AssignedTo)
		
		if err != nil {
			fmt.Printf("❌ Task failed: %v\n", err)
			return "", fmt.Errorf("mission failed at task %d (%s): %w", i+1, task.AssignedTo, err)
		}
		
		fmt.Printf("✅ Task completed: %s\n", result)
		finalResults = append(finalResults, fmt.Sprintf("- %s: %s", task.AssignedTo, result))
	}

	fmt.Println("\n✨ Mission successfully completed!")
	return strings.Join(finalResults, "\n"), nil
}

