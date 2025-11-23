package worker

import (
	"github.com/spec-kit/ticket-service/internal/service"
)

// StartNotificationWorker registers notification handlers.
func StartNotificationWorker(notificationService *service.NotificationService) {
	if notificationService == nil {
		return
	}
	notificationService.RegisterHandlers()
}
