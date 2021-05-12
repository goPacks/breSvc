package mongosvc

import (
	"breSvc/structs"
	"context"
	"fmt"
	"time"

	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/* Used to create a singleton object of MongoDB client.
Initialized and exposed through  GetMongoClient().*/
var clientInstance *mongo.Client

//Used during creation of singleton client object in GetMongoClient().
var clientInstanceError error

//Used to execute client creation procedure only once.
var mongoOnce sync.Once

//I have used below constants just to hold required database config's.
const (
	CONNECTIONSTRING = "mongodb://localhost:27017"
	DB               = "breSvc"
	BREPKG           = "brePkg"
	USER             = "user"
)

//GetMongoClient - Return mongodb connection to work with
func getMongo() (*mongo.Client, error) {
	//Perform connection creation operation only once.
	mongoOnce.Do(func() {
		// Set client options
		clientOptions := options.Client().ApplyURI(CONNECTIONSTRING)
		// Connect to MongoDB
		client, err := mongo.Connect(context.TODO(), clientOptions)
		if err != nil {
			clientInstanceError = err
		}
		// Check the connection
		err = client.Ping(context.TODO(), nil)
		if err != nil {
			clientInstanceError = err
		}
		clientInstance = client
	})

	return clientInstance, clientInstanceError
}

func chkOverLap(pkgCode, site, cat, validFrom, validTo string, collection *mongo.Collection) error {

	filter := bson.M{"site": site, "cat": cat}

	cur, findError := collection.Find(context.TODO(), filter)
	if findError != nil {
		return findError
	}

	for cur.Next(context.TODO()) {
		t := structs.BrePkg{}
		err := cur.Decode(&t)
		if err != nil {
			return err
		}

		if t.PkgCode != pkgCode {
			if validFrom >= t.ValidFrom && validFrom <= t.ValidTo {
				return fmt.Errorf("Package dates conflict with Package %s - From %s to %s", t.PkgCode, t.ValidFrom, t.ValidTo)
			}

			if validTo >= t.ValidFrom && validTo <= t.ValidTo {
				return fmt.Errorf("Package dates conflict with Package %s - From %s to %s", t.PkgCode, t.ValidFrom, t.ValidTo)
			}

			if validFrom <= t.ValidFrom && validTo >= t.ValidTo {
				return fmt.Errorf("Package dates conflict with Package %s - From %s to %s", t.PkgCode, t.ValidFrom, t.ValidTo)
			}
		}

	}
	// once exhausted, close the cursor
	cur.Close(context.TODO())

	return nil

}

func Upsert(brePkg structs.BrePkg, user *structs.User) (bson.M, error) {

	// 1) Create the context
	exp := 120 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), exp)
	defer cancel()

	client, err := getMongo()
	if err != nil {
		return nil, err
	}

	colName := user.Sbu + "." + BREPKG

	collection := client.Database(DB).Collection(colName)

	err = chkOverLap(brePkg.PkgCode, brePkg.Site, brePkg.Cat, brePkg.ValidFrom, brePkg.ValidTo, collection)
	if err != nil {
		return nil, err
	}

	// 5) Create the search filter
	filter := bson.M{"PkgId": fmt.Sprintf("%s.%s.%s", brePkg.Site, brePkg.Cat, brePkg.PkgCode)}

	// 6) Create the update
	update := bson.M{
		"$set": brePkg,
	}

	// 7) Create an instance of an options and set the desired options
	upsert := true
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}

	// 8) Find one result and update it
	result := collection.FindOneAndUpdate(ctx, filter, update, &opt)
	if result.Err() != nil {
		return nil, result.Err()
	}

	// 9) Decode the result
	doc := bson.M{}
	decodeErr := result.Decode(&doc)

	return doc, decodeErr
}

// func Get(pkgCode string) (structs.BrePkg, error) {

// 	result := structs.BrePkg{}

// 	//Define filter query for fetching specific document from collection
// 	filter := bson.D{primitive.E{Key: "PkgCode", Value: pkgCode}}

// 	//Get MongoDB connection using connectionhelper.
// 	client, err := getMongo()
// 	if err != nil {
// 		return result, err
// 	}
// 	//Create a handle to the respective collection in the database.
// 	collection := client.Database(DB).Collection(BREPKG)
// 	//Perform FindOne operation & validate against the error.
// 	err = collection.FindOne(context.TODO(), filter).Decode(&result)
// 	if err != nil {
// 		return result, err
// 	}
// 	//Return result without any error.
// 	return result, nil
// }

func GetBrePkg(pkgCode string, user *structs.User) (structs.BrePkg, error) {

	var result structs.BrePkg

	//Define filter query for fetching specific document from collection
	filter := bson.D{primitive.E{Key: "PkgId", Value: pkgCode}}

	//Get MongoDB connection using connectionhelper.
	client, err := getMongo()
	if err != nil {
		return result, err
	}

	colName := user.Sbu + "." + BREPKG
	//Create a handle to the respective collection in the database.
	collection := client.Database(DB).Collection(colName)
	//Perform FindOne operation & validate against the error.
	err = collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return result, err
	}
	//Return result without any error.
	return result, nil
}

func Del(pkgCode string, user *structs.User) error {
	//Define filter query for fetching specific document from collection
	filter := bson.D{primitive.E{Key: "PkgId", Value: pkgCode}}
	//Get MongoDB connection using connectionhelper.
	client, err := getMongo()
	if err != nil {
		return err
	}

	colName := user.Sbu + "." + BREPKG
	//Create a handle to the respective collection in the database.
	collection := client.Database(DB).Collection(colName)
	//Perform DeleteOne operation & validate against the error.
	_, err = collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		return err
	}
	//Return success without any error.
	return nil
}

func DelAll(user *structs.User) error {
	//Get MongoDB connection using connectionhelper.
	client, err := getMongo()
	if err != nil {
		return err
	}

	colName := user.Sbu + "." + BREPKG
	//Create a handle to the respective collection in the database.
	collection := client.Database(DB).Collection(colName)
	//Perform DeleteOne operation & validate against the error.
	_, err = collection.DeleteMany(context.TODO(), bson.D{{}})

	if err != nil {
		return err
	}
	//Return success without any error.
	return nil
}

func GetAll(user *structs.User) ([]structs.BrePkg, error) {
	//Define filter query for fetching specific document from collection
	filter := bson.D{{}} //bson.D{{}} specifies 'all documents'
	brePkgs := []structs.BrePkg{}
	//Get MongoDB connection using connectionhelper.
	client, err := getMongo()
	if err != nil {
		return brePkgs, err
	}

	colName := user.Sbu + "." + BREPKG
	//Create a handle to the respective collection in the database.
	collection := client.Database(DB).Collection(colName)
	//Perform Find operation & validate against the error.
	cur, findError := collection.Find(context.TODO(), filter)
	if findError != nil {
		return brePkgs, findError
	}
	//Map result to slice
	for cur.Next(context.TODO()) {
		t := structs.BrePkg{}
		err := cur.Decode(&t)
		if err != nil {
			return brePkgs, err
		}
		brePkgs = append(brePkgs, t)
	}
	// once exhausted, close the cursor
	cur.Close(context.TODO())
	if len(brePkgs) == 0 {
		return brePkgs, fmt.Errorf("No Documents")
	}
	return brePkgs, nil
}

func GetUser(userId string) (structs.User, error) {

	//Define filter query for fetching specific document from collection
	filter := bson.D{primitive.E{Key: "UserId", Value: userId}}

	var result structs.User

	//Get MongoDB connection using connectionhelper.
	client, err := getMongo()
	if err != nil {
		return result, err
	}

	//Create a handle to the respective collection in the database.
	collection := client.Database(DB).Collection(USER)
	//Perform FindOne operation & validate against the error.
	err = collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return result, err
	}
	//Return result without any error.
	return result, nil
}

func RegUser(userId string, user structs.User) (bson.M, error) {

	// 1) Create the context
	exp := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), exp)
	defer cancel()

	client, err := getMongo()
	if err != nil {

	}

	collection := client.Database(DB).Collection(USER)

	// 5) Create the search filter
	filter := bson.M{"UserId": userId}

	// 6) Create the update
	update := bson.M{
		//	"$set": bson.M{"lastname": "skywalker"},
		"$set": user,
	}

	// 7) Create an instance of an options and set the desired options
	upsert := true
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}

	// 8) Find one result and update it
	result := collection.FindOneAndUpdate(ctx, filter, update, &opt)
	if result.Err() != nil {
		return nil, result.Err()
	}

	// 9) Decode the result
	doc := bson.M{}
	decodeErr := result.Decode(&doc)

	return doc, decodeErr
}
