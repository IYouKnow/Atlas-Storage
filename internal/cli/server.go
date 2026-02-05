package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/IYouKnow/atlas-drive/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the Atlas storage server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		port := viper.GetString("port")
		if port == "" {
			port = "8080"
		}
		addr := ":" + port

		dataDir := viper.GetString("data_dir")
		if dataDir == "" {
			// Default to ./data
			dataDir = "data"
		}

		// Resolve absolute paths for clarity
		absDataDir, _ := filepath.Abs(dataDir)

		// Get User Store
		store, err := getUserStore()
		if err != nil {
			return fmt.Errorf("failed to load user store: %w", err)
		}

		if len(store.Users) == 0 {
			log.Println("WARNING: No users defined. Server will reject all connections. Use 'atlas user add' to create a user.")
		}

		srv := server.New(addr, absDataDir, store)

		// Graceful Shutdown Channel
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		go func() {
			if err := srv.Start(); err != nil {
				log.Fatalf("Server failed: %v", err)
			}
		}()

		<-stop
		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}

		log.Println("Server stopped gracefully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Flags
	serverCmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	serverCmd.Flags().StringP("data-dir", "d", "data", "Directory to store data files")

	// Bind flags to viper
	viper.BindPFlag("port", serverCmd.Flags().Lookup("port"))
	viper.BindPFlag("data_dir", serverCmd.Flags().Lookup("data-dir"))
}
