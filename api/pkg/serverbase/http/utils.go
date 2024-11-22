package sbhttp

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
)

func ReturnHttpError(w http.ResponseWriter, err, defaultErr *lhttp.HttpError) {
	if err.Err != nil {
		if defaultErr != nil {
			ReturnError(w, defaultErr.Code, defaultErr.Message, err.Err)
		} else {
			ReturnError(w, http.StatusInternalServerError, "Internal server error", err.Err)
		}
	} else {
		ReturnError(w, err.Code, err.Message, err)
	}
}

func ReturnError(w http.ResponseWriter, code int, message string, err error) {
	http.Error(w, message, code)
}

func WriteJson(w http.ResponseWriter, code int, result interface{}) error {
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		w.Write([]byte("error serializing response"))
		return err
	}
	return nil
}

// ReturnFile returns a file on disk in an response
func ReturnFile(writer http.ResponseWriter, filename string) error {

	f, err := os.Open(filename)
	if err != nil {
		//File not found, send 404
		http.Error(writer, "File not found.", 404)
		return err
	}

	defer f.Close() //Close after function return

	//Get the Content-Type of the file
	//Create a buffer to store the header of the file in
	fileHeader := make([]byte, 512)
	//Copy the headers into the FileHeader buffer
	f.Read(fileHeader)
	//Get content type of file
	fileContentType := http.DetectContentType(fileHeader)

	//Get the file size
	fileStat, err := f.Stat() //Get info from file
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(err.Error()))
		return err
	}
	FileSize := strconv.FormatInt(fileStat.Size(), 10) //Get file size as a string

	//Send the headers
	writer.Header().Set("Content-Disposition", "attachment; filename="+f.Name())
	writer.Header().Set("Content-Type", fileContentType)
	writer.Header().Set("Content-Length", FileSize)

	//Send the file
	//We read 512 bytes from the file already, so we reset the offset back to 0
	f.Seek(0, 0)
	io.Copy(writer, f) //'Copy' the file to the client
	return nil
}
