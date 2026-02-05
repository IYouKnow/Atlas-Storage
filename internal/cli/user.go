package cli

import (
	"fmt"
	"path/filepath"

	"github.com/IYouKnow/atlas-drive/pkg/user"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Long:  `Add, remove, and list users for the Atlas server.`,
}

var userAddCmd = &cobra.Command{
	Use:   "add [username] [password]",
	Short: "Add a new user",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := getUserStore()
		if err != nil {
			return err
		}

		username := args[0]
		password := args[1]

		if err := store.Add(username, password); err != nil {
			return err
		}

		if err := store.Save(); err != nil {
			return fmt.Errorf("failed to save user: %w", err)
		}

		fmt.Printf("User %s created successfully.\n", username)
		return nil
	},
}

var userRmCmd = &cobra.Command{
	Use:   "rm [username]",
	Short: "Remove a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := getUserStore()
		if err != nil {
			return err
		}

		username := args[0]
		store.Delete(username)

		if err := store.Save(); err != nil {
			return fmt.Errorf("failed to save changes: %w", err)
		}

		fmt.Printf("User %s removed (if existed).\n", username)
		return nil
	},
}

var userLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := getUserStore()
		if err != nil {
			return err
		}

		users := store.List()
		if len(users) == 0 {
			fmt.Println("No users found.")
			return nil
		}

		fmt.Println("Users:")
		for _, u := range users {
			fmt.Println("-", u)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userRmCmd)
	userCmd.AddCommand(userLsCmd)

	// Define flags for config location if distinct from global config?
	// We reuse global config or env vars.
}

func getUserStore() (*user.Store, error) {
	configDir := viper.GetString("config_dir")
	if configDir == "" {
		// Default to current directory or /var/lib/atlas depending on design.
		// For now simple default relative or absolute.
		// Prefer picking up a generic flag or env var "ATLAS_CONFIG_DIR"
		configDir = "."
	}

	// Create dir if not exists?
	// The user store expects the full path to file? No, NewStore takes simple path.
	// Let's decide on a standard file name.
	dbPath := filepath.Join(configDir, "users.json")
	return user.NewStore(dbPath)
}
