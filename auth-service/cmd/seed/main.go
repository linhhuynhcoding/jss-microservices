package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/linhhuynhcoding/jss-microservices/auth-service/internal/domain"
	"github.com/linhhuynhcoding/jss-microservices/auth-service/internal/repository"
	"github.com/linhhuynhcoding/jss-microservices/auth-service/pkg/database"
	"github.com/linhhuynhcoding/jss-microservices/auth-service/pkg/hashing"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func main() {
	logg := zap.NewExample()
	defer logg.Sync()

	db := database.Connect( logg)
	ctx := context.Background()

	roleRepo := repository.NewRoleRepository(db, logg)
    log.Println("Seeding default roles...")
    roleRepo.SeedDefaultRoles()
    log.Println("Default roles seeded.")

	userRepo := repository.NewUserRepository(db, logg)
	hashingSvc := hashing.NewHashingService()

	adminEmail := os.Getenv("DEFAULT_ADMIN_EMAIL")
	adminPass := os.Getenv("DEFAULT_ADMIN_PASSWORD")

	existing, _ := userRepo.FindByEmail(ctx, adminEmail)
	if existing != nil {
		log.Println("Admin already exists:", adminEmail)
		return
	}

	hashed, _ := hashingSvc.Hash(adminPass)
	adminRoleID, err := roleRepo.GetRoleIDByName(ctx, "admin")
	if err != nil {
		log.Fatalf("Get admin role ID failed: %v", err)
	}

	_, err = userRepo.CreateUser(ctx, &domain.User{
		ID:        primitive.NewObjectID(),
		Username:  "admin",
		Email:     adminEmail,
		Password:  hashed,
		RoleID:    adminRoleID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		log.Fatalf("Create admin user failed: %v", err)
	}

	log.Println("Seeded default admin:", adminEmail)
}
