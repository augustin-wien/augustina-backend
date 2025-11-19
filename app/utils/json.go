package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/augustin-wien/augustina-backend/config"
	"go.uber.org/zap"
)

// JSONMarshal marshals the data into json without escaping html
// https://stackoverflow.com/a/28596225/19932351
func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	if err != nil {
		log.Error("JSONMarshal: ", err)
		return nil, err
	}
	b := bytes.TrimRight(buffer.Bytes(), "\n")
	return b, err
}

// WriteJSON writes the data to the response writer as json
func WriteJSON(w http.ResponseWriter, status int, data interface{}, wrap ...string) error {
	// out will hold the final version of the json to send to the client
	var out []byte

	// decide if we wrap the json payload in an overall json tag
	if len(wrap) > 0 {
		// wrapper
		wrapper := make(map[string]interface{})
		wrapper[wrap[0]] = data
		jsonBytes, err := JSONMarshal(wrapper)
		if err != nil {
			return err
		}
		out = jsonBytes
	} else {
		// wrapper
		jsonBytes, err := JSONMarshal(data)
		if err != nil {
			return err
		}
		out = jsonBytes
	}

	// set the content type & status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// write the json out
	_, err := w.Write(out)
	if err != nil {
		return err
	}
	return nil
}

type jsonError struct {
	Message string `json:"message"`
}

// ErrorJSON writes an error to the response writer as json
func ErrorJSON(w http.ResponseWriter, err error, status ...int) {
	statusCode := http.StatusBadRequest
	if len(status) > 0 {
		statusCode = status[0]
	}

	theError := jsonError{
		Message: err.Error(),
	}

	err = WriteJSON(w, statusCode, theError, "error")
	if err != nil {
		log.Error("ErrorJSON: ", err)
	}
}

// ReadJSON reads the request body and decodes the json into the data interface
func ReadJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	maxBytes := 1024 * 1024 // one megabyte
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)

	// Use a map to decode into, allowing unknown fields
	var m map[string]interface{}

	if err := dec.Decode(&m); err != nil {
		log.Info(zap.Any("Body of data at ReadJSON", m))
		log.Error("ReadJSON: ", err)
		return err
	}

	// Convert the map to JSON to ignore unknown fields
	jsonData, err := json.Marshal(m)
	if err != nil {
		log.Error("ReadJSON: ", err)
		return err
	}

	// Print the JSON to the log only in development mode
	if config.Config.Development {
		log.Info(zap.String("JSON of data at ReadJSON", string(jsonData)))
	}

	// Unmarshal the JSON back into the provided data structure
	if err := json.Unmarshal(jsonData, data); err != nil {
		log.Error("ReadJSON: ", err)
		return err
	}

	// Make sure only one JSON value in payload
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		log.Error("ReadJSON: ", err)
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}
