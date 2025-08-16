package repository

import (
	"context"
	"time"

	"github.com/linhhuynhcoding/jss-microservices/auth-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type UserRepository struct {
    col *mongo.Collection
    log *zap.Logger
}

func NewUserRepository(db *mongo.Database, log *zap.Logger) *UserRepository {
    return &UserRepository{
        col: db.Collection("users"),
        log: log.With(zap.String("component", "UserRepository")),
    }
}

func (r *UserRepository) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
    user.ID = primitive.NewObjectID()
    user.CreatedAt = time.Now()
    user.UpdatedAt = time.Now()
    user.IsActive = true

    _, err := r.col.InsertOne(ctx, user)
    if err != nil {
        r.log.Error("CreateUser failed", zap.Error(err), zap.Any("user", user))
        return nil, err
    }
    r.log.Info("CreateUser succeeded", zap.String("userID", user.ID.Hex()), zap.String("email", user.Email))
    return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
    var user domain.User
    err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&user)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, nil
        }
        r.log.Error("FindByEmail failed", zap.String("email", email), zap.Error(err))
        return nil, err
    }
    return &user, nil
}

// FindWithRoleByEmail returns user with role populated by aggregation
func (r *UserRepository) FindWithRoleByEmail(ctx context.Context, email string) (*domain.User, error) {
    pipeline := mongo.Pipeline{
        {{"$match", bson.D{{"email", email}}}},
        {{"$lookup", bson.D{
            {"from", "roles"},
            {"localField", "roleId"},
            {"foreignField", "_id"},
            {"as", "role"},
        }}},
        {{"$unwind", "$role"}},
    }
    cursor, err := r.col.Aggregate(ctx, pipeline)
    if err != nil {
        r.log.Error("FindWithRoleByEmail aggregate failed", zap.String("email", email), zap.Error(err))
        return nil, err
    }
    defer cursor.Close(ctx)
    var users []domain.User
    if err := cursor.All(ctx, &users); err != nil {
        r.log.Error("FindWithRoleByEmail cursor decode failed", zap.Error(err))
        return nil, err
    }
    if len(users) == 0 {
        return nil, nil
    }
    return &users[0], nil
}

func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error) {
    var user domain.User
    err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, nil
        }
        r.log.Error("FindByID failed", zap.String("userID", id.Hex()), zap.Error(err))
        return nil, err
    }
    return &user, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID primitive.ObjectID, hashedPassword string) error {
    update := bson.M{"password": hashedPassword, "updatedAt": time.Now()}
    _, err := r.col.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": update})
    if err != nil {
        r.log.Error("UpdatePassword failed", zap.String("userID", userID.Hex()), zap.Error(err))
    }
    return err
}

func (r *UserRepository) UpdateUser(ctx context.Context, userID primitive.ObjectID, updates bson.M) error {
    updates["updatedAt"] = time.Now()
    _, err := r.col.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": updates})
    if err != nil {
        r.log.Error("UpdateUser failed", zap.String("userID", userID.Hex()), zap.Error(err))
    }
    return err
}

func (r *UserRepository) FindAll(ctx context.Context) ([]*domain.User, error) {
    cursor, err := r.col.Find(ctx, bson.M{})
    if err != nil {
        r.log.Error("FindAll failed", zap.Error(err))
        return nil, err
    }
    defer cursor.Close(ctx)

    var users []*domain.User
    if err := cursor.All(ctx, &users); err != nil {
        r.log.Error("FindAll decode failed", zap.Error(err))
        return nil, err
    }
    return users, nil
}

func (r *UserRepository) FindByRole(ctx context.Context, roleName string) ([]*domain.User, error) {
    // Aggregation to filter users by role name
    pipeline := mongo.Pipeline{
        {{"$lookup", bson.D{
            {"from", "roles"},
            {"localField", "roleId"},
            {"foreignField", "_id"},
            {"as", "role"},
        }}},
        {{"$unwind", "$role"}},
        {{"$match", bson.D{{"role.name", roleName}}}},
    }
    cursor, err := r.col.Aggregate(ctx, pipeline)
    if err != nil {
        r.log.Error("FindByRole aggregate failed", zap.String("role", roleName), zap.Error(err))
        return nil, err
    }
    defer cursor.Close(ctx)

    var users []*domain.User
    if err := cursor.All(ctx, &users); err != nil {
        r.log.Error("FindByRole decode failed", zap.Error(err))
        return nil, err
    }
    return users, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, userID primitive.ObjectID) error {
    _, err := r.col.DeleteOne(ctx, bson.M{"_id": userID})
    if err != nil {
        r.log.Error("DeleteUser failed", zap.String("userID", userID.Hex()), zap.Error(err))
    }
    return err
}
