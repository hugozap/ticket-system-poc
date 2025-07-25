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
	ErrCreatingTicket = errors.New("error creating ticket in database")
	ErrLoadingTicket  = errors.New("error loading ticket from database")
)

type TicketRepository interface {
	CreateTicket(ctx context.Context, ticket *models.Ticket) (string, error)
	GetTicket(ctx context.Context, id string) (*models.Ticket, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateTicket(ctx context.Context, ticket *models.Ticket) error
	UpdateAssignTo(ctx context.Context, id string, assignTo string) error
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
	//TODO: validate possible status values

	ticket.Status = models.TicketStatus(status)
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
