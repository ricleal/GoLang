// To execute Go code, please declare a func main() in a package "main"

package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TaskResultStatus int

const (
	TaskResultStatusSuccess TaskResultStatus = iota
	TaskResultStatusFailed
	TaskResultStatusPanic
)

type TaskResult struct {
	Time   time.Time
	Status TaskResultStatus
	Err    string
}

type TaskResultStatusHistory struct {
	mu sync.RWMutex
	m  map[string][]TaskResult
}

func (t *TaskResultStatusHistory) Add(id string, status TaskResultStatus, err string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.m[id] = append(t.m[id], TaskResult{
		Time:   time.Now(),
		Status: status,
		Err:    err,
	})
}

func (t *TaskResultStatusHistory) Print() {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for k, v := range t.m {
		fmt.Printf("Task ID: %s\n", k)
		for _, r := range v {
			fmt.Printf("\tTime: %s, Status: %d, Err: %s\n", r.Time, r.Status, r.Err)
		}
	}
}

func NewTaskResultStatusHistory() *TaskResultStatusHistory {
	return &TaskResultStatusHistory{
		m: make(map[string][]TaskResult),
	}
}

// this pretends to be a database
var taskResultStatusHistory = NewTaskResultStatusHistory()

type Scheduler struct {
	tasks map[int64][]Task
}

func (s *Scheduler) Register(t Task) {
	s.tasks[t.Schedule()] = append(s.tasks[t.Schedule()], t)
}

func TaskRunner(ctx context.Context, taskCh <-chan Task) {
	for t := range taskCh {
		func() {
			// recover from panic
			defer func() {
				if r := recover(); r != nil {
					errString := fmt.Sprintf("%+v", r)
					taskResultStatusHistory.Add(t.ID(), TaskResultStatusPanic, errString)
					fmt.Println(errString)
				}
			}()
			err := t.Run(ctx)
			if err != nil {
				errString := fmt.Sprintf("%+v", err)
				taskResultStatusHistory.Add(t.ID(), TaskResultStatusFailed, errString)
				fmt.Println(errString)
			}
		}()
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	taskCh := make(chan Task, 10)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		TaskRunner(ctx, taskCh)
		fmt.Println("TaskRunner done")
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

forcyle:
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Scheduler done")
			break forcyle
		case <-ticker.C:
			nowSecs := time.Now().Unix()
			for k, v := range s.tasks {
				if nowSecs%k == 0 {
					for _, t := range v {
						taskCh <- t
					}
				}
			}
		}
	}
	close(taskCh)
	wg.Wait()
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks: make(map[int64][]Task),
	}
}

// //
type Task interface {
	Run(ctx context.Context) error
	Schedule() int64
	ID() string
}

// Make sure Email implements Task
var _ Task = (*Email)(nil)

type Email struct {
	schedule int64
	uuid     uuid.UUID
}

func (e *Email) Run(ctx context.Context) error {
	r := rand.Intn(10)
	if r == 5 {
		return errors.New("random error")
	}
	if r == 7 {
		panic("random panic")
	}

	fmt.Println(e.uuid.ID(), "succeeded")
	return nil
}

func (e *Email) Schedule() int64 {
	return e.schedule
}

func (e *Email) ID() string {
	return e.uuid.String()
}

func NewEmail(schedule int64) *Email {
	return &Email{
		schedule: schedule,
		uuid:     uuid.New(),
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	e1 := NewEmail(1)
	e2 := NewEmail(2)

	s := NewScheduler()
	s.Register(e1)
	s.Register(e2)

	s.Run(ctx)

	<-ctx.Done()
	taskResultStatusHistory.Print()
	fmt.Println("Main one!")
}
