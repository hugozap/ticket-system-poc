package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	types "example.com/ticket-system/internal/http"
	"example.com/ticket-system/internal/models"
	"example.com/ticket-system/internal/repositories"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    types.CreateTicketRequest
		mockSetup      func(*repositories.MockTicketRepository)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful ticket creation",
			requestBody: types.CreateTicketRequest{
				Description: "Test ticket",
				CreatedBy:   "testuser",
			},
			mockSetup: func(mockRepo *repositories.MockTicketRepository) {
				mockRepo.EXPECT().CreateTicket(mock.Anything, mock.MatchedBy(func(ticket *models.Ticket) bool {
					return ticket.Description == "Test ticket" &&
						ticket.CreatedBy == "testuser" &&
						ticket.Status == models.StatusOpen &&
						ticket.AssignedTo == "None"
				})).Return("ticket-123", nil)
			},
			expectedStatus: 200,
			expectedBody:   `{"id":"ticket-123"}`,
		},
		{
			name: "repository error",
			requestBody: types.CreateTicketRequest{
				Description: "Test ticket",
				CreatedBy:   "testuser",
			},
			mockSetup: func(mockRepo *repositories.MockTicketRepository) {
				mockRepo.EXPECT().CreateTicket(mock.Anything, mock.Anything).Return("", assert.AnError)
			},
			expectedStatus: 400,
			expectedBody:   `{"error":"bad request"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repositories.NewMockTicketRepository(t)
			controller := NewTicketController(mockRepo)

			tt.mockSetup(mockRepo)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/tickets", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			controller.CreateTicket(context.Background(), c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestGetTicketDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		ticketID       string
		mockSetup      func(*repositories.MockTicketRepository)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "successful ticket retrieval",
			ticketID: "ticket-123",
			mockSetup: func(mockRepo *repositories.MockTicketRepository) {
				ticket := &models.Ticket{
					TicketID:    "ticket-123",
					Description: "Test ticket",
					Status:      models.StatusOpen,
					CreatedBy:   "testuser",
					AssignedTo:  "None",
					CreatedAt:   "2023-01-01T00:00:00Z",
				}
				mockRepo.EXPECT().GetTicket(mock.Anything, "ticket-123").Return(ticket, nil)
			},
			expectedStatus: 200,
			expectedBody: `{
				"ticket": {
					"TicketID": "ticket-123",
					"Description": "Test ticket",
					"Status": "OPEN",
					"CreatedBy": "testuser",
					"CreatedAt": "2023-01-01T00:00:00Z",
					"AssignedTo": "None"
				}
			}`,
		},
		{
			name:     "repository error",
			ticketID: "ticket-456",
			mockSetup: func(mockRepo *repositories.MockTicketRepository) {
				mockRepo.EXPECT().GetTicket(mock.Anything, "ticket-456").Return(nil, assert.AnError)
			},
			expectedStatus: 400,
			expectedBody:   `{"error":"bad request"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repositories.NewMockTicketRepository(t)
			controller := NewTicketController(mockRepo)

			tt.mockSetup(mockRepo)

			req := httptest.NewRequest(http.MethodGet, "/tickets/"+tt.ticketID, nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: tt.ticketID}}

			controller.GetTicketDetails(context.Background(), c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}
