package controllers

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"log/slog"
	"net/http"

	types "example.com/ticket-system/internal/http"
	"example.com/ticket-system/internal/models"
	"example.com/ticket-system/internal/repositories"
	"github.com/gin-gonic/gin"
)

var (
	ErrBadRequest     = errors.New("bad request")
	ErrImportError    = errors.New("import failure")
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
	var req types.CreateTicketRequest
	err := c.BindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return
	}
	id, err := tc.repo.CreateTicket(ctx, req.ToTicket())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create ticket", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return
	}
	slog.InfoContext(ctx, "Ticket created with", "id", id)
	c.JSON(200, types.CreateTicketResponse{
		Id: id,
	})
}

func (tc *ticketController) GetTicketDetails(ctx context.Context, c *gin.Context) {
	id := c.Param("id")

	ticket, err := tc.repo.GetTicket(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get ticket", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return
	}

	c.JSON(200, gin.H{
		"ticket": ticket,
	})

}

func (tc *ticketController) GetTicketsAssignedToSupportUser(ctx context.Context, c *gin.Context) {
	var request types.GetTicketsAssignedToRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		slog.ErrorContext(ctx, "Failed to read parameters", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return
	}

	tickets, err := tc.repo.GetTicketsAssignedTo(ctx, request.UserName)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get tickets", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return
	}

	c.JSON(200, gin.H{
		"tickets": tickets,
	})

}

func (tc *ticketController) UpdateStatus(ctx context.Context, c *gin.Context) error {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return err
	}

	err := tc.repo.UpdateStatus(ctx, id, req.Status)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update ticket status", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return err
	}

	c.JSON(200, gin.H{"message": "status updated"})
	return nil
}

func (tc *ticketController) UpdateAssignTo(ctx context.Context, c *gin.Context) {

	var request types.AssignToRequest
	request.TicketID = c.Param("id")

	if err := c.BindJSON(&request); err != nil {
		slog.ErrorContext(ctx, "Failed to assign ticket", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return
	}

	err := tc.repo.UpdateAssignTo(ctx, request.TicketID, request.Assignee)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to assign ticket", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "assignee updated"})
}

func (tc *ticketController) BulkCreate(ctx context.Context, c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		slog.ErrorContext(ctx, "Failed to receive file", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.TrimLeadingSpace = true

	// returns lines numbers of failed records
	badRecords := make(map[int]string)

	lineNumber := 0
	var entries []models.Ticket

	//NOTE: this should be part of a service
	for {
		lineNumber += 1
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.ErrorContext(ctx, "Error parsing ticket from CSV", "error", err)
			continue
		}

		ticketRecord := models.BulkImportRecord{}
		err = ticketRecord.LoadFromRecord(record)
		if err != nil {
			slog.ErrorContext(ctx, "Error parsing ticket from CSV", "error", err)
			badRecords[lineNumber] = err.Error()
			continue

		}
		entries = append(entries, ticketRecord.ToTicket())
	}

	err = tc.repo.BulkImport(ctx, entries)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to import records", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrImportError.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "OK", "entries": entries})

}
