package facade

import "go-notification-sse/internal/port/inbound"

type UserApplicationService struct{}

var _ inbound.UserUseCase = (*UserApplicationService)(nil)

func NewUserApplicationService() *UserApplicationService {
	return &UserApplicationService{}
}
