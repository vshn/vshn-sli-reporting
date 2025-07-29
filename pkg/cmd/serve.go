package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/vshn/vshn-sli-reporting/pkg/api"
	"github.com/vshn/vshn-sli-reporting/pkg/lieutenant"
	"github.com/vshn/vshn-sli-reporting/pkg/store"
)

var (
	serverCommandName = "serve"
	serverConfig      = api.ApiServerConfig{}
	lieutenantConfig  = lieutenant.Config{}
	dbPath            string
	serveCmd          = &cobra.Command{
		Use:   serverCommandName,
		Short: "Serve API endpoints",
		Long:  "Serve API endpoints",
		Run: func(cmd *cobra.Command, args []string) {
			lieutenant, err := lieutenant.NewLieutenantClient(lieutenantConfig)
			if err != nil {
				log.Fatal(err)
				return
			}
			store, err := store.NewDowntimeStore(dbPath, lieutenant)
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
	serveCmd.Flags().StringVar(&lieutenantConfig.Host, "lieutenant-k8s-url", "0.0.0.0", "URL of Lieutenant Kubernetes API")
	serveCmd.Flags().StringVar(&lieutenantConfig.Token, "lieutenant-sa-token", "", "Service Account token of Lieutenant Kubernetes API")
	serveCmd.Flags().StringVar(&lieutenantConfig.Namespace, "lieutenant-namespace", "lieutenant", "Namespace in which Clusters are stored in Lieutenant")

	rootCmd.AddCommand(serveCmd)
}
