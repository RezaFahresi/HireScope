package repository

import (
	"context"
	"errors"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user with this email already exists")
)

// UserRepository handles database operations for User entities.
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Count(ctx context.Context) (int64, error)
	List(ctx context.Context) ([]model.User, error)
}

type gormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository backed by GORM.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

func (r *gormUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	normalized := model.NormalizeEmail(email)
	err := r.db.WithContext(ctx).Where("email = ?", normalized).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *gormUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *gormUserRepository) Create(ctx context.Context, user *model.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrUserAlreadyExists
		}
		return err
	}
	return nil
}

func (r *gormUserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error
	return count, err
}

func (r *gormUserRepository) List(ctx context.Context) ([]model.User, error) {
	var users []model.User
	err := r.db.WithContext(ctx).Order("name ASC").Find(&users).Error
	return users, err
}

