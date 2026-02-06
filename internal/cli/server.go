package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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

		quotaBytes := parseQuotaBytes(viper.GetString("quota"))
		if quotaBytes > 0 {
			log.Printf("Quota: %d bytes (%.2f GB) — drive will report this size to clients", quotaBytes, float64(quotaBytes)/(1<<30))
		}

		srv := server.New(addr, absDataDir, store, quotaBytes)

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
	serverCmd.Flags().String("quota", "", "Storage quota to report to clients (e.g. 2G, 512M). If set, the mapped drive shows this size instead of the host disk.")

	// Bind flags to viper
	viper.BindPFlag("port", serverCmd.Flags().Lookup("port"))
	viper.BindPFlag("data_dir", serverCmd.Flags().Lookup("data-dir"))
	viper.BindPFlag("quota", serverCmd.Flags().Lookup("quota"))
}

// parseQuotaBytes parses a size string like "2G", "512M", "1G" into bytes. Returns 0 for empty or invalid.
func parseQuotaBytes(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	s = strings.ToUpper(s)
	var mult uint64 = 1
	if strings.HasSuffix(s, "B") && (strings.HasSuffix(s, "GB") || strings.HasSuffix(s, "MB") || strings.HasSuffix(s, "KB")) {
		s = s[:len(s)-1]
	}
	if strings.HasSuffix(s, "G") {
		mult = 1 << 30
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "M") {
		mult = 1 << 20
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "K") {
		mult = 1 << 10
		s = s[:len(s)-1]
	}
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n * mult
}
