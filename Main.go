package main

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var jwtKey = []byte("supersecretkey")

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `json:"username"`
	Password string             `json:"password"`
}

type Lesson struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title   string             `json:"title"`
	Content string             `json:"content"`
}

type Progress struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `json:"user_id"`
	LessonID  primitive.ObjectID `json:"lesson_id"`
	Completed bool               `json:"completed"`
}

type Claims struct {
	Username string `json:"username"`
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

func register(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	collection := client.Database("learning").Collection("users")
	_, err := collection.InsertOne(context.TODO(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not register user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func login(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	collection := client.Database("learning").Collection("users")
	var dbUser User
	err := collection.FindOne(context.TODO(), bson.M{"username": user.Username}).Decode(&dbUser)
	if err != nil || dbUser.Password != user.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}
func getUsers(c *gin.Context) {
	collection := client.Database("learning").Collection("users")

	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch users"})
		return
	}

	var users []User
	if err = cursor.All(context.TODO(), &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func getLessons(c *gin.Context) {
	collection := client.Database("learning").Collection("lessons")
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch lessons"})
		return
	}
	var lessons []Lesson
	if err = cursor.All(context.TODO(), &lessons); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing lessons"})
		return
	}
	c.JSON(http.StatusOK, lessons)
}

func createLesson(c *gin.Context) {
	var lesson Lesson
	if err := c.ShouldBindJSON(&lesson); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	collection := client.Database("learning").Collection("lessons")
	_, err := collection.InsertOne(context.TODO(), lesson)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create lesson"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Lesson created successfully"})
}

func getProgress(c *gin.Context) {
	collection := client.Database("learning").Collection("progress")
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch progress"})
		return
	}
	var progress []Progress
	if err = cursor.All(context.TODO(), &progress); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing progress"})
		return
	}
	c.JSON(http.StatusOK, progress)
}

func updateProgress(c *gin.Context) {
	var progress Progress
	if err := c.ShouldBindJSON(&progress); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	collection := client.Database("learning").Collection("progress")
	_, err := collection.InsertOne(context.TODO(), progress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update progress"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Progress updated"})
}
