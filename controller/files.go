package controller

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// FilesHandler is a web handler
func FilesHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
	}

	ctl := FilesController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "POST"})
		return
	case "POST":
		ctl.Create(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// FileHandler is a web handler
func FileHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := FileController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET"})
		return
	case "GET":
		ctl.Read(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// FilesController is a web controller
type FilesController struct{}

// Create handles POST
func (ctl *FilesController) Create(c *models.Context) {
	if c.Auth.UserID < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	mr, err := c.Request.MultipartReader()
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Only multipart forms can be posted: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	var files []models.FileMetadataType

	part, err := mr.NextPart()
	for err == nil {
		// FormName() is only populated if the part has
		// Content-Disposition set to "form-data"
		if part.FormName() != "" {
			if part.FileName() != "" {

				// Persist file and metadata
				md := models.FileMetadataType{}
				md.FileName = part.FileName()
				md.Created = time.Now()
				md.MimeType = part.Header.Get("Content-Type")

				md.Content, err = io.ReadAll(part)
				if err != nil {
					glog.Errorf("+%v", err)
					c.RespondWithErrorMessage(
						fmt.Sprintf("Couldn't not read form part: %v", err.Error()),
						http.StatusBadRequest,
					)
					return
				}

				sha1, err := h.SHA1(md.Content)
				if err != nil {
					glog.Errorf("+%v", err)
					c.RespondWithErrorMessage(
						fmt.Sprintf("Couldn't generate SHA-1: %v", err.Error()),
						http.StatusInternalServerError,
					)
					return
				}
				md.FileHash = sha1
				md.FileSize = int32(len(md.Content))

				// Check whether the file size overflowed int32 (over 2GB)
				if int(md.FileSize) != len(md.Content) {
					c.RespondWithErrorMessage(
						"File too large. Max size = 2147482548 bytes (2GB)",
						http.StatusInternalServerError,
					)
					return
				}

				// Resize if needed
				query := c.Request.URL.Query()

				var (
					maxWidth  int64
					maxHeight int64
				)
				if query.Get("maxWidth") != "" {
					max, err := strconv.ParseInt(strings.Trim(query.Get("maxWidth"), " "), 10, 64)
					if err != nil || max < 0 {
						c.RespondWithErrorMessage("maxWidth needs to be a positive integer", http.StatusBadRequest)
						return
					}
					maxWidth = max
				}
				if query.Get("maxHeight") != "" {
					max, err := strconv.ParseInt(strings.Trim(query.Get("maxHeight"), " "), 10, 64)
					if err != nil || max < 0 {
						c.RespondWithErrorMessage("maxHeight needs to be a positive integer", http.StatusBadRequest)
						return
					}
					maxHeight = max
				}

				status, err := md.Insert(maxWidth, maxHeight)
				if err != nil {
					glog.Errorf("+%v", err)
					c.RespondWithErrorMessage(
						fmt.Sprintf("Couldn't upload file and metadata: %v", err.Error()),
						status,
					)
					return
				}
				files = append(files, md)
			}
		}
		part, err = mr.NextPart()
	}

	c.RespondWithData(files)
}

// FileController is a web controller
type FileController struct{}

// Read handles GET
func (ctl *FileController) Read(c *models.Context) {
	fileHash := c.RouteVars["fileHash"]
	if fileHash == "" {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The supplied file hash cannot be zero characters: %s", c.RouteVars["fileHash"]),
			http.StatusBadRequest,
		)
		return
	}

	fileBytes, headers, _, err := models.GetFile(fileHash)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not retrieve file: %v", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	oneYear := time.Hour * 24 * 365
	nextYear := time.Now().Add(oneYear)
	c.ResponseWriter.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, immutable", oneYear/time.Second))
	c.ResponseWriter.Header().Set("Expires", nextYear.Format(time.RFC1123))

	mimetype := mime.TypeByExtension("." + c.RouteVars["fileExt"])
	if mimetype != "" {
		c.ResponseWriter.Header().Set("Content-Type", mimetype)
	}

	for h, v := range headers {
		c.ResponseWriter.Header().Set(h, v)
	}

	c.WriteResponse(fileBytes, http.StatusOK)
}
