package main

import (
	"context"
	"fmt"

	"github.com/projuktisheba/erp-mini-api/api"
)

// startup is called at application startup
func main() {
	ctx := context.Background()
	// Start backend server
	if err := api.RunServer(ctx); err != nil {
		fmt.Printf("Failed to start backend server: %v\n", err)
	}
}