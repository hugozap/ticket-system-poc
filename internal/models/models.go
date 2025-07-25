package models

import (
	"time"
)

type TicketStatus string

const (
	StatusOpen   TicketStatus = "OPEN"
	StatusClosed TicketStatus = "CLOSED"
)

type CreateTicketRequest struct {
	Description string `json:"description"`
	CreatedBy   string `json:"createdBy"`
}

type CreateTicketResponse struct {
	Id string `json:"id"`
}

// Generates a ticket that has just created
func (tr *CreateTicketRequest) ToTicket() *Ticket {
	return &Ticket{
		Description: tr.Description,
		CreatedBy:   tr.CreatedBy,
		CreatedAt:   time.Now().UTC().String(),
		Status:      StatusOpen,
		AssignedTo:  "None",
	}
}

type Ticket struct {
	TicketID    string       `dynamodbav:"ticket_id"`
	Description string       `dynamodbav:"description"`
	Status      TicketStatus `dynamodbav:"status"`
	CreatedBy   string       `dynamodbav:"createdBy"`
	CreatedAt   string       `dynamodbav:"createdAt"`
	AssignedTo  string       `dynamodbav:"assignedTo"`
}

type TicketDbRecord struct {
	Ticket
	PK string `dynamodbav:"PK"`
	SK string `dynamodbav:"SK"`
}
