package jsonstore

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"mint/code"

	"github.com/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var _ types.Application = (*JSONStoreApplication)(nil)
var db *mgo.Database

// Post ...
type Post struct {
	ID          bson.ObjectId `bson:"_id" json:"_id"`
	Title       string        `bson:"title" json:"title"`
	URL         string        `bson:"url" json:"url"`
	Text        string        `bson:"text" json:"text"`
	Author      bson.ObjectId `bson:"author" json:"author"`
	Upvotes     int           `bson:"upvotes" json:"upvotes"`
	Date        time.Time     `bson:"date" json:"date"`
	Score       float64       `bson:"score" json:"score"`
	NumComments int           `bson:"numComments" json:"numComments"`
	AskHN       bool          `bson:"askHN" json:"askHN"`
	ShowHN      bool          `bson:"showHN" json:"showHN"`
}

// Comment ...
type Comment struct {
	ID              bson.ObjectId `bson:"_id" json:"_id"`
	Content         string        `bson:"content" json:"content"`
	Author          bson.ObjectId `bson:"author" json:"author"`
	Upvotes         int           `bson:"upvotes" json:"upvotes"`
	Score           float64       `bson:"score" json:"score"`
	Date            time.Time
	PostID          bson.ObjectId `bson:"postID" json:"postID"`
	ParentCommentID bson.ObjectId `bson:"parentCommentId,omitempty" json:"parentCommentId"`
}

// User ...
type User struct {
	ID        bson.ObjectId `bson:"_id" json:"_id"`
	Name      string        `bson:"name" json:"name"`
	Username  string        `bson:"username" json:"username"`
	PublicKey string        `bson:"publicKey" json:"publicKey"`
}

// UserPostVote ...
type UserPostVote struct {
	ID     bson.ObjectId `bson:"_id" json:"_id"`
	UserID bson.ObjectId `bson:"userID" json:"userID"`
	PostID bson.ObjectId `bson:"postID" json:"postID"`
}

// UserCommentVote ...
type UserCommentVote struct {
	ID        bson.ObjectId `bson:"_id" json:"_id"`
	UserID    bson.ObjectId `bson:"userID" json:"userID"`
	CommentID bson.ObjectId `bson:"commentID" json:"commentID"`
}

// DBStats ...
type DBStats struct {
	Collections int     `bson:"collections"`
	Objects     int64   `bson:"objects"`
	AvgObjSize  float64 `bson:"avgObjSize"`
	DataSize    float64 `bson:"dataSize"`
	StorageSize float64 `bson:"storageSize"`
	FileSize    float64 `bson:"fileSize"`
	IndexSize   float64 `bson:"indexSize"`
}

// JSONStoreApplication ...
type JSONStoreApplication struct {
	types.BaseApplication
}

func byteToHex(input []byte) string {
	var hexValue string
	for _, v := range input {
		hexValue += fmt.Sprintf("%02x", v)
	}
	return hexValue
}

func hotScore(votes int, date time.Time) float64 {
	gravity := 1.8
	hoursAge := float64(date.Unix() * 3600)
	return float64(votes-1) / math.Pow(hoursAge+2, gravity)
}

// FindTimeFromObjectID ... Convert ObjectID string to Time
func FindTimeFromObjectID(id string) time.Time {
	ts, _ := strconv.ParseInt(id[0:8], 16, 64)
	return time.Unix(ts, 0)
}

// NewJSONStoreApplication ...
func NewJSONStoreApplication(dbCopy *mgo.Database) *JSONStoreApplication {
	db = dbCopy
	return &JSONStoreApplication{}
}

// Info ...
func (app *JSONStoreApplication) Info(req types.RequestInfo) (resInfo types.ResponseInfo) {
	return types.ResponseInfo{Data: fmt.Sprintf("{\"size\":%v}", 0)}
}

// DeliverTx ... Update the global state
func (app *JSONStoreApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {

	var temp interface{}
	err := json.Unmarshal(tx, &temp)

	if err != nil {
		panic(err)
	}

	message := temp.(map[string]interface{})

	var bodyTemp interface{}

	errBody := json.Unmarshal([]byte(message["body"].(string)), &bodyTemp)

	if errBody != nil {
		panic(errBody)
	}

	body := bodyTemp.(map[string]interface{})

	switch body["type"] {
	case "createPost":
		entity := body["entity"].(map[string]interface{})

		var post Post
		post.ID = bson.ObjectIdHex(entity["id"].(string))
		post.Title = entity["title"].(string)

		if entity["url"] != nil {
			post.URL = entity["url"].(string)
		}
		if entity["text"] != nil {
			post.Text = entity["text"].(string)
		}

		if strings.Index(post.Title, "Show HN:") == 0 {
			post.ShowHN = true
		} else if strings.Index(post.Title, "Ask HN:") == 0 {
			post.AskHN = true
		}

		pubKeyBytes, errDecode := base64.StdEncoding.DecodeString(message["publicKey"].(string))

		if errDecode != nil {
			panic(errDecode)
		}

		publicKey := strings.ToUpper(byteToHex(pubKeyBytes))

		var user User
		err := db.C("users").Find(bson.M{"publicKey": publicKey}).One(&user)
		if err != nil {
			panic(err)
		}
		post.Author = user.ID

		post.Date = FindTimeFromObjectID(post.ID.Hex())

		post.Upvotes = 1

		post.NumComments = 0

		// Calculate hot rank
		post.Score = hotScore(post.Upvotes, post.Date)

		dbErr := db.C("posts").Insert(post)

		if dbErr != nil {
			panic(dbErr)
		}

		var document UserPostVote
		document.ID = bson.NewObjectId()
		document.UserID = user.ID
		document.PostID = post.ID

		db.C("userpostvotes").Insert(document)

		break
	case "createUser":
		entity := body["entity"].(map[string]interface{})

		var user User
		user.ID = bson.ObjectIdHex(entity["id"].(string))
		user.Username = entity["username"].(string)
		user.Name = entity["name"].(string)
		user.PublicKey = entity["publicKey"].(string)

		dbErr := db.C("users").Insert(user)

		if dbErr != nil {
			panic(dbErr)
		}

		break
	case "upvotePost":
		entity := body["entity"].(map[string]interface{})

		userID := bson.ObjectIdHex(entity["upvoter"].(string))
		postID := bson.ObjectIdHex(entity["postId"].(string))

		userPostVote := UserPostVote{}
		err := db.C("userpostvotes").Find(bson.M{"userID": userID, "postID": postID}).One(&userPostVote)

		if err == nil {
			// A document was found
			errRemoval := db.C("userpostvotes").Remove(bson.M{"userID": userID, "postID": postID})
			if errRemoval == nil {
				db.C("posts").Update(bson.M{"_id": postID}, bson.M{"$inc": bson.M{"upvotes": -1}})
			}
		} else {
			var document UserPostVote
			document.ID = bson.NewObjectId()
			document.UserID = userID
			document.PostID = postID

			insertErr := db.C("userpostvotes").Insert(document)

			if insertErr == nil {
				db.C("posts").Update(bson.M{"_id": postID}, bson.M{"$inc": bson.M{"upvotes": 1}})
			}
		}

		// Calculate hot rank
		var post Post
		errPost := db.C("posts").Find(bson.M{"_id": postID}).One(&post)

		if errPost != nil {
			panic(errPost)
		}

		score := hotScore(post.Upvotes, post.Date)

		db.C("posts").Update(bson.M{"_id": postID}, bson.M{"$set": bson.M{"score": score}})

		break
	case "createComment":
		entity := body["entity"].(map[string]interface{})

		var comment Comment

		comment.ID = bson.ObjectIdHex(entity["id"].(string))
		comment.Content = entity["content"].(string)
		comment.Date = FindTimeFromObjectID(comment.ID.Hex())
		comment.PostID = bson.ObjectIdHex(entity["postId"].(string))
		comment.Upvotes = 1
		comment.Score = hotScore(comment.Upvotes, comment.Date)

		if entity["parentCommentId"] != nil {
			comment.ParentCommentID = bson.ObjectIdHex(entity["parentCommentId"].(string))
		}

		pubKeyBytes, errDecode := base64.StdEncoding.DecodeString(message["publicKey"].(string))

		if errDecode != nil {
			panic(errDecode)
		}

		publicKey := strings.ToUpper(byteToHex(pubKeyBytes))

		var user User
		err := db.C("users").Find(bson.M{"publicKey": publicKey}).One(&user)
		if err != nil {
			panic(err)
		}
		comment.Author = user.ID

		dbErr := db.C("comments").Insert(comment)

		if dbErr != nil {
			panic(dbErr)
		}

		// For recording default upvote
		var document UserCommentVote
		document.ID = bson.NewObjectId()
		document.UserID = user.ID
		document.CommentID = comment.ID

		db.C("usercommentvotes").Insert(document)
		db.C("posts").Update(bson.M{"_id": comment.PostID}, bson.M{"$inc": bson.M{"numComments": 1}})

		break
	case "upvoteComment":
		entity := body["entity"].(map[string]interface{})

		userID := bson.ObjectIdHex(entity["upvoter"].(string))
		commentID := bson.ObjectIdHex(entity["commentId"].(string))

		userCommentVote := UserCommentVote{}
		err := db.C("usercommentvotes").Find(bson.M{"userID": userID, "commentID": commentID}).One(&userCommentVote)

		if err == nil {
			// A document was found
			errRemoval := db.C("usercommentvotes").Remove(bson.M{"userID": userID, "commentID": commentID})
			if errRemoval == nil {
				db.C("comments").Update(bson.M{"_id": commentID}, bson.M{"$inc": bson.M{"upvotes": -1}})
			}
		} else {
			var document UserCommentVote
			document.ID = bson.NewObjectId()
			document.UserID = userID
			document.CommentID = commentID

			insertErr := db.C("usercommentvotes").Insert(document)

			if insertErr == nil {
				db.C("comments").Update(bson.M{"_id": commentID}, bson.M{"$inc": bson.M{"upvotes": 1}})
			}
		}

		// Calculate hot rank
		var comment Comment
		errComment := db.C("comments").Find(bson.M{"_id": commentID}).One(&comment)

		if errComment != nil {
			panic(errComment)
		}

		score := hotScore(comment.Upvotes, comment.Date)

		db.C("comments").Update(bson.M{"_id": commentID}, bson.M{"$set": bson.M{"score": score}})

		break
	}

	return types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: nil}
}

// CheckTx ... Verify the transaction
func (app *JSONStoreApplication) CheckTx(tx []byte) types.ResponseCheckTx {
	var temp interface{}
	err := json.Unmarshal(tx, &temp)

	if err != nil {
		panic(err)
	}

	message := temp.(map[string]interface{})

	pubKeyBytes, err := base64.StdEncoding.DecodeString(message["publicKey"].(string))
	sigBytes, err := hex.DecodeString(message["signature"].(string))
	messageBytes := []byte(message["body"].(string))

	isCorrect := ed25519.Verify(pubKeyBytes, messageBytes, sigBytes)

	if isCorrect != true {
		return types.ResponseCheckTx{Code: code.CodeTypeBadSignature}
	}

	return types.ResponseCheckTx{Code: code.CodeTypeOK}
}

// Commit ...Commit the block. Calculate the appHash
func (app *JSONStoreApplication) Commit() types.ResponseCommit {
	appHash := make([]byte, 8)

	var dbStats DBStats
	if err := db.Run(bson.D{{"dbStats", 1}, {"scale", 1}}, &dbStats); err != nil {
		log.Fatal(err)
	}

	binary.PutVarint(appHash, dbStats.Objects)

	return types.ResponseCommit{Data: appHash}
}

// Query ... Query the blockchain. Unimplemented as of now.
func (app *JSONStoreApplication) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	return
}
