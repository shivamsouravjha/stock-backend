package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"stockbackend/routes"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Peer represents the data of a peer stock
type Peer struct {
	Name            string
	PE              float64
	MarketCap       float64
	DividendYield   float64
	ROCE            float64
	QuarterlySales  float64
	QuarterlyProfit float64
}

// QuarterlyData holds historical data for a stock
type QuarterlyData struct {
	NetProfit        float64
	Sales            float64
	TotalLiabilities float64
	ROCE             float64
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, trell-auth-token, trell-app-version-int, creator-space-auth-token")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// GracefulShutdown handles graceful shutdown of the server and ticker
func GracefulShutdown(server *http.Server, ticker *time.Ticker) {
	stopper := make(chan os.Signal, 1)
	// Listen for interrupt and SIGTERM signals
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stopper
		zap.L().Info("Shutting down gracefully...")

		// Stop the ticker
		ticker.Stop()

		// Create a context with a timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Shut down the server
		if err := server.Shutdown(ctx); err != nil {
			zap.L().Error("Server shutdown failed", zap.Error(err))
			return
		}
		zap.L().Info("Server exited gracefully")
	}()
}

func main() {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	logger, _ := config.Build()
	zap.ReplaceGlobals(logger)

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	log.Println("MONGO_URI:", os.Getenv("MONGO_URI"))
	log.Println("CLOUDINARY_URL:", os.Getenv("CLOUDINARY_URL"))

	router := gin.New()
	router.Use(CORSMiddleware())

	ticker := startTicker()

	routes.Routes(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	// Create a server instance using gin engine as handler
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Call GracefulShutdown with the server and ticker
	GracefulShutdown(server, ticker)

	// Start the server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting server: %v", err)
	}

}

func startTicker() *time.Ticker {
	ticker := time.NewTicker(48 * time.Second)

	go func() {
		for t := range ticker.C {
			log.Println("Tick at", t)
			cmd := exec.Command("curl", "https://stock-backend-hz83.onrender.com/api/keepServerRunning")
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Println("Error running curl:", err)
				return
			}

			// Print the output of the curl command
			log.Println("Curl output:", string(output))

		}
	}()

	return ticker
}
