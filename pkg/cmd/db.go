package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/vshn/vshn-sli-reporting/pkg/store"
)

var (
	dbCommandName = "db"
	dbCmd         = &cobra.Command{
		Use:   dbCommandName,
		Short: "Database operations",
	}
	initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize database if it does not yet exist",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Initializing database...")
			var store, err = store.NewDowntimeStore(dbPath, nil)
			if err != nil {
				log.Fatal(err)
				return
			}
			defer store.CloseDB()
			err = store.InitializeDB()
			if err != nil {
				log.Fatal(err)
				return
			}
			fmt.Println("Database has been initialized")
		},
	}
)

func init() {
	initCmd.Flags().StringVar(&dbPath, "db-file", "./data.db", "Path of the SQLite DB file")

	dbCmd.AddCommand(initCmd)
	rootCmd.AddCommand(dbCmd)
}
