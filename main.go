package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
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

func main() {
	gin.SetMode(gin.ReleaseMode)
	mgdClientOpt := options.Client().ApplyURI("mongodb+srv://na-admin:rV9byQPoKeQBJpka@trashtalkpuddle-qpofb.mongodb.net/")
	mgdClient, err := mongo.Connect(context.TODO(), mgdClientOpt)

	usersCollection := mgdClient.Database("trashtalk").Collection("users")
	postsCollection := mgdClient.Database("trashtalk").Collection("posts")

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

	router.GET("/post", Auth(), func(c *gin.Context) {
		c.File("./public/post.html")
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

	router.GET("/api/info", Auth(), func(c *gin.Context) {
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

	router.GET("/api/posts/:email", Auth(), func(c *gin.Context) {
		requestedEmail := c.Param("email")
		var final []Post
		items, err := postsCollection.Find(context.TODO(), bson.M{"authoremail": requestedEmail})
		if err != nil {
			log.Fatal(err)
		}
		for items.Next(context.TODO()) {
			var post Post
			items.Decode(&post)
			final = append(final, post)
		}
		data, err := json.Marshal(final)
		if err != nil {
			log.Fatal(err)
		}
		c.String(200, string(data))
	})

	router.GET("/api/posts/", Auth(), func(c *gin.Context) {
		var final []Post
		items, err := postsCollection.Find(context.TODO(), bson.M{})
		if err != nil {
			log.Fatal(err)
		}
		for items.Next(context.TODO()) {
			var post Post
			items.Decode(&post)
			final = append(final, post)
		}
		data, err := json.Marshal(final)
		if err != nil {
			log.Fatal(err)
		}
		c.String(200, string(data))
	})

	router.GET("/api/isitmypost/:ID", Auth(), func(c *gin.Context) {
		session := sessions.Default(c)

		clientEmail := session.Get("email")

		requestedID := c.Param("ID")
		filter := bson.M{
			"postid": requestedID,
		}
		var post Post

		postsCollection.FindOne(context.TODO(), filter).Decode(&post)

		if post.AuthorEmail == clientEmail {
			c.JSON(200, bson.M{"isClientPost": true})
		} else {
			c.JSON(200, bson.M{"isClientPost": false})
		}
	})

	router.POST("/api/delete/post/:ID", Auth(), func(c *gin.Context) {
		session := sessions.Default(c)

		clientEmail := session.Get("email")

		requestedID := c.Param("ID")
		filter := bson.M{
			"postid": requestedID,
		}
		var post Post

		postsCollection.FindOne(context.TODO(), filter).Decode(&post)

		if post.AuthorEmail == clientEmail {
			_, err := postsCollection.DeleteOne(context.TODO(), filter)
			if err != nil {
				log.Fatal(err)
			}
			c.Redirect(302, "/dashboard")
		} else {
			c.Redirect(302, "/dashboard")
		}
	})

	router.GET("/api/post/:ID", Auth(), func(c *gin.Context) {
		requestedID := c.Param("ID")
		filter := bson.M{
			"postid": requestedID,
		}
		var post Post
		err := postsCollection.FindOne(context.TODO(), filter).Decode(&post)
		if err != nil {
			c.JSON(200, bson.M{"message": "No posts found"})
		} else {
			data, err := json.Marshal(post)
			if err != nil {
				log.Fatal(err)
			}
			c.String(200, string(data))
		}
	})

	router.POST("/api/addpost", Auth(), func(c *gin.Context) {
		session := sessions.Default(c)

		prodTitle, _ := c.GetPostForm("prodTitle")
		prodDesc, _ := c.GetPostForm("prodDesc")
		prodLoc, _ := c.GetPostForm("prodLoc")
		prodImageFile, err := c.FormFile("prodImage")

		if err != nil {
			log.Fatal(err)
		}

		guid := xid.New()
		uniqueID := guid.String()

		imageExtension := filepath.Ext(prodImageFile.Filename)
		if stringInSlice(imageExtension, []string{".png", ".jpg"}) {
			err := c.SaveUploadedFile(prodImageFile, "./public/posts/"+uniqueID+imageExtension)
			if err != nil {
				log.Fatal(err)
			}

			postID := uniqueID
			authorEmail := session.Get("email")
			authorName := session.Get("name")
			date := time.Now().Format("01-02-2006")
			imagePath := uniqueID + imageExtension
			title := prodTitle
			desc := prodDesc
			location := prodLoc

			post := Post{
				postID,
				fmt.Sprintf("%v", authorEmail), // interface conversion
				fmt.Sprintf("%v", authorName),  // interface conversion
				date,
				imagePath,
				title,
				desc,
				location,
			}

			postsCollection.InsertOne(context.TODO(), post)

			c.Redirect(302, "/dashboard")
		} else {
			c.Redirect(302, "/dashboard")
		}
	})

	// port 3000
	router.Run(":3000")
}
