package controllers

import (
	"context"
	"errors"
	"log/slog"

	"example.com/ticket-system/internal/models"
	"example.com/ticket-system/internal/repositories"
	"github.com/gin-gonic/gin"
)

var (
	ErrBadRequest     = errors.New("bad request")
	ErrCreatingTicket = errors.New("error creating ticket")
)

type TicketController interface {
	CreateTicket(ctx context.Context, c *gin.Context)
	GetTicketDetails(ctx context.Context, c *gin.Context)
	UdpateStatus(ctx context.Context, c *gin.Context)
	AssignTo(ctx context.Context, c *gin.Context)
}
type ticketController struct {
	// Note: in a larger system I would use a service that uses the repository
	// For the POC the controller will directly call repository methods
	repo repositories.TicketRepository
}

func NewTicketController(repo repositories.TicketRepository) ticketController {
	return ticketController{
		repo: repo,
	}
}

func (tc *ticketController) CreateTicket(ctx context.Context, c *gin.Context) {
	var req models.CreateTicketRequest
	err := c.BindJSON(&req)
	if err != nil {
		c.JSON(400, gin.H{"error": ErrBadRequest.Error()})
		return
	}
	id, err := tc.repo.CreateTicket(ctx, req.ToTicket())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create ticket", "error", err)
		c.JSON(400, gin.H{"error": ErrBadRequest.Error()})
		return
	}
	slog.InfoContext(ctx, "Ticket created with", "id", id)
	c.JSON(200, models.CreateTicketResponse{
		Id: id,
	})
}

func (tc *ticketController) GetTicketDetails(ctx context.Context, c *gin.Context) {
	id := c.Param("id")

	ticket, err := tc.repo.GetTicket(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get ticket", "error", err)
		c.JSON(400, gin.H{"error": ErrBadRequest.Error()})
		return
	}

	c.JSON(200, gin.H{
		"ticket": ticket,
	})

}

func (tc *ticketController) UpdateStatus(ctx context.Context, c *gin.Context) error {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": ErrBadRequest.Error()})
		return err
	}

	err := tc.repo.UpdateStatus(ctx, id, req.Status)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update ticket status", "error", err)
		c.JSON(400, gin.H{"error": ErrBadRequest.Error()})
		return err
	}

	c.JSON(200, gin.H{"message": "status updated"})
	return nil
}

func (tc *ticketController) UpdateAssignTo(ctx context.Context, c *gin.Context) error {
	id := c.Param("id")
	var req struct {
		Assignee string `json:"assignee" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": ErrBadRequest.Error()})
		return err
	}

	err := tc.repo.UpdateAssignTo(ctx, id, req.Assignee)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to assign ticket", "error", err)
		c.JSON(400, gin.H{"error": ErrBadRequest.Error()})
		return err
	}

	c.JSON(200, gin.H{"message": "assignee updated"})
	return nil
}
