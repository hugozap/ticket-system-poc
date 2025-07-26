package repositories

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	models "example.com/ticket-system/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

const (
	TableName = "tickets_poc"
)

var (
	ErrCreatingTicket        = errors.New("error creating ticket in database")
	ErrLoadingTicket         = errors.New("error loading ticket from database")
	ErrLoadingTicketsForUser = errors.New("could not load user assigned tickets")
	ErrInvalidStatus         = errors.New("invalid status")
)

type TicketRepository interface {
	CreateTicket(ctx context.Context, ticket *models.Ticket) (string, error)
	GetTicket(ctx context.Context, id string) (*models.Ticket, error)
	GetTicketsAssignedTo(ctx context.Context, userName string) ([]models.Ticket, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateTicket(ctx context.Context, ticket *models.Ticket) error
	UpdateAssignTo(ctx context.Context, id string, assignTo string) error
	BulkImport(ctx context.Context, entries []models.Ticket) error
}

type ticketRepository struct {
	client *dynamodb.Client
}

func NewTicketRepository(ctx context.Context) *ticketRepository {
	cfg, _ := config.LoadDefaultConfig(ctx)
	client := dynamodb.NewFromConfig(cfg)
	return &ticketRepository{
		client: client,
	}
}
func (tr *ticketRepository) GetTicket(ctx context.Context, id string) (*models.Ticket, error) {
	result, err := tr.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("#ticket#%s", id)},
			"SK": &types.AttributeValueMemberS{Value: "details"},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("%w - %w", ErrLoadingTicket, err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("%w - ticket not found", ErrLoadingTicket)
	}

	var ticketRecord models.TicketDbRecord
	err = attributevalue.UnmarshalMap(result.Item, &ticketRecord)
	if err != nil {
		return nil, fmt.Errorf("%w - %w", ErrLoadingTicket, err)
	}

	return &ticketRecord.Ticket, nil

}

func (tr *ticketRepository) GetTicketsAssignedTo(ctx context.Context, userName string) ([]models.Ticket, error) {
	// Assuming a GSI named "AssignedTo-index" with "AssignedTo" as the partition key exists
	input := &dynamodb.QueryInput{
		TableName:              aws.String(TableName),
		IndexName:              aws.String("AssignedTo"),
		KeyConditionExpression: aws.String("AssignedTo = :assignedTo"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":assignedTo": &types.AttributeValueMemberS{Value: userName},
		},
	}

	result, err := tr.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("%w - %w", ErrLoadingTicketsForUser, err)
	}

	var ticketRecords []models.TicketDbRecord

	for _, item := range result.Items {
		var ticketRecord models.TicketDbRecord
		err = attributevalue.UnmarshalMap(item, &ticketRecord)
		if err != nil {
			return nil, fmt.Errorf("%w - %w", ErrLoadingTicket, err)
		}
		ticketRecords = append(ticketRecords, ticketRecord)
	}

	var tickets []models.Ticket
	for _, record := range ticketRecords {
		ticket := record.Ticket
		tickets = append(tickets, ticket)
	}

	return tickets, nil

}

// Creates a new ticket and returns the ticket id
func (tr *ticketRepository) CreateTicket(ctx context.Context, ticket *models.Ticket) (string, error) {
	slog.InfoContext(ctx, "Creating Ticket", "ticket", ticket)
	if ticket.TicketID == "" {
		ticket.TicketID = uuid.NewString()
	}
	ticketRecord := models.TicketDbRecord{
		Ticket: *ticket,
		PK:     fmt.Sprintf("#ticket#%s", ticket.TicketID),
		SK:     "details",
	}
	item, err := attributevalue.MarshalMap(ticketRecord)
	if err != nil {
		return "", fmt.Errorf("%w - %w", ErrCreatingTicket, err)
	}

	_, err = tr.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	})

	if err != nil {
		return "", fmt.Errorf("%w - %w", ErrCreatingTicket, err)
	}

	return ticket.TicketID, nil

}

func (tr *ticketRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	ticket, err := tr.GetTicket(ctx, id)
	if err != nil {
		return err
	}

	ticket.Status = models.TicketStatus(status)
	valid := ticket.ValidateStatus()
	if !valid {
		return fmt.Errorf("%s - %s", ErrInvalidStatus, status)
	}
	return tr.UpdateTicket(ctx, ticket)
}

func (tr *ticketRepository) UpdateAssignTo(ctx context.Context, id string, assignTo string) error {
	ticket, err := tr.GetTicket(ctx, id)
	if err != nil {
		return err
	}
	ticket.AssignedTo = assignTo
	return tr.UpdateTicket(ctx, ticket)
}

func (tr *ticketRepository) UpdateTicket(ctx context.Context, ticket *models.Ticket) error {
	if ticket.TicketID == "" {
		return errors.New("ticket ID required")
	}
	ticketRecord := models.TicketDbRecord{
		Ticket: *ticket,
		PK:     fmt.Sprintf("#ticket#%s", ticket.TicketID),
		SK:     "details",
	}
	item, err := attributevalue.MarshalMap(ticketRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal ticket: %w", err)
	}

	_, err = tr.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}
	return nil
}

func (tr *ticketRepository) BulkImport(ctx context.Context, entries []models.Ticket) error {
	const batchSize = 40

	// process in batches of batchSize
	for i := 0; i < len(entries); i += batchSize {
		end := i + batchSize
		if end > len(entries) {
			end = len(entries)
		}
		batch := entries[i:end]
		err := tr.processBatch(ctx, batch)
		if err != nil {
			return fmt.Errorf("could not proccess batch from %d to %d", i, end)
		}
	}

	return nil
}

func (tr *ticketRepository) processBatch(ctx context.Context, batch []models.Ticket) error {
	var requests []types.WriteRequest
	for _, entry := range batch {

		ticketRecord := models.TicketDbRecord{
			Ticket: entry,
			PK:     fmt.Sprintf("#ticket#%s", entry.TicketID),
			SK:     "details",
		}

		item, err := attributevalue.MarshalMap(ticketRecord)

		if err != nil {
			return fmt.Errorf("error marshalling record")
		}

		req := types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: item,
			},
		}
		requests = append(requests, req)
	}
	input := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			TableName: requests,
		},
	}
	_, err := tr.client.BatchWriteItem(ctx, input)
	if err != nil {
		return fmt.Errorf("batch write failed: %w", err)
	}
	return nil
}
