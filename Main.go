package main

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/golang-jwt/jwt/v5"

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
	ID       string `bson:"_id" json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin" default:"false"`
}

type Lesson struct {
	ID      string `bson:"_id" json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Progress struct {
	ID            string  `bson:"_id" json:"id"`
	UserID        string  `bson:"user_id" json:"user_id"`
	Course        string  `bson:"course" json:"course"`
	VideosDone    int     `bson:"videos_done" json:"videos_done"`
	QuizzesDone   int     `bson:"quizzes_done" json:"quizzes_done"`
	Percentage    float64 `bson:"percentage" json:"percentage"`
	LanguageLevel string  `bson:"language_level" json:"language_level"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type QuizProgress struct {
	ID             string  `bson:"_id" json:"id"`
	UserID         string  `bson:"user_id" json:"user_id"`
	Course         string  `bson:"course" json:"course"`
	QuizID         string  `bson:"quiz_id" json:"quiz_id"`
	Score          float64 `bson:"score" json:"score"`
	CorrectAnswers int     `bson:"correct_answers" json:"correct_answers"`
	TotalQuestions int     `bson:"total_questions" json:"total_questions"`
	UserAnswers    []int   `bson:"user_answers" json:"user_answers"`
	Completed      bool    `bson:"completed" json:"completed"`
	Timestamp      string  `bson:"timestamp" json:"timestamp"`
}

type UserProgress struct {
	CourseType string  `bson:"course_type" json:"course_type"`
	Score      float64 `bson:"score" json:"score"`
	LastTest   string  `bson:"last_test" json:"last_test"`
}

type VideoProgress struct {
	ID        string `bson:"_id" json:"id"`
	UserID    string `bson:"user_id" json:"user_id"`
	Course    string `bson:"course" json:"course"`
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

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	initDatabase()

	r := gin.Default()

	r.Static("/static", "./frontend")

	r.LoadHTMLGlob("frontend/*.html")

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

	r.GET("/videokz", func(c *gin.Context) {
		c.HTML(http.StatusOK, "videokz.html", nil)
	})

	r.GET("/quizkz", func(c *gin.Context) {
		c.HTML(http.StatusOK, "quizkz.html", nil)
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

	r.POST("/quiz/submit", submitQuizResult)
	r.GET("/user/progress/:userId", getUserProgress)

	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.html", nil)

	})

	r.DELETE("/quiz/delete/:userId/:course", deleteQuizResults)

	r.POST("/video/progress", updateVideoProgress)
	r.GET("/course/progress/:userId/:course", getCourseProgress)

	r.GET("/quiz/progress/:userId/:quizId", getQuizProgress)
	r.GET("/quizzes/completed/:userId/:course", getCompletedQuizzes)

	r.GET("/videos/completed/:userId/:course", getCompletedVideos)

	r.GET("/update-progress/:userId/:course", updateUserProgress)

	r.POST("/level-test/submit", submitLevelTest)

	r.Run(":8000")
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
	user.IsAdmin = false

	_, err = collection.InsertOne(context.TODO(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Registration successful",
		"is_admin": user.IsAdmin,
	})
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
		UserID         string  `json:"user_id"`
		Course         string  `json:"course"`
		QuizID         string  `json:"quiz_id"`
		Score          float64 `json:"score"`
		CorrectAnswers int     `json:"correct_answers"`
		TotalQuestions int     `json:"total_questions"`
		UserAnswers    []int   `json:"user_answers"`
		Completed      bool    `json:"completed"`
	}

	if err := c.ShouldBindJSON(&submission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	quizCollection := client.Database("learning").Collection("quiz_progress")

	var existingProgress QuizProgress
	err := quizCollection.FindOne(context.TODO(), bson.M{
		"user_id": submission.UserID,
		"quiz_id": submission.QuizID,
		"course":  submission.Course,
	}).Decode(&existingProgress)

	if err != nil && err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing progress"})
		return
	}

	quizProgress := QuizProgress{
		UserID:         submission.UserID,
		Course:         submission.Course,
		QuizID:         submission.QuizID,
		Score:          submission.Score,
		CorrectAnswers: submission.CorrectAnswers,
		TotalQuestions: submission.TotalQuestions,
		UserAnswers:    submission.UserAnswers,
		Completed:      true,
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	if err == mongo.ErrNoDocuments {
		quizProgress.ID = nextID("quiz_progress")
		_, err = quizCollection.InsertOne(context.TODO(), quizProgress)
	} else {
		_, err = quizCollection.UpdateOne(
			context.TODO(),
			bson.M{
				"user_id": submission.UserID,
				"quiz_id": submission.QuizID,
				"course":  submission.Course,
			},
			bson.M{
				"$set": bson.M{
					"score":           submission.Score,
					"correct_answers": submission.CorrectAnswers,
					"user_answers":    submission.UserAnswers,
					"completed":       true,
					"timestamp":       time.Now().Format(time.RFC3339),
				},
			},
		)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save quiz progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Quiz progress saved successfully",
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

func deleteQuizResults(c *gin.Context) {
	userID := c.Param("userId")
	course := c.Param("course")

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

func updateVideoProgress(c *gin.Context) {
	var submission struct {
		UserID    string `json:"user_id"`
		Course    string `json:"course"`
		VideoID   string `json:"video_id"`
		Completed bool   `json:"completed"`
	}

	if err := c.ShouldBindJSON(&submission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	collection := client.Database("learning").Collection("video_progress")

	var existingProgress VideoProgress
	err := collection.FindOne(context.TODO(), bson.M{
		"user_id":  submission.UserID,
		"video_id": submission.VideoID,
		"course":   submission.Course,
	}).Decode(&existingProgress)

	videoProgress := VideoProgress{
		UserID:    submission.UserID,
		Course:    submission.Course,
		VideoID:   submission.VideoID,
		Completed: true,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err == mongo.ErrNoDocuments {
		videoProgress.ID = nextID("video_progress")
		_, err = collection.InsertOne(context.TODO(), videoProgress)
	} else {
		_, err = collection.UpdateOne(
			context.TODO(),
			bson.M{
				"user_id":  submission.UserID,
				"video_id": submission.VideoID,
				"course":   submission.Course,
			},
			bson.M{
				"$set": bson.M{
					"completed": true,
					"timestamp": time.Now().Format(time.RFC3339),
				},
			},
		)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video progress"})
		return
	}

	progress, err := calculateProgress(submission.UserID, submission.Course)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"message":  "Video progress saved",
		"progress": progress,
	})
}

func getCourseProgress(c *gin.Context) {
	userID := c.Param("userId")
	course := c.Param("course")

	quizCollection := client.Database("learning").Collection("quiz_progress")
	completedQuizzes, err := quizCollection.CountDocuments(context.TODO(), bson.M{
		"user_id":   userID,
		"course":    course,
		"completed": true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count completed quizzes"})
		return
	}

	videoCollection := client.Database("learning").Collection("video_progress")
	completedVideos, err := videoCollection.CountDocuments(context.TODO(), bson.M{
		"user_id":   userID,
		"course":    course,
		"completed": true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count completed videos"})
		return
	}

	progress := CourseProgress{
		UserID:      userID,
		Course:      course,
		QuizDone:    int(completedQuizzes),
		VideosDone:  int(completedVideos),
		TotalQuiz:   5,
		TotalVideos: 10,
	}

	totalItems := progress.TotalQuiz + progress.TotalVideos
	completedItems := progress.QuizDone + progress.VideosDone
	progress.Percentage = (float64(completedItems) / float64(totalItems)) * 100

	c.JSON(http.StatusOK, progress)
}

func initDatabase() {
	ctx := context.TODO()
	db := client.Database("learning")

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

func calculateProgress(userID, course string) (Progress, error) {
	var progress Progress

	collection := client.Database("learning").Collection("progress")
	err := collection.FindOne(context.TODO(), bson.M{
		"user_id": userID,
		"course":  course,
	}).Decode(&progress)

	if err == mongo.ErrNoDocuments {
		progress.ID = nextID("progress")
		progress.UserID = userID
		progress.Course = course
	} else if err != nil {
		return progress, err
	}

	videoCollection := client.Database("learning").Collection("video_progress")
	completedVideos, err := videoCollection.CountDocuments(context.TODO(), bson.M{
		"user_id":   userID,
		"course":    course,
		"completed": true,
	})
	if err != nil {
		return progress, err
	}

	quizCollection := client.Database("learning").Collection("quiz_progress")
	completedQuizzes, err := quizCollection.CountDocuments(context.TODO(), bson.M{
		"user_id":   userID,
		"course":    course,
		"completed": true,
	})
	if err != nil {
		return progress, err
	}

	progress.VideosDone = int(completedVideos)
	progress.QuizzesDone = int(completedQuizzes)

	totalVideos := 10
	totalTests := 5
	totalItems := totalVideos + totalTests
	completedItems := progress.VideosDone + progress.QuizzesDone
	progress.Percentage = (float64(completedItems) / float64(totalItems)) * 100

	filter := bson.M{
		"user_id": userID,
		"course":  course,
	}
	update := bson.M{
		"$set": bson.M{
			"_id":          progress.ID,
			"videos_done":  progress.VideosDone,
			"quizzes_done": progress.QuizzesDone,
			"percentage":   progress.Percentage,
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err = collection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		return progress, err
	}

	return progress, nil
}

func getQuizProgress(c *gin.Context) {
	userID := c.Param("userId")
	quizID := c.Param("quizId")

	var progress QuizProgress
	collection := client.Database("learning").Collection("quiz_progress")
	err := collection.FindOne(context.TODO(), bson.M{
		"user_id": userID,
		"quiz_id": quizID,
	}).Decode(&progress)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "No progress found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

func getCompletedQuizzes(c *gin.Context) {
	userID := c.Param("userId")
	course := c.Param("course")

	collection := client.Database("learning").Collection("quiz_progress")
	cursor, err := collection.Find(context.TODO(), bson.M{
		"user_id":   userID,
		"course":    course,
		"completed": true,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(context.TODO())

	var completedQuizzes []QuizProgress
	if err = cursor.All(context.TODO(), &completedQuizzes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing results"})
		return
	}

	quizIds := make([]string, 0)
	for _, quiz := range completedQuizzes {
		quizIds = append(quizIds, quiz.QuizID)
	}

	c.JSON(http.StatusOK, quizIds)
}

func getCompletedVideos(c *gin.Context) {
	userID := c.Param("userId")
	course := c.Param("course")

	collection := client.Database("learning").Collection("video_progress")
	cursor, err := collection.Find(context.TODO(), bson.M{
		"user_id":   userID,
		"course":    course,
		"completed": true,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(context.TODO())

	var completedVideos []VideoProgress
	if err = cursor.All(context.TODO(), &completedVideos); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing results"})
		return
	}

	videoIds := make([]string, 0)
	for _, video := range completedVideos {
		videoIds = append(videoIds, video.VideoID)
	}

	c.JSON(http.StatusOK, videoIds)
}

func updateUserProgress(c *gin.Context) {
	userID := c.Param("userId")
	course := c.Param("course")

	progress, err := calculateProgress(userID, course)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

func submitLevelTest(c *gin.Context) {
	var submission struct {
		UserID string  `json:"user_id"`
		Course string  `json:"course"`
		Score  float64 `json:"score"`
		Level  string  `json:"level"`
	}

	if err := c.ShouldBindJSON(&submission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	collection := client.Database("learning").Collection("progress")
	filter := bson.M{
		"user_id": submission.UserID,
		"course":  submission.Course,
	}
	update := bson.M{
		"$set": bson.M{
			"language_level": submission.Level,
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update language level"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Language level updated successfully",
	})
}
