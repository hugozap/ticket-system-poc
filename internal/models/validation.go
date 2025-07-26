package models

func (m *Ticket) ValidateStatus() bool {
	validStatuses := map[TicketStatus]bool{
		StatusClosed: true,
		StatusOpen:   true,
	}
	return validStatuses[m.Status]
}
