package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strconv"

	cliadapter "exp/clean_architecture/adapters/cli"
	httpadapter "exp/clean_architecture/adapters/http"
	"exp/clean_architecture/application"
	"exp/clean_architecture/infrastructure/duckdb"
	"exp/clean_architecture/infrastructure/server"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/spf13/cobra"
)

func main() {
	rootCommand := newRootCommand()
	if err := rootCommand.Execute(); err != nil {
		fmt.Fprintln(rootCommand.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var databasePath string
	rootCommand := &cobra.Command{
		Use:   "taskapp",
		Short: "Task Management PoC",
	}
	rootCommand.PersistentFlags().StringVar(&databasePath, "database", "tasks.duckdb", "DuckDB database file")

	rootCommand.AddCommand(newCreateCommand(&databasePath))
	rootCommand.AddCommand(newGetCommand(&databasePath))
	rootCommand.AddCommand(newServeCommand(&databasePath))
	return rootCommand
}

func newCreateCommand(databasePath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "create [title]",
		Short: "Create a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			presenter, closeDatabase, err := buildCLI(*databasePath, command.OutOrStdout())
			if err != nil {
				return err
			}
			defer closeDatabase()
			return presenter.Create(command.Context(), arguments[0])
		},
	}
}

func newGetCommand(databasePath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get [id]",
		Short: "Get a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			id, err := strconv.ParseInt(arguments[0], 10, 64)
			if err != nil || id < 1 {
				return fmt.Errorf("task ID must be a positive integer")
			}
			presenter, closeDatabase, err := buildCLI(*databasePath, command.OutOrStdout())
			if err != nil {
				return err
			}
			defer closeDatabase()
			return presenter.Get(command.Context(), id)
		},
	}
}

func newServeCommand(databasePath *string) *cobra.Command {
	var address string
	command := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API",
		RunE: func(command *cobra.Command, _ []string) error {
			database, repository, err := openRepository(*databasePath)
			if err != nil {
				return err
			}
			defer database.Close()

			createTask := application.NewCreateTask(repository)
			getTask := application.NewGetTask(repository)
			controller := httpadapter.NewTaskController(createTask, getTask)
			return server.New(address, controller).ListenAndServe()
		},
	}
	command.Flags().StringVar(&address, "address", ":8080", "HTTP listen address")
	return command
}

func buildCLI(databasePath string, output io.Writer) (cliadapter.TaskPresenter, func(), error) {
	database, repository, err := openRepository(databasePath)
	if err != nil {
		return cliadapter.TaskPresenter{}, func() {}, err
	}
	return cliadapter.NewTaskPresenter(application.NewCreateTask(repository), application.NewGetTask(repository), output), func() {
		_ = database.Close()
	}, nil
}

func openRepository(databasePath string) (*sql.DB, *duckdb.TaskRepository, error) {
	database, err := sql.Open("duckdb", databasePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open DuckDB: %w", err)
	}
	repository, err := duckdb.NewTaskRepository(database)
	if err != nil {
		_ = database.Close()
		return nil, nil, err
	}
	return database, repository, nil
}
