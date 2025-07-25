package main

import (
	"context"
	"log"
	"os"

	"example.com/ticket-system/internal/http/controllers"
	"example.com/ticket-system/internal/repositories"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
)

var ginLambda *ginadapter.GinLambda

func init() {

	ctx := context.Background()

	if os.Getenv("GIN_MODE") != "" {
		gin.SetMode(os.Getenv("GIN_MODE"))
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	repo := repositories.NewTicketRepository(ctx)
	controller := controllers.NewTicketController(repo)

	// Add CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	router.PUT("/ticket", func(c *gin.Context) {
		controller.CreateTicket(ctx, c)
	})

	router.GET("/ticket/:id", func(c *gin.Context) {
		controller.GetTicketDetails(ctx, c)
	})

	router.PATCH("/ticket/:id/status", func(c *gin.Context) {
		controller.UpdateStatus(ctx, c)
	})

	router.PATCH("/ticket/:id/assignto", func(c *gin.Context) {
		controller.UpdateAssignTo(ctx, c)
	})

	// Catch all for debugging
	router.NoRoute(func(c *gin.Context) {
		log.Printf("No route found for path: %s", c.Request.URL.Path)
		c.JSON(404, gin.H{
			"error":  "Route not found",
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})
	})

	ginLambda = ginadapter.New(router)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
