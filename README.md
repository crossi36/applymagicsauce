# applymagicsauce

## What it is
applymagicsauce is a library that enables you to easily use the [Apply Magic Sauce](https://applymagicsauce.com) API.

## Usage
The first step is to obtain an authentication Token. This should be done with the `Auth()` function.
```
func main() {
	token, err := ams.Auth(YOUR_CUSTOMER_ID, "YOUR_API_KEY")
	if err != nil {
		log.Fatalf("could not get authentication token: %v", err)
	}
```

Next, you create an options object for the desired prediction function. In this example we will use the `PredictText()` function. Therefore, we create a options object with the `PredictTextOptions()` function. We pass `nil` as the parameter for traits, because we do not want to limit the traits in our prediction. An empty slice would work as well.
```
	textOptions := ams.PredictTextOptions(ams.SourceOther, nil, true)
```

The last step is to use the options object and make a prediction for some arbitrary text. We use the `PredictText()` function for this.
```
	textPrediction, err := ams.PredictText("Lorem ipsum dolor sit amet", textOptions, token)
	if err != nil {
		log.Fatalf("could not predict text: %v", err)
	}
	log.Printf("%#v\n", textPrediction)
}
```

A basic implementation of this usage can be found in the example directory.

## License

Distributed under the MIT license. See the [LICENSE](https://github.com/crossi36/applymagicsauce/blob/master/LICENSE) file for details.
