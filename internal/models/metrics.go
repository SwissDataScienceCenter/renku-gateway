package models

type MetricsClientInterface interface {
	UserLoggedIn(userId string) error
}
