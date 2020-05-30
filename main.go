package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

// Auth middleware
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)

		if session.Get("auth") != true {
			c.Redirect(302, "/login")
		} else {
			// c.Redirect(302, "/dashboard")
			c.Next()
		}
	}
}

// AlrAuth middleware
func AlrAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)

		if session.Get("auth") == true {
			c.Redirect(302, "/dashboard")
		} else {
			c.Next()
		}
	}
}

func emailExists(email string, collection *mongo.Collection) bool {
	filter := bson.M{"email": email}
	var result User
	err := collection.FindOne(context.TODO(), filter).Decode(&result)

	if err != nil {
		return false
	}

	return true
}

func hasher(password string) string {
	// Hash password
	hasher := sha1.New()
	hasher.Write([]byte(password))
	passwordHashHex := hasher.Sum(nil)

	passwordHash := hex.EncodeToString(passwordHashHex)
	return passwordHash
}

// User Modal
type User struct {
	FName    string
	LName    string
	Email    string
	Password string
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	mgdClientOpt := options.Client().ApplyURI("mongodb+srv://na-admin:rV9byQPoKeQBJpka@trashtalkpuddle-qpofb.mongodb.net/")
	mgdClient, err := mongo.Connect(context.TODO(), mgdClientOpt)

	usersCollection := mgdClient.Database("trashtalk").Collection("users")

	if err != nil {
		log.Fatal(err)
	}

	err = mgdClient.Ping(context.TODO(), nil)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")

	router := gin.New()

	store := cookie.NewStore([]byte("exmaple_secret_for_now_change_in_production!"))

	// Static files router
	router.Use(static.Serve("/static/", static.LocalFile("./public", true)))

	// Session
	router.Use(sessions.Sessions("userSession", store))

	router.GET("/", func(c *gin.Context) {
		c.File("./public/home.html")
	})

	router.GET("/dashboard", Auth(), func(c *gin.Context) {
		c.File("./public/dashboard.html")
	})

	router.GET("/signup", AlrAuth(), func(c *gin.Context) {
		c.File("./public/signup.html")
	})

	router.GET("/login", AlrAuth(), func(c *gin.Context) {
		c.File("./public/login.html")
	})

	// Logout route
	router.GET("/logout", Auth(), func(c *gin.Context) {
		session := sessions.Default(c)

		session.Set("auth", false)
		session.Save()

		c.Redirect(302, "/login")
	})

	router.GET("/info", Auth(), func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("auth") == true {
			c.JSON(200, gin.H{
				"auth":  session.Get("auth"),
				"name":  session.Get("name"),
				"email": session.Get("email"),
			})
		} else {
			c.JSON(200, gin.H{
				"auth": session.Get("auth"),
			})
		}
	})

	router.GET("/profile", Auth(), func(c *gin.Context) {
		c.File("./public/profile.html")
	})

	router.POST("/api/signup", AlrAuth(), func(c *gin.Context) {
		session := sessions.Default(c)

		fname, _ := c.GetPostForm("fname")
		lname, _ := c.GetPostForm("lname")
		email, _ := c.GetPostForm("email")
		password, _ := c.GetPostForm("password")
		passwordR, _ := c.GetPostForm("password_r")

		if !emailExists(email, usersCollection) {
			// If password is not same
			if password != passwordR {
				c.Redirect(302, "/signup")

			} else {

				// Hash password
				passwordHash := hasher(password)

				user := User{
					fname,
					lname,
					email,
					passwordHash,
				}

				usersCollection.InsertOne(context.TODO(), user)

				session.Set("auth", true) // authenticate user
				session.Set("email", email)
				session.Set("name", fname+" "+lname)

				session.Save() // save session

				c.Redirect(302, "/dashboard")
			}
		} else {
			c.Redirect(302, "/signup")
		}
	})

	router.POST("/api/login", AlrAuth(), func(c *gin.Context) {
		session := sessions.Default(c)

		email, _ := c.GetPostForm("email")
		password, _ := c.GetPostForm("password")

		filter := bson.M{"email": email, "password": hasher(password)}
		var result User
		err := usersCollection.FindOne(context.TODO(), filter).Decode(&result)

		if err != nil {
			c.Redirect(302, "/login")
		} else {
			session.Set("auth", true) // authenticate user
			session.Set("email", email)
			session.Set("name", result.FName+" "+result.LName)
			session.Save()

			c.Redirect(302, "/dashboard")
		}

	})

	// port 3000
	router.Run(":3000")
}
