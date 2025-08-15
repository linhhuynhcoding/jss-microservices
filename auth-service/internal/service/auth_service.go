package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/linhhuynhcoding/jss-microservices/auth-service/internal/domain"
	"github.com/linhhuynhcoding/jss-microservices/auth-service/internal/dto"
	"github.com/linhhuynhcoding/jss-microservices/auth-service/internal/repository"
	"github.com/linhhuynhcoding/jss-microservices/auth-service/pkg/queue"
	"github.com/linhhuynhcoding/jss-microservices/auth-service/pkg/token"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)


var (
    ErrEmailAlreadyExists      = errors.New("email already exists")
    ErrEmailNotFound           = errors.New("email not found")
    ErrInvalidOTP              = errors.New("invalid OTP")
    ErrOTPExpired              = errors.New("OTP expired")
    ErrFailedToSendOTP         = errors.New("failed to send OTP")
    ErrInvalidCredentials      = errors.New("invalid credentials")
    ErrUnauthorizedAccess      = errors.New("unauthorized access")
    ErrRefreshTokenAlreadyUsed = errors.New("refresh token already used")
)

type AuthService struct {
    userRepo              *repository.UserRepository
    roleRepo              *repository.RoleRepository
    codeRepo              *repository.VerificationCodeRepository
    deviceRepo            *repository.DeviceRepository
    refreshTokenRepo      *repository.RefreshTokenRepository
    hashingSvc            HashingService
    tokenSvc              *token.TokenService
    emailSvc              EmailService
    mq                    *queue.Publisher
    log                   *zap.Logger
    otpExpires            time.Duration
}

func NewAuthService(
    userRepo *repository.UserRepository,
    roleRepo *repository.RoleRepository,
    codeRepo *repository.VerificationCodeRepository,
    deviceRepo *repository.DeviceRepository,
    refreshTokenRepo *repository.RefreshTokenRepository,
    hashingSvc HashingService,
    tokenSvc *token.TokenService,
    emailSvc EmailService,
    mq *queue.Publisher,
    otpExpires time.Duration,
    log *zap.Logger,
) *AuthService {
    return &AuthService{
        userRepo:         userRepo,
        roleRepo:         roleRepo,
        codeRepo:         codeRepo,
        deviceRepo:       deviceRepo,
        refreshTokenRepo: refreshTokenRepo,
        hashingSvc:       hashingSvc,
        tokenSvc:         tokenSvc,
        emailSvc:         emailSvc,
        mq:               mq,
        log:              log,
        otpExpires:       otpExpires,
    }
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*domain.User, error) {
    // 1. Validate OTP
    if err := s.codeRepo.Validate(ctx, req.Email, req.Code, domain.VerificationTypeRegister); err != nil {
        return nil, ErrInvalidOTP
    }
    // 2. Check existing user
    if u, _ := s.userRepo.FindByEmail(ctx, req.Email); u != nil {
        return nil, ErrEmailAlreadyExists
    }
    // 3. Hash password
    hashed, err := s.hashingSvc.Hash(req.Password)
    if err != nil {
        s.log.Error("hash error", zap.Error(err))
        return nil, err
    }
    // 4. Create user
    roleID, _ := s.roleRepo.GetClientRoleID(ctx)
    user := &domain.User{
        Username: req.Username,
        Email:    req.Email,
        Password: hashed,
        RoleID:   roleID,
    }
    created, err := s.userRepo.CreateUser(ctx, user)
    if mongo.IsDuplicateKeyError(err) {
        return nil, ErrEmailAlreadyExists
    } else if err != nil {
        return nil, err
    }
    // 5. Delete OTP
    s.codeRepo.Delete(ctx, req.Email, req.Code, domain.VerificationTypeRegister)
    // 6. Publish event
    if s.mq != nil {
        event := map[string]interface{}{
            "type":     "user.created",
            "userId":   created.ID.Hex(),
            "email":    created.Email,
            "username": created.Username,
            "roleId":   created.RoleID.Hex(),
            "time":     time.Now(),
        }
        s.mq.Publish("user.created", event)
    }
    return created, nil
}

func (s *AuthService) SendOTP(ctx context.Context, req dto.SendOTPRequest) error {
    // Check user existence based on type
    u, _ := s.userRepo.FindByEmail(ctx, req.Email)
    if req.Type == domain.VerificationTypeRegister && u != nil {
        return ErrEmailAlreadyExists
    }
    if req.Type == domain.VerificationTypeForgotPassword && u == nil {
        return ErrEmailNotFound
    }
    // Generate and store OTP
    code := generateOTP()
    expires := time.Now().Add(s.otpExpires)
    if err := s.codeRepo.Create(ctx, &domain.VerificationCode{
        Email:     req.Email,
        Code:      code,
        Type:      req.Type,
        ExpiresAt: expires,
    }); err != nil {
        return err
    }
    // Send via email
    if err := s.emailSvc.SendOTP(req.Email, code); err != nil {
        return ErrFailedToSendOTP
    }
    return nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error) {
    user, err := s.userRepo.FindWithRoleByEmail(ctx, req.Email)
    if err != nil || user == nil {
        return nil, ErrEmailNotFound
    }
    ok, err := s.hashingSvc.Compare(req.Password, user.Password)
    if err != nil || !ok {
        return nil, ErrInvalidCredentials
    }
    device := &domain.Device{
        UserID:    user.ID,
        UserAgent: req.UserAgent,
        IP:        req.Ip,
        IsActive:  true,
    }
    created, err := s.deviceRepo.CreateDevice(ctx, device)
    if err != nil {
        return nil, err
    }
    return s.generateTokens(ctx, user.ID, created.ID, user.RoleID, user.Role.Name)
}

func (s *AuthService) RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (*dto.TokenResponse, error) {
    info, err := s.tokenSvc.VerifyRefresh(req.RefreshToken)
    if err != nil {
        return nil, ErrUnauthorizedAccess
    }
    stored, err := s.refreshTokenRepo.FindWithUser(ctx, req.RefreshToken)
    if err != nil || stored == nil {
        return nil, ErrRefreshTokenAlreadyUsed
    }
    // Update device
    s.deviceRepo.UpdateDevice(ctx, stored.DeviceID, map[string]interface{}{
        "userAgent": req.UserAgent,
        "ip":        req.Ip,
    })
    // Delete old token
    s.refreshTokenRepo.Delete(ctx, req.RefreshToken)
    return s.generateTokens(ctx, info.UserID, stored.DeviceID, stored.User.RoleID, stored.User.Role.Name)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
    // Delete token and mark device inactive
    deleted, err := s.refreshTokenRepo.Delete(ctx, refreshToken)
    if err != nil || deleted == nil {
        return ErrRefreshTokenAlreadyUsed
    }
    return s.deviceRepo.UpdateDevice(ctx, deleted.DeviceID, map[string]interface{}{"isActive": false})
}

func (s *AuthService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error {
    user, err := s.userRepo.FindByEmail(ctx, req.Email)
    if err != nil || user == nil {
        return ErrEmailNotFound
    }
    if err := s.codeRepo.Validate(ctx, req.Email, req.Code, domain.VerificationTypeForgotPassword); err != nil {
        return ErrInvalidOTP
    }
    hashed, err := s.hashingSvc.Hash(req.NewPassword)
    if err != nil {
        return err
    }
    s.userRepo.UpdatePassword(ctx, user.ID, hashed)
    s.codeRepo.Delete(ctx, req.Email, req.Code, domain.VerificationTypeForgotPassword)
    return nil
}

func (s *AuthService) GetAuthorizationUrl(ctx context.Context, req dto.GoogleAuthState) (string, error) {
    return s.emailSvc.GenerateGoogleURL(ctx, req.UserAgent, req.Ip)
}

func (s *AuthService) GoogleCallback(ctx context.Context, req dto.GoogleCallbackRequest) (*dto.TokenResponse, error) {
    tokens, err := s.emailSvc.ProcessGoogleCallback(ctx, req.Code, req.State)
    if err != nil {
        return nil, err
    }
    return &dto.TokenResponse{
        AccessToken:  tokens.AccessToken,
        RefreshToken: tokens.RefreshToken,
    }, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, token string) (bool, string, string) {
    claims, err := s.tokenSvc.VerifyAccess(token)
    if err != nil {
        return false, "", ""
    }
    return true, claims.UserID, claims.RoleName
}

func (s *AuthService) generateTokens(ctx context.Context, userID, deviceID primitive.ObjectID, roleID primitive.ObjectID, roleName string) (*dto.TokenResponse, error) {
    at, err := s.tokenSvc.SignAccess(ctx, userID, deviceID, roleID, roleName)
    if err != nil {
        return nil, err
    }
    rt, err := s.tokenSvc.SignRefresh(ctx, userID)
    if err != nil {
        return nil, err
    }
    decoded, _ := s.tokenSvc.DecodeRefresh(rt)
    s.refreshTokenRepo.Create(ctx, &domain.RefreshToken{
        Token:     rt,
        UserID:    userID,
        DeviceID:  deviceID,
        ExpiresAt: decoded.ExpiresAt,
    })
    return &dto.TokenResponse{AccessToken: at, RefreshToken: rt}, nil
}

// generateOTP returns a 6-digit numeric OTP
func generateOTP() string {
    return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}
