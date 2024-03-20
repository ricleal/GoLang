// To execute Go code, please declare a func main() in a package "main"

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"
)

type Scheduler struct {
	tasks map[int64][]Task
}

func (s *Scheduler) Register(t Task) {
	s.tasks[t.Schedule()] = append(s.tasks[t.Schedule()], t)
}

func TaskRunner(ctx context.Context, taskCh <-chan Task) {
	for t := range taskCh {
		t.Run(ctx)
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
	Run(ctx context.Context)
	Schedule() int64
}

type Email struct {
	schedule int64
}

func (e *Email) Run(ctx context.Context) {
	fmt.Printf("Email %+v run\n", e)
}

func (e *Email) Schedule() int64 {
	return e.schedule
}

func NewEmail(schedule int64) *Email {
	return &Email{schedule: schedule}
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
	fmt.Println("Main one!")
}
