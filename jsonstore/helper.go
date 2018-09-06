package jsonstore

import (
	"fmt"
	"math"
	"strconv"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func byteToHex(input []byte) string {
	var hexValue string
	for _, v := range input {
		hexValue += fmt.Sprintf("%02x", v)
	}
	return hexValue
}

func getPosts(db *mgo.Database, sortBy string, searchConfig map[string]interface{}) ([]interface{}, error) {
	var posts []interface{}
	var user User

	err := db.C("posts").Find(searchConfig).Sort("-" + sortBy).All(&posts)
	if err != nil {
		panic(err)
	}

	for i := range posts {
		err = db.C("users").Find(bson.M{"_id": posts[i].(bson.M)["author"].(bson.ObjectId)}).One(&user)
		if err != nil {
			panic(err)
		}

		posts[i].(bson.M)["author"] = user
	}

	return posts, nil
}

func findTotalDocuments(db *mgo.Database) int64 {
	collections := [5]string{"posts", "comments", "users", "userpostvotes", "usercommentvotes"}
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
