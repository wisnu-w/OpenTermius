package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/opentermius/opentermius/internal/domain"
	"github.com/opentermius/opentermius/internal/ports"
	"golang.org/x/crypto/bcrypt"
)

type AuthUseCase struct {
	userRepo  ports.UserRepository
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewAuthUseCase(userRepo ports.UserRepository, jwtSecret string, tokenTTL time.Duration) *AuthUseCase {
	return &AuthUseCase{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

func (a *AuthUseCase) Register(ctx context.Context, email, password, name string) (*domain.User, string, error) {
	if email == "" || password == "" || name == "" {
		return nil, "", domain.ErrInvalidInput
	}

	existing, err := a.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, "", domain.ErrConflict
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := a.userRepo.Create(ctx, user); err != nil {
		return nil, "", err
	}

	token, err := a.generateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (a *AuthUseCase) Login(ctx context.Context, email, password string) (*domain.User, string, error) {
	user, err := a.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", domain.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", domain.ErrUnauthorized
	}

	token, err := a.generateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (a *AuthUseCase) ValidateToken(ctx context.Context, tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, domain.ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, domain.ErrUnauthorized
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, domain.ErrUnauthorized
	}

	return uuid.Parse(sub)
}

func (a *AuthUseCase) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return a.userRepo.GetByID(ctx, userID)
}

func (a *AuthUseCase) ListUsers(ctx context.Context) ([]domain.User, error) {
	return a.userRepo.ListAll(ctx)
}

func (a *AuthUseCase) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return a.userRepo.Delete(ctx, userID)
}

func (a *AuthUseCase) generateToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(a.tokenTTL).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}
