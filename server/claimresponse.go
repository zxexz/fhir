package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/intervention-engine/fhir/models"
	"gopkg.in/mgo.v2/bson"
)

func ClaimResponseIndexHandler(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	var result []models.ClaimResponse
	c := Database.C("claimresponses")
	iter := c.Find(nil).Limit(100).Iter()
	err := iter.All(&result)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	var claimresponseEntryList []models.BundleEntryComponent
	for _, claimresponse := range result {
		var entry models.BundleEntryComponent
		entry.Resource = &claimresponse
		claimresponseEntryList = append(claimresponseEntryList, entry)
	}

	var bundle models.Bundle
	bundle.Id = bson.NewObjectId().Hex()
	bundle.Type = "searchset"
	var total = uint32(len(result))
	bundle.Total = &total
	bundle.Entry = claimresponseEntryList

	log.Println("Setting claimresponse search context")
	context.Set(r, "ClaimResponse", result)
	context.Set(r, "Resource", "ClaimResponse")
	context.Set(r, "Action", "search")

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(rw).Encode(&bundle)
}

func LoadClaimResponse(r *http.Request) (*models.ClaimResponse, error) {
	var id bson.ObjectId

	idString := mux.Vars(r)["id"]
	if bson.IsObjectIdHex(idString) {
		id = bson.ObjectIdHex(idString)
	} else {
		return nil, errors.New("Invalid id")
	}

	c := Database.C("claimresponses")
	result := models.ClaimResponse{}
	err := c.Find(bson.M{"_id": id.Hex()}).One(&result)
	if err != nil {
		return nil, err
	}

	log.Println("Setting claimresponse read context")
	context.Set(r, "ClaimResponse", result)
	context.Set(r, "Resource", "ClaimResponse")
	return &result, nil
}

func ClaimResponseShowHandler(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	context.Set(r, "Action", "read")
	_, err := LoadClaimResponse(r)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(rw).Encode(context.Get(r, "ClaimResponse"))
}

func ClaimResponseCreateHandler(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	decoder := json.NewDecoder(r.Body)
	claimresponse := &models.ClaimResponse{}
	err := decoder.Decode(claimresponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	c := Database.C("claimresponses")
	i := bson.NewObjectId()
	claimresponse.Id = i.Hex()
	err = c.Insert(claimresponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	log.Println("Setting claimresponse create context")
	context.Set(r, "ClaimResponse", claimresponse)
	context.Set(r, "Resource", "ClaimResponse")
	context.Set(r, "Action", "create")

	host, err := os.Hostname()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	rw.Header().Add("Location", "http://"+host+":3001/ClaimResponse/"+i.Hex())
}

func ClaimResponseUpdateHandler(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	var id bson.ObjectId

	idString := mux.Vars(r)["id"]
	if bson.IsObjectIdHex(idString) {
		id = bson.ObjectIdHex(idString)
	} else {
		http.Error(rw, "Invalid id", http.StatusBadRequest)
	}

	decoder := json.NewDecoder(r.Body)
	claimresponse := &models.ClaimResponse{}
	err := decoder.Decode(claimresponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	c := Database.C("claimresponses")
	claimresponse.Id = id.Hex()
	err = c.Update(bson.M{"_id": id.Hex()}, claimresponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	log.Println("Setting claimresponse update context")
	context.Set(r, "ClaimResponse", claimresponse)
	context.Set(r, "Resource", "ClaimResponse")
	context.Set(r, "Action", "update")
}

func ClaimResponseDeleteHandler(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	var id bson.ObjectId

	idString := mux.Vars(r)["id"]
	if bson.IsObjectIdHex(idString) {
		id = bson.ObjectIdHex(idString)
	} else {
		http.Error(rw, "Invalid id", http.StatusBadRequest)
	}

	c := Database.C("claimresponses")

	err := c.Remove(bson.M{"_id": id.Hex()})
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Setting claimresponse delete context")
	context.Set(r, "ClaimResponse", id.Hex())
	context.Set(r, "Resource", "ClaimResponse")
	context.Set(r, "Action", "delete")
}