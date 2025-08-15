package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Role struct {
    ID   primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    Name string             `bson:"name" json:"name"`
}

type User struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    Username  string             `bson:"username" json:"username"`
    Email     string             `bson:"email" json:"email"`
    Password  string             `bson:"password" json:"-"`
    RoleID    primitive.ObjectID `bson:"roleId" json:"roleId"`
    Role      *Role              `bson:"role,omitempty" json:"role,omitempty"`
    IsActive  bool               `bson:"isActive" json:"isActive"`
    CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
    UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type UserEvent struct {
    Type string      `json:"type"`
    User interface{} `json:"user"`
}
