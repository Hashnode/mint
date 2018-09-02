package jsonstore

import (
	"time"

	"github.com/tendermint/abci/types"
	"gopkg.in/mgo.v2/bson"
)

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
	AskUH       bool          `bson:"askUH" json:"askUH"`
	ShowUH      bool          `bson:"showUH" json:"showUH"`
	Spam        bool          `bson:"spam" json:"spam"`
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

// JSONStoreApplication ...
type JSONStoreApplication struct {
	types.BaseApplication
}
