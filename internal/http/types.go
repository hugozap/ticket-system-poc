package types

import (
	"time"

	"example.com/ticket-system/internal/models"
)

type GetTicketsAssignedToRequest struct {
	UserName string `form:"username"`
}

type CreateTicketRequest struct {
	Description string `json:"description"`
	CreatedBy   string `json:"createdBy"`
}

type CreateTicketResponse struct {
	Id string `json:"id"`
}

// Generates a ticket that has just created
func (tr *CreateTicketRequest) ToTicket() *models.Ticket {
	return &models.Ticket{
		Description: tr.Description,
		CreatedBy:   tr.CreatedBy,
		CreatedAt:   time.Now().UTC().String(),
		Status:      models.StatusOpen,
		AssignedTo:  "None",
	}
}
