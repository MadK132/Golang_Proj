package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client *mongo.Client
	jwtKey = []byte("supersecretkey")
	mu     sync.Mutex
)

type User struct {
	ID       string         `bson:"_id" json:"id"`
	Username string         `json:"username"`
	Email    string         `json:"email"`
	Password string         `json:"password"`
	Progress []UserProgress `json:"progress,omitempty"`
}

type Lesson struct {
	ID      string `bson:"_id" json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Progress struct {
	ID          string  `bson:"_id" json:"id"`
	UserID      string  `bson:"user_id" json:"user_id"`
	Course      string  `bson:"course" json:"course"`
	VideosDone  int     `bson:"videos_done" json:"videos_done"`
	QuizzesDone int     `bson:"quizzes_done" json:"quizzes_done"`
	LevelTest   string  `bson:"level_test" json:"level_test"`
	Percentage  float64 `bson:"percentage" json:"percentage"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Login request structure
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Add these new structures
type QuizProgress struct {
	ID        string  `bson:"_id" json:"id"`
	UserID    string  `bson:"user_id" json:"user_id"`
	Course    string  `bson:"course" json:"course"`
	Score     float64 `bson:"score" json:"score"`
	Level     string  `bson:"level" json:"level"`
	Timestamp string  `bson:"timestamp" json:"timestamp"`
}

// Add new struct for user progress
type UserProgress struct {
	CourseType string  `bson:"course_type" json:"course_type"` // "russian" or "kazakh"
	Level      string  `bson:"level" json:"level"`             // A1, A2, B1, B2, C1
	Score      float64 `bson:"score" json:"score"`
	LastTest   string  `bson:"last_test" json:"last_test"` // timestamp
}

// Add these new structs
type VideoProgress struct {
	ID        string `bson:"_id" json:"id"`
	UserID    string `bson:"user_id" json:"user_id"`
	Course    string `bson:"course" json:"course"` // "russian" or "kazakh"
	VideoID   string `bson:"video_id" json:"video_id"`
	Completed bool   `bson:"completed" json:"completed"`
	Timestamp string `bson:"timestamp" json:"timestamp"`
}

type CourseProgress struct {
	UserID      string  `json:"user_id"`
	Course      string  `json:"course"`
	QuizDone    int     `json:"quiz_done"`
	VideosDone  int     `json:"videos_done"`
	TotalQuiz   int     `json:"total_quiz"`
	TotalVideos int     `json:"total_videos"`
	Percentage  float64 `json:"percentage"`
}

func main() {
	var err error
	client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017/learning"))
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	initDatabase()

	r := gin.Default()

	// Раздача CSS, JS и изображений
	r.Static("/static", "./frontend")

	// Подключаем все HTML-файлы
	r.LoadHTMLGlob("frontend/*.html")

	// Определяем маршруты для всех страниц
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.GET("/contact", func(c *gin.Context) {
		c.HTML(http.StatusOK, "contact.html", nil)
	})

	r.GET("/courses", func(c *gin.Context) {
		c.HTML(http.StatusOK, "courses.html", nil)
	})

	r.GET("/general", func(c *gin.Context) {
		c.HTML(http.StatusOK, "General.html", nil)
	})

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "Login.html", nil)
	})

	r.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "reg.html", nil)
	})

	r.GET("/test", func(c *gin.Context) {
		c.HTML(http.StatusOK, "Test.html", nil)
	})

	r.GET("/quiz", func(c *gin.Context) {
		c.HTML(http.StatusOK, "quiz.html", nil)
	})

	r.GET("/video", func(c *gin.Context) {
		c.HTML(http.StatusOK, "video.html", nil)
	})

	r.GET("/russian-quiz", func(c *gin.Context) {
		c.HTML(http.StatusOK, "russian_quiz.html", nil)
	})

	r.GET("/kazakh-quiz", func(c *gin.Context) {
		c.HTML(http.StatusOK, "kazakh_quiz.html", nil)
	})

	r.POST("/register", register)
	r.POST("/login", login)
	r.GET("/lessons", getLessons)
	r.POST("/lessons", createLesson)
	r.GET("/progress", getProgress)
	r.POST("/progress", updateProgress)
	r.GET("/users", getUsers)
	r.DELETE("/users/:username", deleteUser)
	r.PUT("/users/:username", updateUser)

	// Quiz routes
	r.POST("/quiz/submit", submitQuizResult)
	r.GET("/user/progress/:userId", getUserProgress)

	// Add this before r.Run(":8080")
	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.html", nil)
		// Or redirect to home page
		// c.Redirect(http.StatusFound, "/")
	})

	// Add this new route in main()
	r.DELETE("/quiz/delete/:userId/:course", deleteQuizResults)

	// Add these new routes in main()
	r.POST("/video/progress", updateVideoProgress)
	r.GET("/course/progress/:userId/:course", getCourseProgress)

	r.Run(":8080")
}

func nextID(collectionName string) string {
	mu.Lock()
	defer mu.Unlock()

	collection := client.Database("learning").Collection("counters")
	var result struct {
		Seq int `bson:"seq"`
	}

	filter := bson.M{"_id": collectionName}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	err := collection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	return strconv.Itoa(result.Seq)
}

func register(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	collection := client.Database("learning").Collection("users")

	var existingUser User
	err := collection.FindOne(context.TODO(), bson.M{"email": user.Email}).Decode(&existingUser)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	user.ID = nextID("users")
	_, err = collection.InsertOne(context.TODO(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Registration successful"})
}

func login(c *gin.Context) {
	var loginReq LoginRequest
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	collection := client.Database("learning").Collection("users")

	var user User
	err := collection.FindOne(context.TODO(), bson.M{
		"email":    loginReq.Email,
		"password": loginReq.Password,
	}).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Login successful",
		"username": user.Username,
		"email":    user.Email,
	})
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

	lesson.ID = nextID("lessons")
	collection := client.Database("learning").Collection("lessons")
	_, err := collection.InsertOne(context.TODO(), lesson)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create lesson"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lesson created successfully", "id": lesson.ID})
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

	progress.ID = nextID("progress")
	collection := client.Database("learning").Collection("progress")
	_, err := collection.InsertOne(context.TODO(), progress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Progress updated", "id": progress.ID})
}

func deleteUser(c *gin.Context) {
	username := c.Param("username")
	collection := client.Database("learning").Collection("users")

	_, err := collection.DeleteOne(context.TODO(), bson.M{"username": username})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func updateUser(c *gin.Context) {
	username := c.Param("username")
	var updateData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	collection := client.Database("learning").Collection("users")
	update := bson.M{
		"$set": bson.M{
			"email":    updateData.Email,
			"password": updateData.Password,
		},
	}

	_, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"username": username},
		update,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

func submitQuizResult(c *gin.Context) {
	var submission struct {
		UserID   string  `json:"user_id"`
		Course   string  `json:"course"`
		Score    float64 `json:"score"`
		Level    string  `json:"level"`
		Complete bool    `json:"complete"`
	}

	if err := c.ShouldBindJSON(&submission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// First, delete previous results
	quizCollection := client.Database("learning").Collection("quiz_progress")
	_, err := quizCollection.DeleteMany(
		context.TODO(),
		bson.M{
			"user_id": submission.UserID,
			"course":  submission.Course,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete previous results"})
		return
	}

	// Update user's progress by removing old results
	collection := client.Database("learning").Collection("users")
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"username": submission.UserID},
		bson.M{
			"$pull": bson.M{
				"progress": bson.M{
					"course_type": submission.Course,
				},
			},
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user progress"})
		return
	}

	// Now add new progress
	timestamp := time.Now().Format(time.RFC3339)
	progress := UserProgress{
		CourseType: submission.Course,
		Level:      submission.Level,
		Score:      submission.Score,
		LastTest:   timestamp,
	}

	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"username": submission.UserID},
		bson.M{
			"$push": bson.M{
				"progress": progress,
			},
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update progress"})
		return
	}

	// Save new quiz progress
	quizProgress := QuizProgress{
		ID:        nextID("quiz_progress"),
		UserID:    submission.UserID,
		Course:    submission.Course,
		Score:     submission.Score,
		Level:     submission.Level,
		Timestamp: timestamp,
	}

	_, err = quizCollection.InsertOne(context.TODO(), quizProgress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save quiz progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Progress saved successfully",
		"level":   submission.Level,
		"score":   submission.Score,
	})
}

func getUserProgress(c *gin.Context) {
	userID := c.Param("userId")
	course := c.Param("course")

	progress, err := calculateProgress(userID, course)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// Add new function to delete quiz results
func deleteQuizResults(c *gin.Context) {
	userID := c.Param("userId")
	course := c.Param("course")

	// Delete from quiz_progress collection
	quizCollection := client.Database("learning").Collection("quiz_progress")
	_, err := quizCollection.DeleteMany(
		context.TODO(),
		bson.M{
			"user_id": userID,
			"course":  course,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete quiz progress"})
		return
	}

	// Update user's progress array
	collection := client.Database("learning").Collection("users")
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"username": userID},
		bson.M{
			"$pull": bson.M{
				"progress": bson.M{
					"course_type": course,
				},
			},
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quiz results deleted successfully"})
}

// Add new function to update video progress
func updateVideoProgress(c *gin.Context) {
	var progress VideoProgress
	if err := c.ShouldBindJSON(&progress); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	progress.ID = nextID("video_progress")
	progress.Timestamp = time.Now().Format(time.RFC3339)

	collection := client.Database("learning").Collection("video_progress")
	_, err := collection.InsertOne(context.TODO(), progress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Video progress saved"})
}

// Add function to get overall course progress
func getCourseProgress(c *gin.Context) {
	userID := c.Param("userId")
	course := c.Param("course")

	// Get quiz progress
	quizCollection := client.Database("learning").Collection("quiz_progress")
	quizCount, err := quizCollection.CountDocuments(context.TODO(), bson.M{
		"user_id": userID,
		"course":  course,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get quiz progress"})
		return
	}

	// Get video progress
	videoCollection := client.Database("learning").Collection("video_progress")
	videoCount, err := videoCollection.CountDocuments(context.TODO(), bson.M{
		"user_id":   userID,
		"course":    course,
		"completed": true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get video progress"})
		return
	}

	// Calculate total progress
	totalQuiz := 5    // Update this based on your total number of quizzes
	totalVideos := 10 // Update this based on your total number of videos
	totalItems := totalQuiz + totalVideos
	completedItems := int(quizCount + videoCount)
	percentage := (float64(completedItems) / float64(totalItems)) * 100

	progress := CourseProgress{
		UserID:      userID,
		Course:      course,
		QuizDone:    int(quizCount),
		VideosDone:  int(videoCount),
		TotalQuiz:   totalQuiz,
		TotalVideos: totalVideos,
		Percentage:  percentage,
	}

	c.JSON(http.StatusOK, progress)
}

// Add this function to initialize collections
func initDatabase() {
	ctx := context.TODO()
	db := client.Database("learning")

	// Define indexes for collections
	collections := map[string][]mongo.IndexModel{
		"users": {
			{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		"quiz_progress": {
			{
				Keys: bson.D{
					{Key: "user_id", Value: 1},
					{Key: "course", Value: 1},
				},
			},
		},
		"video_progress": {
			{
				Keys: bson.D{
					{Key: "user_id", Value: 1},
					{Key: "course", Value: 1},
				},
			},
		},
		"progress": {
			{
				Keys: bson.D{
					{Key: "user_id", Value: 1},
					{Key: "course", Value: 1},
				},
				Options: options.Index().SetUnique(true),
			},
		},
	}

	// Create collections and indexes
	for collName, indexes := range collections {
		err := db.CreateCollection(ctx, collName)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			log.Printf("Error creating collection %s: %v", collName, err)
		}

		if len(indexes) > 0 {
			_, err = db.Collection(collName).Indexes().CreateMany(ctx, indexes)
			if err != nil {
				log.Printf("Error creating indexes for %s: %v", collName, err)
			}
		}
	}
}

// Update progress calculation function
func calculateProgress(userID, course string) (Progress, error) {
	var progress Progress
	progress.ID = nextID("progress")
	progress.UserID = userID
	progress.Course = course

	// Get video progress
	videoCollection := client.Database("learning").Collection("video_progress")
	videosCount, err := videoCollection.CountDocuments(context.TODO(), bson.M{
		"user_id":   userID,
		"course":    course,
		"completed": true,
	})
	if err != nil {
		return progress, err
	}

	// Get quiz progress (excluding level test)
	quizCollection := client.Database("learning").Collection("quiz_progress")
	quizzesCount, err := quizCollection.CountDocuments(context.TODO(), bson.M{
		"user_id":       userID,
		"course":        course,
		"is_level_test": bson.M{"$ne": true},
	})
	if err != nil {
		return progress, err
	}

	// Get level test result
	var levelTest QuizProgress
	err = quizCollection.FindOne(context.TODO(), bson.M{
		"user_id":       userID,
		"course":        course,
		"is_level_test": true,
	}).Decode(&levelTest)

	if err == nil {
		progress.LevelTest = levelTest.Level
	} else if err != mongo.ErrNoDocuments {
		return progress, err
	} else {
		progress.LevelTest = "Not determined"
	}

	progress.VideosDone = int(videosCount)
	progress.QuizzesDone = int(quizzesCount)

	// Calculate percentage
	totalVideos := 10
	totalQuizzes := 5
	totalItems := totalVideos + totalQuizzes
	completedItems := progress.VideosDone + progress.QuizzesDone
	progress.Percentage = (float64(completedItems) / float64(totalItems)) * 100

	// Save progress
	collection := client.Database("learning").Collection("progress")
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"user_id": userID, "course": course}
	update := bson.M{"$set": progress}

	_, err = collection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		return progress, err
	}

	return progress, nil
}
