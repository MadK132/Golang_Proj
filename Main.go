package main

import (
	"context"
	"log"
	_ "net/http"
	_ "time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var jwtKey = []byte("supersecretkey")

type User struct {
	ID       primitive.ObjectID bson:"_id,omitempty" json:"id"
	Username string             json:"username"
	Password string             json:"password"
}

type Lesson struct {
	ID      primitive.ObjectID bson:"_id,omitempty" json:"id"
	Title   string             json:"title"
	Content string             json:"content"
}

type Progress struct {
	ID        primitive.ObjectID bson:"_id,omitempty" json:"id"
	UserID    primitive.ObjectID json:"user_id"
	LessonID  primitive.ObjectID json:"lesson_id"
	Completed bool               json:"completed"
}

type Claims struct {
	Username string json:"username"
	jwt.RegisteredClaims
}

func main() {
	var err error
	client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017/learning"))
	if err != nil {
		log.Fatal(err)
	}
	r := gin.Default()
	r.POST("/register", register)
	r.POST("/login", login)
	r.GET("/lessons", getLessons)
	r.POST("/lessons", createLesson)
	r.GET("/progress", getProgress)
	r.POST("/progress", updateProgress)
	r.GET("/users", getUsers)
	r.Run(":8080")
}