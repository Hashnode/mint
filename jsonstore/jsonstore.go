package jsonstore

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mint/code"

	"github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	// TODO should come from config file
	numActiveValidators = 4
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

// Validator
type Validator struct {
	ID      []byte       `bson:"_id" json:"_id"`
	Name    string       `bson:"name" json:"name"`
	PubKey  types.PubKey `bson:"pubKey" json:"pubKey"`
	Upvotes int          `bson:"upvotes" json:"upvotes"`
	Power   int64        `bson:"power" json:"power"`
}

func (v Validator) ToTDValidator() types.Validator {
	return types.Validator{
		PubKey: v.PubKey,
		Power:  v.Power,
	}
}

// ValidatorsVotes
// TODO there should be an index ensured on ValidatorID, UserID and
// probs multikey index on ValidatorID and UserID
type UserValidatorVote struct {
	ID          bson.ObjectId `bson:"_id" json:"_id"`
	ValidatorID bson.ObjectId `bson:"validatorID" json:"validatorID"`
	UserID      bson.ObjectId `bson:"userID" json:"userID"`
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

func findTotalDocuments(db *mgo.Database) int64 {
	collections := [6]string{"posts", "comments", "users", "userpostvotes", "usercommentvotes", "validators"}
	var sum int64

	for _, collection := range collections {
		count, _ := db.C(collection).Find(nil).Count()
		sum += int64(count)
	}

	return sum
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

		if strings.Index(post.Title, "Show UH:") == 0 {
			post.ShowUH = true
		} else if strings.Index(post.Title, "Ask UH:") == 0 {
			post.AskUH = true
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

		// While replaying the transaction, check if it has been marked as spam
		spamCount, _ := db.C("spams").Find(bson.M{"postID": post.ID}).Count()

		if spamCount > 0 {
			post.Spam = true
		}

		dbErr := db.C("posts").Insert(post)

		if dbErr != nil {
			panic(dbErr)
		}

		// TODO is that really needed? it's being created on upvotePost too.
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

		pubKeyBytes, errDecode := base64.StdEncoding.DecodeString(message["publicKey"].(string))

		if errDecode != nil {
			panic(errDecode)
		}

		publicKey := strings.ToUpper(byteToHex(pubKeyBytes))

		user.PublicKey = publicKey

		dbErr := db.C("users").Insert(user)

		if dbErr != nil {
			panic(dbErr)
		}

		break

	case "upvoteValidator":
		entity := body["entity"].(map[string]interface{})

		// validate user exists
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
		fmt.Println("user validated!")

		userID := user.ID
		validatorID := bson.ObjectIdHex(entity["validator"].(string))

		// validate validator exists
		_, err = db.C("validators").Find(bson.M{"_id": validatorID}).Limit(1).Count()
		if err != nil {
			panic(err)
		}

		userValidatorVote := UserValidatorVote{}
		var upvote int8
		err = db.C("uservalidatorvotes").Find(bson.M{"userID": userID, "validatorID": validatorID}).One(&userValidatorVote)
		if err == nil {
			errRemoval := db.C("uservalidatorvotes").Remove(bson.M{"userID": userID, "validatorID": validatorID})
			if errRemoval == nil {
				upvote = -1
			}
		} else {
			var newUserValidatorVote UserValidatorVote
			newUserValidatorVote.ID = bson.NewObjectId()
			newUserValidatorVote.UserID = userID
			newUserValidatorVote.ValidatorID = validatorID

			insertErr := db.C("uservalidatorvotes").Insert(newUserValidatorVote)
			if insertErr == nil {
				upvote = 1
			}
		}
		err = db.C("validators").Update(bson.M{"_id": validatorID}, bson.M{"$inc": bson.M{"upvotes": upvote}})
		if err != nil {
			// TODO should be properly logged
			fmt.Sprintf("Failed to update votes for validator %s by user %s", validatorID, userID)
		}
		fmt.Println("validator validated!")

	case "upvotePost":
		entity := body["entity"].(map[string]interface{})

		pubKeyBytes, errDecode := base64.StdEncoding.DecodeString(message["publicKey"].(string))

		if errDecode != nil {
			panic(errDecode)
		}

		publicKey := strings.ToUpper(byteToHex(pubKeyBytes))

		var user User
		errUser := db.C("users").Find(bson.M{"publicKey": publicKey}).One(&user)
		if errUser != nil {
			panic(errUser)
		}

		userID := user.ID
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

		pubKeyBytes, errDecode := base64.StdEncoding.DecodeString(message["publicKey"].(string))

		if errDecode != nil {
			panic(errDecode)
		}

		publicKey := strings.ToUpper(byteToHex(pubKeyBytes))

		var user User
		errUser := db.C("users").Find(bson.M{"publicKey": publicKey}).One(&user)
		if errUser != nil {
			panic(errUser)
		}

		userID := user.ID
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

	// ==== Signature Validation =======
	pubKeyBytes, err := base64.StdEncoding.DecodeString(message["publicKey"].(string))
	sigBytes, err := hex.DecodeString(message["signature"].(string))
	messageBytes := []byte(message["body"].(string))

	isCorrect := ed25519.Verify(pubKeyBytes, messageBytes, sigBytes)

	if isCorrect != true {
		return types.ResponseCheckTx{Code: code.CodeTypeBadSignature}
	}
	// ==== Signature Validation =======

	var bodyTemp interface{}

	errBody := json.Unmarshal([]byte(message["body"].(string)), &bodyTemp)

	if errBody != nil {
		panic(errBody)
	}

	body := bodyTemp.(map[string]interface{})

	// ==== Does the user really exist? ======
	if body["type"] != "createUser" {
		publicKey := strings.ToUpper(byteToHex(pubKeyBytes))

		count, _ := db.C("users").Find(bson.M{"publicKey": publicKey}).Count()

		if count == 0 {
			return types.ResponseCheckTx{Code: code.CodeTypeBadData}
		}
	}
	// ==== Does the user really exist? ======

	codeType := code.CodeTypeOK

	// ===== Data Validation =======
	switch body["type"] {
	// TODO consider doing more sophisticated validations here like
	// checking existance of users and validators here
	// assuming tx cannot be tempered with beteen CheckTx and DeliverTx
	case "upvoteValidator":
		fmt.Println("received upvote start")
		entity := body["entity"].(map[string]interface{})

		if (entity["validator"] == nil) || (bson.IsObjectIdHex(entity["validator"].(string)) != true) {
			codeType = code.CodeTypeBadData
			break
		}
		fmt.Println("received upvote here")
		if (entity["user"] == nil) || (bson.IsObjectIdHex(entity["user"].(string)) != true) {
			codeType = code.CodeTypeBadData
			break
		}
		fmt.Println("received upvote end")

	case "createPost":
		entity := body["entity"].(map[string]interface{})

		if (entity["id"] == nil) || (bson.IsObjectIdHex(entity["id"].(string)) != true) {
			codeType = code.CodeTypeBadData
			break
		}

		if entity["title"] == nil || strings.TrimSpace(entity["title"].(string)) == "" {
			codeType = code.CodeTypeBadData
			break
		}

		if (entity["url"] != nil) && (strings.TrimSpace(entity["url"].(string)) != "") {
			_, err := url.ParseRequestURI(entity["url"].(string))
			if err != nil {
				codeType = code.CodeTypeBadData
				break
			}
		}
	case "createUser":
		entity := body["entity"].(map[string]interface{})

		if (entity["id"] == nil) || (bson.IsObjectIdHex(entity["id"].(string)) != true) {
			codeType = code.CodeTypeBadData
			break
		}

		r, _ := regexp.Compile("^[A-Za-z_0-9]+$")

		if (entity["username"] == nil) || (strings.TrimSpace(entity["username"].(string)) == "") || (r.MatchString(entity["username"].(string)) != true) {
			codeType = code.CodeTypeBadData
			break
		}

		if (entity["name"] == nil) || (strings.TrimSpace(entity["name"].(string)) == "") {
			codeType = code.CodeTypeBadData
			break
		}
	case "createComment":
		entity := body["entity"].(map[string]interface{})

		if (entity["id"] == nil) || (bson.IsObjectIdHex(entity["id"].(string)) != true) {
			codeType = code.CodeTypeBadData
			break
		}

		if (entity["postId"] == nil) || (bson.IsObjectIdHex(entity["postId"].(string)) != true) {
			codeType = code.CodeTypeBadData
			break
		}

		if (entity["content"] == nil) || (strings.TrimSpace(entity["content"].(string)) == "") {
			codeType = code.CodeTypeBadData
			break
		}
	}

	// ===== Data Validation =======

	return types.ResponseCheckTx{Code: codeType}
}

// Commit ...Commit the block. Calculate the appHash
func (app *JSONStoreApplication) Commit() types.ResponseCommit {
	appHash := make([]byte, 8)

	count := findTotalDocuments(db)

	binary.PutVarint(appHash, count)

	return types.ResponseCommit{Data: appHash}
}

// Query ... Query the blockchain. Unimplemented as of now.
func (app *JSONStoreApplication) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	return
}

func (app *JSONStoreApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	fmt.Println("calling InitChain")
	// TODO only used for testing atm!!!
	addValidators(req.GetValidators())
	return types.ResponseInitChain{}
}

func addValidators(validators []types.Validator) {
	var mintValidators []interface{}
	if validators != nil {
		for _, element := range validators {
			validator := Validator{
				ID:      element.GetAddress(),
				Name:    "mintValidator",
				PubKey:  element.GetPubKey(),
				Power:   element.GetPower(),
				Upvotes: 0,
			}
			mintValidators = append(mintValidators, validator)
		}
		dbErr := db.C("validators").Insert(mintValidators...)
		if dbErr != nil {
			panic(dbErr)
		}
	}
}

func (app *JSONStoreApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	// TODO below solution needs confirmation through testing on multiple nodes.
	// according to documentation "To add a new validator or update an existing one,
	// simply include them in the list returned in the EndBlock response.
	// To remove one, include it in the list with a power equal to 0."
	// That means below solution may not be removing any validators atm!

	var validators []Validator

	err := db.C("validators").Find(nil).Sort("-upvotes").Limit(numActiveValidators).All(&validators)
	if err != nil {
		panic(err)
	}

	var tdValidators []types.Validator

	for _, validator := range validators {
		tdValidator := validator.ToTDValidator()
		tdValidators = append(tdValidators, tdValidator)
	}
	fmt.Println(tdValidators)

	return types.ResponseEndBlock{ValidatorUpdates: tdValidators}
}
