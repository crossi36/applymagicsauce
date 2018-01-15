// Package applymagicsauce provides easy access to the API provided by https://applymagicsauce.com.
// For an example scenario please refer to https://applymagicsauce.com/documentation_technical.html
// or the example directory.
package applymagicsauce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiURL = "https://api.applymagicsauce.com"

// APIKey is an optional place to set your APIKey. Normally a call to a prediction endpoint with an
// expired token will fail. However, if you set APIKey this package will try to renew your token
// automatically.
var APIKey string

// Valid keys for the options parameter in the calls to predict functions (PredictLikeIDs or PredictText).
const (
	OptionsSource          = "source"
	OptionsTraits          = "traits"
	OptionsInterpretations = "interpretations"
	OptionsContributors    = "contributors"
)

// Valid values for OptionsSource.
const (
	SourceWebsite      = "WEBSITE"
	SourceEmail        = "EMAIL"
	SourceBrochure     = "BROCHURE"
	SourceStatusUpdate = "STATUS_UPDATE"
	SourceTweet        = "TWEET"
	SourceCV           = "CV"
	SourceOther        = "OTHER"
)

// Token represents the response of the API to the Authentication endpoint.
//
// It looks like they do not use any of the supported RFCs for the "expires" field.
// I have not yet figured out how to parse that field into time.Time, so returning it as int for now.
//
// From documentation:
// "expires": [timestamp when the token expires, integer]
//
// Tokens usually expire after ~1 hour.
type Token struct {
	Token       string   `json:"token"`
	CustomerID  int      `json:"customer_id"`
	Expires     int      `json:"expires"`
	Permissions []string `json:"permissions"`
	UsageLimits []Limits `json:"usage_limits"`
}

// Limits represents the limitations for a Token for the given Method.
//
// CallsAvailableSince can not be parsed into time.Time with any of the supported RFCs. Returning the
// plain value for now.
//
// From documentation:
// "callsAvailableSince": [date and time of last reset, unix timestamp ms]
type Limits struct {
	Method              string `json:"method"`
	CallsLimit          int    `json:"callsLimit"`
	CallsAvailable      int    `json:"callsAvailable"`
	CallsAvailableSince int64  `json:"callsAvailableSince"`
	CallsRenewal        bool   `json:"callsRenewal"`
	CallsRenewalDays    int    `json:"callsRenewalDays"`
}

// Auth uses the passed customerID and apiKey (obtained during registration on https://applymagicsauce.com)
// to get a valid authentication token.
func Auth(customerID int, apiKey string) (authToken *Token, err error) {
	if apiKey == "" && APIKey != "" {
		apiKey = APIKey
	}

	payload := struct {
		CustomerID int    `json:"customer_id"`
		APIKey     string `json:"api_key"`
	}{
		CustomerID: customerID,
		APIKey:     apiKey,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	status, body, err := doRequest("/auth", bytes.NewReader(payloadJSON), nil)
	if err != nil {
		return nil, err
	}

	switch status {
	case http.StatusBadRequest:
		return nil, fmt.Errorf("bad request: %s", body)
	case http.StatusForbidden:
		return nil, fmt.Errorf("authentication failure")
	case http.StatusNotFound:
		return nil, fmt.Errorf("endpoint not found")
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("api is temporarily not available")
	}

	authToken = new(Token)
	err = json.Unmarshal(body, authToken)
	return authToken, err
}

// Predictions represents the result of your call to one of the prediction endpoints (PredictLikeIDs or
// PredictText).
type Predictions struct {
	InputUsed int `json:"input_used"`

	Predictions []struct {
		Trait string  `json:"trait"`
		Value float64 `json:"value"`
	} `json:"predictions"`

	Interpretations []struct {
		Trait string      `json:"trait"`
		Value interface{} `json:"value"`
	} `json:"interpretations"`

	Contributors []struct {
		Trait    string   `json:"trait"`
		Positive []string `json:"positive"`
		Negative []string `json:"negative"`
	} `json:"contributors"`
}

// PredictLikeIDs queries the API with the provided Like IDs and returns the corresponding predictions.
//
// It is advisable to limit the predicted traits to improve overall performance. If you need addtional
// interpretations of the prediction result or information about the contributors you should set the
// appropriate optional parameters.
//
// You can use the PredictLikeIDsOptions function to get a valid representation of these optional
// parameters for your call to PredictLikeIDs.
func PredictLikeIDs(ids []string, options url.Values, auth *Token) (predictions Predictions, err error) {
	payloadJSON, err := json.Marshal(ids)
	if err != nil {
		return predictions, err
	}

	status, body, err := doRequest("/like_ids?"+options.Encode(), bytes.NewReader(payloadJSON), auth)
	if err != nil {
		return predictions, err
	}

	switch status {
	case http.StatusNoContent:
		return predictions, nil
	case http.StatusBadRequest:
		return predictions, fmt.Errorf("bad request: %s", body)
	case http.StatusNotFound:
		return predictions, fmt.Errorf("endpoint not found")
	case http.StatusTooManyRequests:
		return predictions, fmt.Errorf("usage limit exceeded: %s", body)
	case http.StatusInternalServerError:
		return predictions, fmt.Errorf("api is temporarily not available")
	case http.StatusForbidden:
		if APIKey != "" {
			err = renewToken(auth)
			if err != nil {
				return predictions, err
			}
			return PredictLikeIDs(ids, options, auth)
		}
		return predictions, fmt.Errorf("authentication token expired")
	}

	err = json.Unmarshal(body, &predictions)
	return predictions, err
}

// PredictLikeIDsOptions returns a valid options object for use in PredictLikeIDs. All parameters are
// optional. The zero values represent the default behaviour of the API.
func PredictLikeIDsOptions(traits []string, interpretations bool, contributors bool) (options url.Values) {
	options = url.Values{}
	if len(traits) > 0 {
		options.Set(OptionsTraits, strings.Join(traits, ","))
	}
	options.Set(OptionsInterpretations, fmt.Sprintf("%t", interpretations))
	options.Set(OptionsContributors, fmt.Sprintf("%t", contributors))
	return options
}

// PredictText queries the API with the provided text and returns the corresponding predictions.
//
// It is advisable to limit the predicted traits to improve overall performance. If you need addtional
// interpretations of the prediction result you should set the appropriate optional parameter.
//
// You can use the PredictTextOptions function to get a valid representation of these optional
// parameters for your call to PredictText.
//
// ATTENTION: Not all options are optional! See PredictTextOptions for details.
func PredictText(text string, options url.Values, auth *Token) (predictions Predictions, err error) {
	status, body, err := doRequest("/text?"+options.Encode(), strings.NewReader(text), auth)
	if err != nil {
		return predictions, err
	}

	switch status {
	case http.StatusBadRequest:
		return predictions, fmt.Errorf("bad request: %s", body)
	case http.StatusNotFound:
		return predictions, fmt.Errorf("endpoint not found")
	case http.StatusTooManyRequests:
		return predictions, fmt.Errorf("usage limit exceeded: %s", body)
	case http.StatusInternalServerError:
		return predictions, fmt.Errorf("api is temporarily not available")
	case http.StatusForbidden:
		if APIKey != "" {
			err = renewToken(auth)
			if err != nil {
				return predictions, err
			}
			return PredictText(text, options, auth)
		}
		return predictions, fmt.Errorf("authentication token expired")
	}

	err = json.Unmarshal(body, &predictions)
	return predictions, err
}

// PredictTextOptions returns a valid options object for use in PredictText. The source parameter is
// required. All other parameters are optional and the zero values represent the default behaviour
// of the API.
func PredictTextOptions(source string, traits []string, interpretations bool) (options url.Values) {
	options = url.Values{}
	options.Set(OptionsSource, source)
	if len(traits) > 0 {
		options.Set(OptionsTraits, strings.Join(traits, ","))
	}
	options.Set(OptionsInterpretations, fmt.Sprintf("%t", interpretations))
	return options
}

func doRequest(endpoint string, payload io.Reader, token *Token) (statusCode int, body []byte, err error) {
	request, err := http.NewRequest(http.MethodPost, apiURL+endpoint, payload)
	if err != nil {
		return 0, nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if token != nil {
		request.Header.Set("X-Auth-Token", token.Token)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	response, err := client.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()

	body, err = ioutil.ReadAll(response.Body)

	return response.StatusCode, body, err
}

func renewToken(auth *Token) error {
	token, err := Auth(auth.CustomerID, APIKey)
	if err != nil {
		return fmt.Errorf("could not renew authentication token")
	}

	auth.Expires = token.Expires
	auth.Permissions = token.Permissions
	auth.Token = token.Token
	auth.UsageLimits = token.UsageLimits

	return nil
}
