package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/spf13/cobra"
)

// filename: name of the file to store tasks
var filename = path.Join(os.TempDir(), "tasks.json")

// if file does not exist, create it
// and dump inside an empty array
func init() {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		file, err := os.Create(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer file.Close()

		_, err = file.WriteString("[]")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

// Task: struct for a task
type Task struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Done bool   `json:"done"`
}

// Tasks: slice of Task
type Tasks []Task

// NewTasks
func NewTasks() *Tasks {
	return &Tasks{}
}

// read: read from file and return a list of tasks
func (ts *Tasks) read() (Tasks, error) {
	// read from file and return a list of tasks
	// Open the JSON file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open %q: %v", filename, err)
	}
	defer file.Close()

	var data Tasks
	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("could not decode %q: %v", filename, err)
	}

	return data, nil
}

// write: write to file a list of tasks
func (ts *Tasks) write() error {
	// write to file
	// Open the JSON file
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("could not open %q: %v", filename, err)
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(ts)
	if err != nil {
		return fmt.Errorf("could not encode %q: %v", filename, err)
	}

	return nil
}

// Add: add task to a json file
func (ts *Tasks) Add(taskName string) error {
	// add task to a json file
	// read from file
	tasks, err := ts.read()
	if err != nil {
		return fmt.Errorf("could not get tasks: %v", err)
	}

	// add new task
	newTask := Task{
		ID:   len(tasks) + 1,
		Name: taskName,
		Done: false,
	}
	tasks = append(tasks, newTask)

	// write to file
	err = tasks.write()
	if err != nil {
		return fmt.Errorf("could not write tasks: %v", err)
	}

	return nil
}

// Cleanup: remove done tasks from a json file
func (ts *Tasks) Cleanup() error {
	// remove done tasks from a json file
	// read from file
	tasks, err := ts.read()
	if err != nil {
		return fmt.Errorf("could not get tasks: %v", err)
	}

	// remove done tasks
	var newTasks Tasks
	for _, task := range tasks {
		if !task.Done {
			newTasks = append(newTasks, task)
		}
	}

	// write to file
	err = newTasks.write()
	if err != nil {
		return fmt.Errorf("could not write tasks: %v", err)
	}

	return nil
}

// List: list all tasks
func (ts *Tasks) List() error {
	// list all tasks
	// read from file
	tasks, err := ts.read()
	if err != nil {
		return fmt.Errorf("could not get tasks: %v", err)
	}

	// print all tasks not done!!!
	for _, task := range tasks {
		if !task.Done {
			fmt.Printf("%d. %s\n", task.ID, task.Name)
		}
	}

	return nil
}

// Done: mark task as done
func (ts *Tasks) Done(taskID int) error {
	// mark task as done
	// read from file
	tasks, err := ts.read()
	if err != nil {
		return fmt.Errorf("could not get tasks: %v", err)
	}

	// mark task as done
	for i, task := range tasks {
		if task.ID == taskID {
			tasks[i].Done = true
			break
		}
	}

	// write to file
	err = tasks.write()
	if err != nil {
		return fmt.Errorf("could not write tasks: %v", err)
	}

	return nil
}

// Undone: mark task as undone
func (ts *Tasks) Undone(taskID int) error {
	// mark task as undone
	// read from file
	tasks, err := ts.read()
	if err != nil {
		return fmt.Errorf("could not get tasks: %v", err)
	}

	// mark task as undone
	for i, task := range tasks {
		if task.ID == taskID {
			tasks[i].Done = false
			break
		}
	}

	// write to file
	err = tasks.write()
	if err != nil {
		return fmt.Errorf("could not write tasks: %v", err)
	}

	return nil
}

func main() {
	tasks := NewTasks()
	rootCmd := &cobra.Command{Use: "todolist"}
	cmdAdd := &cobra.Command{
		Use:   "add [Task name]",
		Short: "Add task to the list",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// add task to the list
			if err := tasks.Add(args[0]); err != nil {
				return err
			}
			return nil
		},
	}
	rootCmd.AddCommand(cmdAdd)
	// Implement other commands here

	// Cleanup command
	cmdCleanup := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove done tasks from the list",
		Run: func(cmd *cobra.Command, args []string) {
			// remove done tasks from the list
			err := tasks.Cleanup()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(cmdCleanup)

	// List command
	cmdList := &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		Run: func(cmd *cobra.Command, args []string) {
			// list all tasks
			err := tasks.List()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(cmdList)

	// Done command
	cmdDone := &cobra.Command{
		Use:   "done [Task ID]",
		Short: "Mark task as done",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// mark task as done
			if taskID, err := strconv.Atoi(args[0]); err != nil {
				return err
			} else {
				if err := tasks.Done(taskID); err != nil {
					return err
				}
			}
			return nil
		},
	}
	rootCmd.AddCommand(cmdDone)

	// Undone command
	cmdUndone := &cobra.Command{
		Use:   "undone [Task ID]",
		Short: "Mark task as undone",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// mark task as undone
			taskID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			err = tasks.Undone(taskID)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(cmdUndone)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
