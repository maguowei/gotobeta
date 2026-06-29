package user

import (
	"context"
	"time"
)

// Repository 定义用户聚合持久化端口。
type Repository interface {
	CreateUser(ctx context.Context, u *User) error
	FindUserByID(ctx context.Context, id int64) (*User, error)
	FindUserByEmail(ctx context.Context, normalizedEmail string) (*User, error)
	SaveUser(ctx context.Context, u *User) error
	UpdateUserLastLogin(ctx context.Context, userID int64, now time.Time) error
	CountLoginMethods(ctx context.Context, userID int64) (int, error)
}
