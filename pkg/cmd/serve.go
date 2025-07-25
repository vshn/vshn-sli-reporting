package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/vshn/vshn-sli-reporting/pkg/api"
	"github.com/vshn/vshn-sli-reporting/pkg/store"
)

var (
	serverCommandName = "serve"
	serverConfig      = api.ApiServerConfig{}
	dbPath            string
	serveCmd          = &cobra.Command{
		Use:   serverCommandName,
		Short: "Serve API endpoints",
		Long:  "Serve API endpoints",
		Run: func(cmd *cobra.Command, args []string) {
			var store, err = store.NewDowntimeStore(dbPath)
			if err != nil {
				log.Fatal(err)
				return
			}
			defer store.CloseDB()
			var server = api.NewApiServer(serverConfig, store)
			fmt.Println("Starting API server ...")
			err = server.Start()
			if err != nil {
				log.Fatal(err)
			}
		},
	}
)

func init() {
	serveCmd.Flags().StringVar(&serverConfig.AuthUser, "auth-user", "admin", "Username for authenticating with the API")
	serveCmd.Flags().StringVar(&serverConfig.AuthPass, "auth-pass", "", "Password for authenticating with the API")
	serveCmd.Flags().StringVar(&dbPath, "db-file", "./data.db", "Path of the SQLite DB file")
	serveCmd.Flags().IntVar(&serverConfig.Port, "port", 8080, "Port at which to serve API")
	serveCmd.Flags().StringVar(&serverConfig.Host, "host", "0.0.0.0", "Host address to bind")

	rootCmd.AddCommand(serveCmd)
}
