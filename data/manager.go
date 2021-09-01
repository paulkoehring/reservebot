package data

import (
	"github.com/ameliagapin/reservebot/models"
        "time"
)

type Manager interface {
	GetAllUsersInQueues() []*models.User
	GetPosition(u *models.User, name string, env string) (int, error)
	GetQueueForResource(name string, env string) (*models.Queue, error)
	GetQueues() []*models.Queue
	GetQueuesForEnv(env string) map[string]*models.Queue
	GetReservation(u *models.User, name string, env string) *models.Reservation
	GetReservationForResource(name string, env string) (*models.Reservation, error)
	GetResource(name string, env string, create bool) *models.Resource
	GetResources() []*models.Resource
	GetResourcesForEnv(env string) []*models.Resource
	Remove(u *models.User, name string, env string) error
	RemoveEnv(name string, env string) error
	RemoveResource(name string, env string) error
	Reserve(u *models.User, name string, env string, dur time.Duration) error
	ClearQueueForResource(name, env string) error
}
