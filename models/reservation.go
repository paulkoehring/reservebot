package models

import (
	"time"
)

type Reservation struct {
	User     *User
	Resource *Resource
	Time     time.Time
        Duration time.Duration
}

func (r Reservation) EndTime() time.Time {
    return r.Time.Add(r.Duration)
}

func (r Reservation) TimeRemaining() time.Duration {
    return time.Until(r.EndTime())
}
