package service

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"github.com/spec-kit/ticket-service/internal/config"
	"github.com/spec-kit/ticket-service/internal/events"
)

// NotificationService handles emitting notifications for domain events.
type NotificationService struct {
	dispatcher events.Dispatcher
	logger     *zap.Logger
	cfg        config.NotificationConfig
}

// NewNotificationService creates the service.
func NewNotificationService(dispatcher events.Dispatcher, logger *zap.Logger, cfg config.NotificationConfig) *NotificationService {
	return &NotificationService{
		dispatcher: dispatcher,
		logger:     logger,
		cfg:        cfg,
	}
}

// RegisterHandlers subscribes to events.
func (n *NotificationService) RegisterHandlers() {
	if n.dispatcher == nil {
		return
	}
	n.dispatcher.Subscribe(events.EventTicketCreated, n.handleTicketCreated)
	n.dispatcher.Subscribe(events.EventTicketStatusChanged, n.handleTicketStatusChanged)
	n.dispatcher.Subscribe(events.EventTicketAssigned, n.handleTicketAssigned)
	n.dispatcher.Subscribe(events.EventTicketMessageAdded, n.handleTicketMessageAdded)
}

func (n *NotificationService) handleTicketCreated(ctx context.Context, event events.Event) error {
	n.logger.Info("TicketCreated", zap.String("ticket_id", event.TicketID), zap.Any("payload", event.Payload))
	n.sendEmailNotificationStub(ctx, event)
	n.sendWebhookNotificationStub(ctx, event)
	return nil
}

func (n *NotificationService) handleTicketStatusChanged(ctx context.Context, event events.Event) error {
	n.logger.Info("TicketStatusChanged", zap.String("ticket_id", event.TicketID), zap.Any("payload", event.Payload))
	n.sendWebhookNotificationStub(ctx, event)
	return nil
}

func (n *NotificationService) handleTicketAssigned(ctx context.Context, event events.Event) error {
	n.logger.Info("TicketAssigned", zap.String("ticket_id", event.TicketID), zap.Any("payload", event.Payload))
	n.sendWebhookNotificationStub(ctx, event)
	return nil
}

func (n *NotificationService) handleTicketMessageAdded(ctx context.Context, event events.Event) error {
	n.logger.Info("TicketMessageAdded", zap.String("ticket_id", event.TicketID), zap.Any("payload", event.Payload))
	n.sendEmailNotificationStub(ctx, event)
	return nil
}

func (n *NotificationService) sendEmailNotificationStub(ctx context.Context, event events.Event) {
	if strings.TrimSpace(n.cfg.EmailFrom) == "" {
		return
	}
	n.logger.Debug("sendEmailNotificationStub",
		zap.String("from", n.cfg.EmailFrom),
		zap.String("ticket_id", event.TicketID),
		zap.String("event_type", string(event.Type)))
}

func (n *NotificationService) sendWebhookNotificationStub(ctx context.Context, event events.Event) {
	if strings.TrimSpace(n.cfg.WebhookURL) == "" {
		return
	}
	n.logger.Debug("sendWebhookNotificationStub",
		zap.String("url", n.cfg.WebhookURL),
		zap.String("ticket_id", event.TicketID),
		zap.String("event_type", string(event.Type)))
}
