package main

import (
	"log"

	ams "github.com/crossi36/applymagicsauce"
)

func main() {
	token, err := ams.Auth(0, "YOUR_API_KEY")
	if err != nil {
		log.Fatalf("could not get authentication token: %v", err)
	}

	textOptions := ams.PredictTextOptions(ams.SourceOther, nil, true)
	textPrediction, err := ams.PredictText("Lorem ipsum dolor sit amet", textOptions, token)
	if err != nil {
		log.Fatalf("could not predict text: %v", err)
	}
	log.Printf("%#v\n", textPrediction)

	ids := []string{"5845317146", "6460713406", "22404294985", "35312278675", "105930651606", "171605907303", "199592894970", "274598553922", "340368556015", "100270610030980"}

	likeOptions := ams.PredictLikeIDsOptions(nil, true, true)
	likePredictions, err := ams.PredictLikeIDs(ids, likeOptions, token)
	if err != nil {
		log.Fatalf("could not predict likes: %v", err)
	}
	log.Printf("%#v\n", likePredictions)
}
