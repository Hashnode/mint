package jsonstore

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/Hashnode/mint/code"

	"github.com/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var _ types.Application = (*JSONStoreApplication)(nil)
var db *mgo.Database

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
	var temp interface{}
	var user User
	var t map[string]interface{}

	err := json.Unmarshal(reqQuery.Data, &temp)
	if err != nil {
		panic(err)
	}

	t = temp.(map[string]interface{})

	switch reqQuery.Path {
	case "/fetch-user":
		err := db.C("users").Find(bson.M{"publicKey": t["publicKey"].(string)}).One(&user)
		if err != nil {
			panic(err)
			// resQuery.Log = err.Error()
			// break
		}

		resData, err := json.Marshal(user)
		if err != nil {
			panic(err)
			// resQuery.Log = err.Error()
			// break
		}
		resQuery.Value = resData
	case "/get-posts":

		searchConfig := make(map[string]interface{})

		searchConfig["spam"] = false
		if val, ok := t["type"]; ok {
			if val.(string) == "askUH" || val.(string) == "showUH" {
				searchConfig[val.(string)] = true
			} else {
				resQuery.Log = "type field can only have values 'showUH' or 'askUH'"
				break
			}
		}

		if t["sortBy"].(string) != "score" && t["sortBy"].(string) != "date" {
			resQuery.Log = "sortBy field can only have values 'score' or 'date'"
			break
		}

		posts, err := getPosts(db, t["sortBy"].(string), searchConfig)
		if err != nil {
			panic(err)
		}

		resData, err := json.Marshal(posts)
		if err != nil {
			panic(err)
			// resQuery.Log = err.Error()
			// break
		}
		resQuery.Value = resData
	case "/get-upvote-status":
		var wg sync.WaitGroup
		var status map[string]interface{}

		status = make(map[string]interface{})

		err := db.C("users").Find(bson.M{"publicKey": t["publicKey"].(string)}).One(&user)
		if err != nil {
			panic(err)
		}

		for _, postID := range t["postIds"].([]interface{}) {
			wg.Add(1)
			go func(postID string) {
				defer wg.Done()
				count, err := db.C("userpostvotes").Find(bson.M{"postID": bson.ObjectIdHex(postID), "userID": user.ID}).Count()
				if err != nil {
					panic(err)
				}

				if count > 0 {
					status[postID] = true
				}
			}(postID.(string))
		}

		wg.Wait()
		resData, err := json.Marshal(status)
		if err != nil {
			resQuery.Log = err.Error()
			break
		}
		resQuery.Value = resData
	case "/comment":
		var comment map[string]interface{}
		err := db.C("comments").Find(bson.M{"_id": bson.ObjectIdHex(t["id"].(string))}).One(&comment)
		if err != nil {
			panic(err)
		}

		err = db.C("users").Find(bson.M{"_id": comment["author"].(bson.ObjectId)}).One(&user)
		if err != nil {
			panic(err)
		}

		comment["author"] = user

		resData, err := json.Marshal(comment)
		if err != nil {
			panic(err)
			// resQuery.Log = err.Error()
			// break
		}
		resQuery.Value = resData
	case "/get-comment-upvote-status":
		var wg sync.WaitGroup
		var status map[string]interface{}

		status = make(map[string]interface{})

		err := db.C("users").Find(bson.M{"publicKey": t["publicKey"].(string)}).One(&user)
		if err != nil {
			panic(err)
		}

		for _, commentID := range t["commentIds"].([]interface{}) {
			wg.Add(1)
			go func(commentID string) {
				defer wg.Done()
				count, err := db.C("usercommentvotes").Find(bson.M{"commentID": bson.ObjectIdHex(commentID), "userID": user.ID}).Count()
				if err != nil {
					panic(err)
				}

				if count > 0 {
					status[commentID] = true
				}
			}(commentID.(string))
		}

		wg.Wait()
		resData, err := json.Marshal(status)
		if err != nil {
			panic(err)
			// resQuery.Log = err.Error()
			// break
		}
		resQuery.Value = resData
	case "/post":
		var wg sync.WaitGroup
		var post map[string]interface{}
		var user User
		var comments []map[string]interface{}

		post = make(map[string]interface{})

		err := db.C("posts").Find(bson.M{"_id": bson.ObjectIdHex(t["id"].(string))}).One(&post)
		if err != nil {
			panic(err)
		}

		err = db.C("users").Find(bson.M{"_id": post["author"].(bson.ObjectId)}).One(&user)
		if err != nil {
			panic(err)
		}

		post["author"] = user

		err = db.C("comments").Find(bson.M{"postID": bson.ObjectIdHex(t["id"].(string))}).All(&comments)
		if err != nil {
			panic(err)
		}

		for i := range comments {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				var user User
				err = db.C("users").Find(bson.M{"_id": comments[i]["author"].(bson.ObjectId)}).One(&user)
				if err != nil {
					panic(err)
				}

				comments[i]["author"] = user
			}(i)
		}

		post["comments"] = comments

		wg.Wait()
		resData, err := json.Marshal(post)
		if err != nil {
			panic(err)
			// resQuery.Log = err.Error()
			// break
		}
		resQuery.Value = resData

	}

	return
}
