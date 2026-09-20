/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
)

// snapshotFile is the filesystem boundary for preparing an immutable upload.
// Its small interface lets failure tests exercise disk errors without filling a real disk.
type snapshotFile interface {
	io.WriteCloser
	Name() string
	Stat() (os.FileInfo, error)
}

// createBodySnapshot opens the private spool file and provides the filesystem test seam.
var createBodySnapshot = func() (snapshotFile, error) { return os.CreateTemp("", "ctyun-body-*") }

// openBodySource is the regular-source opening seam for testing descriptor failures.
var openBodySource = os.OpenFile

// BodyInput contains resolved values for one explicit request encoder.
type BodyInput struct {
	MemberFields []string
	JSONFields   []string
	Encoding     string
	ContentType  string
	Fields       map[string]any
	Document     string
	Parts        []BodyPart
}

// BodyPart is a resolved multipart scalar or local file path.
type BodyPart struct {
	Name, Value, ContentType string
	File                     bool
}

// PreparedBody owns an immutable byte snapshot and its signing digest.
type PreparedBody struct {
	sensitiveValues []string
	Length          int64
	SHA256          string
	ContentType     string
	path            string
	data            []byte
	closed          bool
}

// Open starts an independent reader at the beginning of the prepared snapshot.
func (body *PreparedBody) Open() (io.ReadCloser, error) {
	if body.closed {
		return nil, apicontract.Invalid("request.closed")
	}
	if body.path != "" {
		return os.Open(body.path)
	}
	return io.NopCloser(bytes.NewReader(body.data)), nil
}

// Close removes the snapshot once all request attempts have completed.
func (body *PreparedBody) Close() error {
	if body.closed {
		return nil
	}
	body.closed = true
	if body.path != "" {
		return os.Remove(body.path)
	}
	return nil
}

// PrepareBody serializes once, hashes the exact bytes, and bounds file-transfer memory.
func PrepareBody(input BodyInput) (_ *PreparedBody, err error) {
	body := &PreparedBody{ContentType: input.ContentType}
	var buffer bytes.Buffer
	var destination io.Writer = &buffer
	var file snapshotFile
	if input.Encoding == "file" || input.Encoding == "multipart" {
		file, err = createBodySnapshot()
		if err != nil {
			return nil, err
		}
		body.path = file.Name()
		destination = file
		defer func() {
			if file != nil {
				// Preserve the primary failure while releasing this resource.
				_ = file.Close()
			}
			if err != nil {
				// Preserve the primary failure while releasing this resource.
				_ = body.Close()
			}
		}()
	}
	hash := sha256.New()
	writer := io.MultiWriter(destination, hash)
	switch input.Encoding {
	case "json":
		if body.ContentType == "" {
			body.ContentType = "application/json"
		}
		if len(input.Fields) != 0 {
			var data []byte
			data, err = json.Marshal(input.Fields)
			if err == nil {
				_, err = writer.Write(data)
			}
		}
	case "xml":
		if body.ContentType == "" {
			body.ContentType = "application/xml"
		}
		if _, err = DecodeXML([]byte(input.Document)); err == nil {
			_, err = io.WriteString(writer, input.Document)
		}
	case "form":
		if body.ContentType == "" {
			body.ContentType = "application/x-www-form-urlencoded"
		}
		values := url.Values{}
		for key, value := range input.Fields {
			if slices.Contains(input.MemberFields, key) {
				if err := encodeFormMembers(values, input.Fields, key, value); err != nil {
					return nil, err
				}
				continue
			}
			if slices.Contains(input.JSONFields, key) {
				encoded, encodeErr := json.Marshal(value)
				if encodeErr != nil {
					return nil, encodeErr
				}
				values.Set(key, string(encoded))
				continue
			}
			switch value.(type) {
			case string, bool, json.Number:
				values.Set(key, fmt.Sprint(value))
			default:
				return nil, apicontract.Invalid("request.form")
			}
		}
		_, err = io.WriteString(writer, values.Encode())
	case "file":
		if body.ContentType == "" {
			body.ContentType = "application/octet-stream"
		}
		err = copyBodyFile(writer, input.Document)
	case "multipart":
		multipartWriter := multipart.NewWriter(writer)
		body.ContentType = multipartWriter.FormDataContentType()
		seen := map[string]bool{}
		for _, part := range input.Parts {
			if seen[strings.ToLower(part.Name)] {
				return nil, apicontract.Invalid("request.parts.duplicate")
			}
			seen[strings.ToLower(part.Name)] = true
			if part.Name == "" || strings.ContainsAny(part.Name, "\r\n\x00") {
				return nil, apicontract.Invalid("request.parts")
			}
			if !part.File && sensitiveFieldName.MatchString(part.Name) {
				body.sensitiveValues = append(body.sensitiveValues, part.Value)
			}
			params := map[string]string{"name": part.Name}
			if part.File {
				params["filename"] = filepath.Base(part.Value)
			}
			header := textproto.MIMEHeader{"Content-Disposition": {mime.FormatMediaType("form-data", params)}}
			if part.ContentType != "" {
				if media, _, parseErr := mime.ParseMediaType(part.ContentType); parseErr != nil || !strings.Contains(media, "/") || strings.ContainsAny(part.ContentType, "\r\n") {
					return nil, apicontract.Invalid("request.parts.content_type")
				}
				header.Set("Content-Type", part.ContentType)
			} else if part.File {
				header.Set("Content-Type", "application/octet-stream")
			}
			var field io.Writer
			field, err = multipartWriter.CreatePart(header)
			if err != nil {
				return nil, err
			}
			if part.File {
				err = copyBodyFile(field, part.Value)
			} else {
				_, err = io.WriteString(field, part.Value)
			}
			if err != nil {
				return nil, err
			}
		}
		err = multipartWriter.Close()
	default:
		return nil, apicontract.Invalid("request.encoding")
	}
	if err != nil {
		return nil, err
	}
	body.SHA256 = hex.EncodeToString(hash.Sum(nil))
	if file != nil {
		var info os.FileInfo
		info, err = file.Stat()
		if err != nil {
			return nil, err
		}
		body.Length = info.Size()
		err = file.Close()
		file = nil
		if err != nil {
			return nil, err
		}
	} else {
		body.data = buffer.Bytes()
		body.Length = int64(len(body.data))
	}
	return body, nil
}

// copyBodyFile accepts only regular files and closes them after snapshot creation.
func copyBodyFile(destination io.Writer, path string) error {
	file, err := OpenRegularFile(path)
	if err != nil {
		return err
	}
	// The source is read-only; io.Copy reports payload read or write failures.
	defer func() { _ = file.Close() }()
	_, err = io.Copy(destination, file)
	return err
}

// OpenRegularFile rejects special files without waiting for FIFO peers.
func OpenRegularFile(path string) (*os.File, error) {
	file, err := openBodySource(path, os.O_RDONLY|regularOpenFlags, 0)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		// Preserve the primary failure while releasing this resource.
		_ = file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		// Preserve the primary failure while releasing this resource.
		_ = file.Close()
		return nil, apicontract.Invalid("request.file")
	}
	return file, nil
}
