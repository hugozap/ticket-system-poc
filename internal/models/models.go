package models

import (
	"errors"
	"time"
)

type TicketStatus string

const (
	StatusOpen   TicketStatus = "OPEN"
	StatusClosed TicketStatus = "CLOSED"
)

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

// Bulk import models

type BulkImportRecord struct {
	ID          string
	Description string
	Status      string
	AssignedTo  string
	CreatedBy   string
}

func (bi *BulkImportRecord) LoadFromRecord(record []string) error {
	if len(record) < 4 {
		return errors.New("bulk import - record should have 5 columns")
	}

	bi.ID = record[0]
	bi.Description = record[1]
	bi.Status = record[2]
	bi.AssignedTo = record[3]
	bi.CreatedBy = record[4]
	return nil
}

// returns the first validation error found for the record
func (bi *BulkImportRecord) Validate() error {
	validStatuses := map[TicketStatus]bool{
		StatusClosed: true,
		StatusOpen:   true,
	}

	validStatus := validStatuses[TicketStatus(bi.Status)]
	if !validStatus {
		return errors.New("wrong column - status")
	}
	if len(bi.ID) == 0 {
		return errors.New("wrong column - id")
	}
	if len(bi.Description) == 0 {
		return errors.New("wrong column - description")
	}

	return nil
}

func (bi *BulkImportRecord) ToTicket() Ticket {
	return Ticket{
		TicketID:    bi.ID,
		Description: bi.Description,
		CreatedBy:   bi.CreatedBy,
		AssignedTo:  bi.AssignedTo,
		CreatedAt:   time.Now().UTC().String(),
	}
}
