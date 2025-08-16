package repository

import (
	"context"

	"github.com/linhhuynhcoding/jss-microservices/auth-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type RoleRepository struct {
    col *mongo.Collection
    log *zap.Logger
}

func NewRoleRepository(db *mongo.Database, log *zap.Logger) *RoleRepository {
    return &RoleRepository{
        col: db.Collection("roles"),
        log: log.With(zap.String("component", "RoleRepository")),
    }
}

func (r *RoleRepository) SeedDefaultRoles() {
    roles := []string{"admin", "manager", "staff"}
    for _, name := range roles {
        filter := bson.M{"name": name}
        update := bson.M{"$setOnInsert": bson.M{"name": name}}
        opts := options.Update().SetUpsert(true)
        res, err := r.col.UpdateOne(context.Background(), filter, update, opts)
        if err != nil {
            r.log.Error("SeedDefaultRoles update failed", zap.String("role", name), zap.Error(err))
        } else if res.UpsertedCount > 0 {
            r.log.Info("SeedDefaultRoles inserted new role", zap.String("role", name))
        } else {
            r.log.Debug("SeedDefaultRoles role already exists", zap.String("role", name))
        }
    }
}

func (r *RoleRepository) FindRoleByName(ctx context.Context, name string) (*domain.Role, error) {
    var role domain.Role
    err := r.col.FindOne(ctx, bson.M{"name": name}).Decode(&role)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, nil
        }
        r.log.Error("FindRoleByName failed", zap.String("role", name), zap.Error(err))
        return nil, err
    }
    return &role, nil
}

func (r *RoleRepository) FindRoleByID(ctx context.Context, id primitive.ObjectID) (*domain.Role, error) {
    var role domain.Role
    err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&role)
    if err == mongo.ErrNoDocuments {
        return nil, nil
    }
    return &role, err
}


func (r *RoleRepository) GetRoleIDByName(ctx context.Context, name string) (primitive.ObjectID, error) {
    role, err := r.FindRoleByName(ctx, name)
    if err != nil {
        return primitive.NilObjectID, err
    }
    if role == nil {
        return primitive.NilObjectID, mongo.ErrNoDocuments
    }
    return role.ID, nil
}
