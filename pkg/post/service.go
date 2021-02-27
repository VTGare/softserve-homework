package post

import (
	"context"

	"go.uber.org/zap"
)

//Service is a Post service interface which contains all business logic.
type Service interface {
	Create(context.Context, *Post) (int64, error)
	FindOne(context.Context, int64) (*Post, error)
	FindMany(context.Context, *SearchFilter) ([]*Post, error)
	Remove(context.Context, int64) (bool, error)
	Logger() *zap.SugaredLogger
	Count(context.Context) (map[string]int, error)
}
