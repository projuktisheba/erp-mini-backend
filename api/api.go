package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	api "github.com/projuktisheba/erp-mini-api/api/handlers"
	"github.com/projuktisheba/erp-mini-api/internal/config"
	"github.com/projuktisheba/erp-mini-api/internal/dbrepo"
	"github.com/projuktisheba/erp-mini-api/internal/driver"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

// application is the receiver for the various parts of the application
type application struct {
	config   models.Config
	infoLog  *log.Logger
	errorLog *log.Logger
	version  string
	Handlers *api.HandlerRepo
	DB       *dbrepo.DBRepository
	Server   *http.Server
	ctx      context.Context
}

var app *application

// serve starts the server and listens for requests
func (app *application) serve() error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.Port),
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
	}

	app.Server = srv
	app.infoLog.Printf("Starting HTTP Back end server in %s mode on port %d", app.config.Env, app.config.Port)
	app.infoLog.Println(".....................................")
	return srv.ListenAndServe()
}

// ShutdownServer gracefully shuts down the server
func (app *application) ShutdownServer() error {
	// Create a context with a timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	app.infoLog.Println("Shutting down the server gracefully...")
	// Shutdown the server with the context
	if err := app.Server.Shutdown(ctx); err != nil {
		app.errorLog.Printf("Server forced to shutdown: %s", err)
		return err
	}

	app.infoLog.Println("Server exited gracefully")
	return nil
}

// RunServer is the application entry point
func RunServer(ctx context.Context) error {
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	var cfg models.Config
	//get environment variables
	cfg, err := config.Load()
	if err != nil {
		errorLog.Println(err)
		return err
	}
	cfg.JWT = models.JWTConfig{
		SecretKey: "your_secret_key_here",
		Issuer:    "myapp",
		Audience:  "myapp_users",
		Algorithm: "HS256",
		Expiry:    time.Hour * 24,
	}

	infoLog.Println(cfg)
	// Connection to database
	var dbConn *pgxpool.Pool
	if cfg.Env == "live" {
		dbConn, err = driver.NewPgxPool(cfg.DB.DSN)
	} else {
		//connect to dev database
		dbConn, err = driver.NewPgxPool(cfg.DB.DEVDSN)
	}

	if err != nil {
		errorLog.Println(err)
		return err
	}
	defer dbConn.Close()

	dbRepo := dbrepo.NewDBRepository(dbConn)
	infoLog.Println("Connected to database")

	//Initiate handlers
	app = &application{
		config:   cfg,
		infoLog:  infoLog,
		errorLog: errorLog,
		version:  "1.0.0",
		Handlers: api.NewHandlerRepo(dbRepo, cfg.JWT, infoLog, errorLog),
		DB:       dbRepo,
		ctx:      ctx,
	}

	// Run the server in a separate goroutine so we can wait for shutdown signals
	go func() {
		if err := app.serve(); err != nil {
			errorLog.Printf("Error starting server: %s", err)
		}
	}()

	// Channel to listen for OS interrupt signals (e.g., from Ctrl+C)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Wait for shutdown signal
	<-stop

	// Call ShutdownServer to gracefully shut down the server
	return app.ShutdownServer()
}

// Stop server from outer module
func StopServer() error {
	return app.ShutdownServer()
}
