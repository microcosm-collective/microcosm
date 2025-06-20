package models

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/golang/glog"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rwcarlsen/goexif/exif"

	"git.dee.kitchen/buro9/exifutil"
	conf "github.com/microcosm-collective/microcosm/config"
	h "github.com/microcosm-collective/microcosm/helpers"
)

const (
	// AvatarMaxWidth is the maximum width of an avatar
	AvatarMaxWidth int64 = 100

	// AvatarMaxHeight is the maximum height of an avatar
	AvatarMaxHeight int64 = 100

	// MaxFileSize is the maximum size (in bytes) of an attachment
	MaxFileSize int32 = 5242880 * 2 * 3 // 30MB

	// ImageGifMimeType is the mime type for GIF images
	ImageGifMimeType string = "image/gif"

	// ImageJpegMimeType is the mime type for JPG images
	ImageJpegMimeType string = "image/jpeg"

	// ImagePngMimeType is the mime type for PNG images
	ImagePngMimeType string = "image/png"

	// ImageSvgMimeType is the mime type for SVG images
	ImageSvgMimeType string = "image/svg+xml"
)

// FileMetadataType represents the 'attachment_meta' table
type FileMetadataType struct {
	AttachmentMetaID        int64         `json:"-"`
	Created                 time.Time     `json:"created"`
	FileName                string        `json:"fileName"`
	FileExt                 string        `json:"fileExt"`
	FileSize                int32         `json:"fileSize"`
	FileHash                string        `json:"fileHash"`
	MimeType                string        `json:"mimeType"`
	WidthNullable           sql.NullInt64 `json:"-"`
	Width                   int64         `json:"width,omitempty"`
	HeightNullable          sql.NullInt64 `json:"-"`
	Height                  int64         `json:"height,omitempty"`
	ThumbnailWidthNullable  sql.NullInt64 `json:"-"`
	ThumbnailWidth          int64         `json:"thumbnailHeight,omitempty"`
	ThumbnailHeightNullable sql.NullInt64 `json:"-"`
	ThumbnailHeight         int64         `json:"thumbnailWidth,omitempty"`
	AttachCount             int64         `json:"-"`
	Content                 []byte        `json:"-"`
}

// Validate returns true of the file metadata is valid
func (f *FileMetadataType) Validate() (int, error) {

	if f.Created.IsZero() {
		return http.StatusBadRequest, fmt.Errorf("created time must be set")
	}

	if f.FileSize < 1 {
		return http.StatusBadRequest,
			fmt.Errorf("file size (in bytes) must be set")
	}

	if f.FileSize > MaxFileSize {
		return http.StatusBadRequest,
			fmt.Errorf("files must be below 30MB in size")
	}

	// SHA-1 output encoded as string is 40 characters
	if f.FileHash == "" || len(f.FileHash) != 40 {
		return http.StatusBadRequest,
			fmt.Errorf("file hash (SHA-1) must be set")
	}

	if f.MimeType == "" {
		return http.StatusBadRequest, fmt.Errorf("file mime type must be set")
	}

	if f.Width < 0 {
		return http.StatusBadRequest,
			fmt.Errorf("width must be a positive integer, if set")
	}

	if f.Height < 0 {
		return http.StatusBadRequest,
			fmt.Errorf("height must be a positive integer, if set")
	}

	if f.ThumbnailWidth < 0 {
		return http.StatusBadRequest,
			fmt.Errorf("thumbnail width must be a positive integer, if set")
	}

	if f.ThumbnailHeight < 0 {
		return http.StatusBadRequest,
			fmt.Errorf("thumbnail height must be a positive integer, if set")
	}

	if f.AttachCount < 0 {
		return http.StatusBadRequest,
			fmt.Errorf("attach count must be a positive integer, if set")
	}

	return http.StatusOK, nil
}

// Insert saves a file metadata to the database
func (f *FileMetadataType) Insert(
	maxWidth int64,
	maxHeight int64,
) (
	int,
	error,
) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := f.insert(tx, maxWidth, maxHeight)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, fmt.Errorf("transaction failed")
	}

	return http.StatusOK, nil
}

// Insert uploads the file to S3 and inserts the metadata into attachment_meta
func (f *FileMetadataType) insert(
	q h.Er,
	maxWidth int64,
	maxHeight int64,
) (
	int,
	error,
) {
	// Validation has to be performed on images that have already been processed
	// according to their EXIF info (rotated if necessary), so we have to do a
	// load of work to determine info about the file to upload to figure out
	// whether we have an image and can process it, before we are able to call
	// the validate method.
	fileNameBits := strings.Split(f.FileName, ".")
	f.FileExt = "unk"
	if len(fileNameBits) > 0 {
		f.FileExt = fileNameBits[len(fileNameBits)-1]
	}

	var isImage bool
	switch strings.ToLower(f.MimeType) {
	case "application/octet-stream":
		switch f.FileExt {
		case "gif":
			f.MimeType = ImageGifMimeType
			isImage = true
		case "jpeg":
			f.MimeType = ImageJpegMimeType
			isImage = true
		case "jpg":
			f.MimeType = ImageJpegMimeType
			isImage = true
		case "png":
			f.MimeType = ImagePngMimeType
			isImage = true
		case "svg":
			f.MimeType = ImageSvgMimeType
		}
	case ImageGifMimeType:
		f.FileExt = "gif"
		isImage = true
	case ImageJpegMimeType:
		f.FileExt = "jpg"
		isImage = true
	case ImagePngMimeType:
		f.FileExt = "png"
		isImage = true
	case ImageSvgMimeType:
		f.FileExt = "svg"
	}

	if isImage {
		// See image format imports above for supported image types
		// If a match is not made, we assume the upload is bad
		im, format, err := image.DecodeConfig(bytes.NewReader(f.Content))
		if err != nil {
			glog.Warningf(
				"image.DecodeConfig(bytes.NewReader(f.Content)) %+v",
				err,
			)
			return http.StatusBadRequest, err
		}
		f.Height = int64(im.Height)
		f.Width = int64(im.Width)

		// Resize if we've been told the image must fit within a certain size
		if (maxWidth > 0 && f.Width > maxWidth) ||
			(maxHeight > 0 && f.Height > maxHeight) {

			status, err := f.ResizeImage(maxWidth, maxHeight)
			if err != nil {
				glog.Errorf(
					"f.ResizeImage(%d, %d), %+v",
					maxWidth,
					maxHeight,
					err,
				)
				return status, err
			}
		}

		switch format {
		case "gif":
			f.MimeType = ImageGifMimeType
		case "jpeg":
			f.MimeType = ImageJpegMimeType
		case "jpg":
			f.MimeType = ImageJpegMimeType
		case "png":
			f.MimeType = ImagePngMimeType
		}

		// If the image is a jpeg, process the exif data, replace the image,
		// and update the width and height as necessary.
		if f.MimeType == ImageJpegMimeType {
			err := f.processExif()
			if err != nil {
				glog.Errorf("Error processing exif data: %s", err)
			}
		}
	}

	status, err := f.Validate()
	if err != nil {
		return status, err
	}

	meta, status, err := GetMetadata(f.FileHash)
	// File metadata exists meaning we've already uploaded it, since this
	// upload is idempotent, simply return 'OK'
	if err == nil {
		f.AttachmentMetaID = meta.AttachmentMetaID
		return http.StatusOK, nil
	}

	// An error other than 404 occurred
	if status != http.StatusNotFound {
		glog.Errorf("GetMetadata(`%s`) %+v", f.FileHash, err)
		return status, err
	}

	if err := filePut(f.FileHash, f.Content, f.MimeType); err != nil {
		glog.Errorf(
			"filePut(`%s`, f.Content, `%s`,) %+v",
			f.FileHash,
			f.MimeType,
			err,
		)
		return http.StatusInternalServerError, err
	}

	// File is now uploaded, but we haven't stored metadata for it yet.
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertID int64
	err = q.QueryRow(`
INSERT INTO attachment_meta (
    created, file_size, file_sha1, mime_type, width,
    height, thumbnail_width, thumbnail_height, attach_count, file_name,
    file_ext
) VALUES (
    $1, $2, $3, $4, $5
   ,$6, $7, $8, $9, $10
   ,$11
) RETURNING attachment_meta_id`,
		f.Created,
		f.FileSize,
		f.FileHash,
		f.MimeType,
		f.Width,
		f.Height,
		f.ThumbnailWidth,
		f.ThumbnailHeight,
		f.AttachCount,
		f.FileName,
		f.FileExt,
	).Scan(
		&insertID,
	)
	if err != nil {
		glog.Errorf("row.Scan() %+v", err)
		return http.StatusInternalServerError,
			fmt.Errorf("error inserting data and returning ID")
	}
	f.AttachmentMetaID = insertID

	return http.StatusOK, nil
}

// Update saves changes to to a file
func (f *FileMetadataType) Update() (int, error) {
	status, err := f.Validate()
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE attachment_meta
   SET created = $1
      ,file_size = $2
      ,file_sha1 = $3
      ,mime_type = $4
      ,width = $5
      ,height = $6
      ,thumbnail_width = $7
      ,thumbnail_height = $8
      ,attach_count = $9
      ,file_name = $10
      ,file_ext = $11
 WHERE attachment_meta_id = $12`,
		f.Created,
		f.FileSize,
		f.FileHash,
		f.MimeType,
		f.Width,
		f.Height,
		f.ThumbnailWidth,
		f.ThumbnailHeight,
		f.AttachCount,
		f.FileName,
		f.FileExt,
		f.AttachmentMetaID,
	)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError,
			fmt.Errorf("could not update attachment metadata: %+v", err)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %+v", err)
	}

	return http.StatusOK, nil
}

// GetFile retrieves a file by its file hash
func GetFile(fileHash string) ([]byte, map[string]string, int, error) {
	content, headers, err := fileGet(fileHash)
	if err != nil {
		return content, headers, http.StatusInternalServerError, err
	}
	return content, headers, http.StatusOK, nil
}

// GetMetadata returns a file's metadata by it's hash
func GetMetadata(fileHash string) (FileMetadataType, int, error) {
	db, err := h.GetConnection()
	if err != nil {
		return FileMetadataType{}, http.StatusInternalServerError, err
	}

	var m FileMetadataType
	err = db.QueryRow(`
SELECT m.attachment_meta_id
      ,m.created
      ,m.file_size
      ,m.file_sha1
      ,m.mime_type
      ,m.width
      ,m.height
      ,m.thumbnail_width
      ,m.thumbnail_height
      ,m.attach_count
      ,m.file_name
      ,m.file_ext
  FROM attachment_meta m
 WHERE m.file_sha1 = $1`,
		fileHash,
	).Scan(
		&m.AttachmentMetaID,
		&m.Created,
		&m.FileSize,
		&m.FileHash,
		&m.MimeType,
		&m.WidthNullable,
		&m.HeightNullable,
		&m.ThumbnailWidthNullable,
		&m.ThumbnailHeightNullable,
		&m.AttachCount,
		&m.FileName,
		&m.FileExt,
	)
	if err == sql.ErrNoRows {
		return FileMetadataType{}, http.StatusNotFound,
			fmt.Errorf("file metadata with hash %s not found", fileHash)

	}
	if err != nil {
		return FileMetadataType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}

	if m.WidthNullable.Valid {
		m.Width = m.WidthNullable.Int64
	}

	if m.HeightNullable.Valid {
		m.Height = m.HeightNullable.Int64
	}

	if m.ThumbnailWidthNullable.Valid {
		m.ThumbnailWidth = m.ThumbnailWidthNullable.Int64
	}

	if m.ThumbnailHeightNullable.Valid {
		m.ThumbnailHeight = m.ThumbnailHeightNullable.Int64
	}

	return m, http.StatusOK, nil
}

// ResizeImage will resize an image (usually an avatar) to fit within the given
// constraints whilst preserving the aspect ratio
func (f *FileMetadataType) ResizeImage(
	maxWidth int64,
	maxHeight int64,
) (
	int,
	error,
) {
	var (
		width  int
		height int
	)

	if maxWidth > 0 && f.Width > maxWidth {
		width = int(maxWidth)
	}

	if maxHeight > 0 && f.Height > maxHeight && f.Height > f.Width {
		width = 0
		height = int(maxHeight)
	}

	if width == 0 && height == 0 {
		// Nothing to do, either the params weren't supplied or the image is
		// already small enough
		return http.StatusOK, nil
	}

	r := bytes.NewReader(f.Content)

	// middle var is format, i.e. which decoder was used: "gif", "jpeg", "png"
	// in the case of "gif", only the first frame is extracted
	img, format, err := image.Decode(r)
	if err != nil {
		glog.Errorf("image.Decode(r) %+v", err)
		return http.StatusBadRequest, err
	}

	m := imaging.Resize(img, width, height, imaging.Lanczos)

	var buf bytes.Buffer

	switch format {
	case "gif":
		err = gif.Encode(&buf, m, nil)
		if err != nil {
			glog.Errorf("gif.Encode(&buf, m, nil) %+v", err)
			return http.StatusBadRequest, err
		}
		f.MimeType = ImageGifMimeType
	case "jpeg":
		err = jpeg.Encode(&buf, m, nil)
		if err != nil {
			glog.Errorf("jpeg.Encode(&buf, m, nil) %+v", err)
			return http.StatusBadRequest, err
		}
		f.MimeType = ImageJpegMimeType
	default:
		err = png.Encode(&buf, m)
		if err != nil {
			glog.Errorf("png.Encode(&buf, m, nil) %+v", err)
			return http.StatusBadRequest, err
		}
		f.MimeType = ImagePngMimeType
	}

	// Update the file meta data
	f.Content = buf.Bytes()

	sha1, err := h.SHA1(f.Content)
	if err != nil {
		glog.Errorf("h.Sha1(f.Content) %+v", err)
		return http.StatusInternalServerError,
			fmt.Errorf("couldn't generate SHA-1")
	}
	f.FileHash = sha1
	f.FileSize = int32(len(f.Content))

	im, _, err := image.DecodeConfig(bytes.NewReader(f.Content))
	if err != nil {
		glog.Errorf(
			"image.DecodeConfig(bytes.NewReader(f.Content)) %+v",
			err,
		)
		return http.StatusInternalServerError, err
	}
	f.Height = int64(im.Height)
	f.Width = int64(im.Width)

	return http.StatusOK, nil
}

// processExif attempts to rotate a JPEG based on the exif data. If the exif data
// cannot be decoded or the orientation tag not read, we return nil so that the image
// may continue to be uploaded. If there is an error encoding the image after
// modification, this is returned to the caller.
func (f *FileMetadataType) processExif() error {
	// Decode exif.
	ex, err := exif.Decode(bytes.NewReader(f.Content))
	if err != nil {
		return nil
	}
	// Get orientation tag.
	tag, err := ex.Get(exif.Orientation)
	if err != nil {
		return nil
	}
	orientation, err := tag.Int(0)
	if err != nil {
		return nil
	}

	var (
		angle            int
		flipMode         exifutil.FlipDirection
		switchDimensions bool
	)

	angle, flipMode, switchDimensions = exifutil.ProcessOrientation(int64(orientation))

	im, _, err := image.Decode(bytes.NewReader(f.Content))
	if err != nil {
		return err
	}

	if angle != 0 {
		im = exifutil.Rotate(im, angle)
	}

	if flipMode != 0 {
		im = exifutil.Flip(im, flipMode)
	}

	if switchDimensions {
		f.Width, f.Height = f.Height, f.Width
	}

	// Encode JPEG and replace f.Content.
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, im, nil)
	if err != nil {
		return err
	}
	f.Content = buf.Bytes()

	// Update the hash and filesize based on changed content.
	sha1, err := h.SHA1(f.Content)
	if err != nil {
		return err
	}
	f.FileHash = sha1
	f.FileSize = int32(len(f.Content))

	return nil
}

func filePut(fileHash string, content []byte, mimeType string) error {
	var (
		endpoint        = conf.ConfigStrings[conf.S3Endpoint]
		bucketName      = conf.ConfigStrings[conf.S3BucketName]
		accessKeyID     = conf.ConfigStrings[conf.S3AccessKeyID]
		secretAccessKey = conf.ConfigStrings[conf.S3SecretAccessKey]
	)

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false, // of course
	})
	if err != nil {
		return err
	}

	// check whether the object already exists in the bucket
	_, err = client.StatObject(context.Background(), bucketName, fileHash, minio.StatObjectOptions{})
	if err == nil {
		// it exists... we're done
		return nil
	}

	// it does not exist, so we're uploading it
	_, err = client.PutObject(
		context.Background(),
		bucketName,
		fileHash,                 // object name
		bytes.NewReader(content), // object bytes
		int64(len(content)),      // object size
		minio.PutObjectOptions{
			ContentType: mimeType, // mime type
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func fileGet(fileHash string) ([]byte, map[string]string, error) {
	var (
		content = []byte{}
		headers = map[string]string{}
	)

	var (
		endpoint        = conf.ConfigStrings[conf.S3Endpoint]
		bucketName      = conf.ConfigStrings[conf.S3BucketName]
		accessKeyID     = conf.ConfigStrings[conf.S3AccessKeyID]
		secretAccessKey = conf.ConfigStrings[conf.S3SecretAccessKey]
	)

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true, // of course
	})
	if err != nil {
		return content, headers, err
	}

	// get the file
	reader, err := client.GetObject(context.Background(), bucketName, fileHash, minio.GetObjectOptions{})
	if err != nil {
		return content, headers, err
	}
	defer reader.Close()

	return readerToBytes(reader), headers, nil
}

func readerToBytes(reader io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.Bytes()
}
