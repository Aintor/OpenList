package aliyun_oss

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/http"
	stdpath "path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/errs"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	log "github.com/sirupsen/logrus"
)

type AliyunOSS struct {
	model.Storage
	Addition
	config             driver.Config
	bucket             *oss.Bucket
	linkBucket         *oss.Bucket
	directUploadBucket *oss.Bucket
}

func (d *AliyunOSS) Config() driver.Config {
	return d.config
}

func (d *AliyunOSS) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *AliyunOSS) Init(ctx context.Context) error {
	var err error
	d.bucket, err = newBucket(d.Endpoint, d.Bucket, d.AccessKeyID, d.SecretAccessKey, d.SecurityToken, false)
	if err != nil {
		return fmt.Errorf("failed to initialize Aliyun OSS bucket: %w", err)
	}

	if d.CustomHost != "" {
		d.linkBucket, err = newBucket(d.CustomHost, d.Bucket, d.AccessKeyID, d.SecretAccessKey, d.SecurityToken, true)
		if err != nil {
			return fmt.Errorf("failed to initialize custom host bucket: %w", err)
		}
	} else {
		d.linkBucket = d.bucket
	}

	if d.DirectUploadHost != "" {
		d.directUploadBucket, err = newBucket(d.DirectUploadHost, d.Bucket, d.AccessKeyID, d.SecretAccessKey, d.SecurityToken, true)
		if err != nil {
			return fmt.Errorf("failed to initialize direct upload host bucket: %w", err)
		}
	} else if d.CustomHost != "" {
		d.directUploadBucket, err = newBucket(d.CustomHost, d.Bucket, d.AccessKeyID, d.SecretAccessKey, d.SecurityToken, true)
		if err != nil {
			return fmt.Errorf("failed to initialize custom host direct upload bucket: %w", err)
		}
	} else {
		d.directUploadBucket = nil
	}

	return nil
}

func (d *AliyunOSS) Drop(ctx context.Context) error {
	return nil
}

func (d *AliyunOSS) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	dirPath := dir.GetPath()
	prefix := getKey(dirPath, true)
	var files []model.Obj
	var continuationToken string

	for {
		var options []oss.Option
		options = append(options, oss.Prefix(prefix), oss.Delimiter("/"))
		if continuationToken != "" {
			options = append(options, oss.ContinuationToken(continuationToken))
		}

		res, err := d.bucket.ListObjectsV2(options...)
		if err != nil {
			return nil, err
		}

		for _, p := range res.CommonPrefixes {
			name := stdpath.Base(strings.Trim(p, "/"))
			files = append(files, &model.Object{
				Path:     stdpath.Join(dirPath, name),
				Name:     name,
				Modified: d.Modified,
				IsFolder: true,
			})
		}

		for _, o := range res.Objects {
			if strings.HasSuffix(o.Key, "/") {
				continue
			}
			name := stdpath.Base(o.Key)
			if !args.S3ShowPlaceholder && (name == ".openlist" || name == ".alist" || (d.Placeholder != "" && name == d.Placeholder)) {
				continue
			}
			file := &model.Object{
				Path:     stdpath.Join(dirPath, name),
				Name:     name,
				Size:     o.Size,
				Modified: o.LastModified,
			}
			files = append(files, d.wrapWithThumb(ctx, dirPath, name, file))
		}

		if !res.IsTruncated {
			break
		}
		continuationToken = res.NextContinuationToken
	}

	return files, nil
}

func (d *AliyunOSS) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	key := getKey(file.GetPath(), false)
	fileName := stdpath.Base(key)
	expireSeconds := int64(time.Hour * time.Duration(d.SignURLExpire) / time.Second)

	var options []oss.Option
	if args.Type == "thumb" && (d.ThumbQueryValue != "" || d.ImageThumbValue != "") {
		ext := strings.ToLower(utils.Ext(fileName))
		isImg := utils.SliceContains([]string{"jpg", "jpeg", "png", "gif", "webp", "bmp"}, ext)
		isVid := utils.SliceContains([]string{"mp4", "mkv", "avi", "mov", "flv", "webm", "3gp", "wmv"}, ext)
		var val string
		if isImg {
			val = d.ImageThumbValue
		} else if isVid {
			val = d.ThumbQueryValue
		}
		if val != "" {
			options = append(options, oss.Process(val))
		}
	} else {
		disposition := utils.GenerateContentDisposition(fileName)
		options = append(options, oss.ResponseContentDisposition(disposition))
	}

	linkURL, err := d.linkBucket.SignURL(key, oss.HTTPGet, expireSeconds, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to sign URL: %w", err)
	}

	if parts := strings.SplitN(linkURL, "?", 2); len(parts) == 2 {
		linkURL = strings.ReplaceAll(parts[0], "%2F", "/") + "?" + parts[1]
	}

	return &model.Link{
		URL: linkURL,
	}, nil
}

func (d *AliyunOSS) Get(ctx context.Context, path string) (model.Obj, error) {
	rootPath := d.GetRootPath()
	if rootPath != "/" && utils.IsSubPath(rootPath, path) {
	} else {
		path = stdpath.Join(rootPath, path)
	}

	key := getKey(path, false)
	meta, err := d.bucket.GetObjectDetailedMeta(key)
	if err == nil {
		size, _ := strconv.ParseInt(meta.Get("Content-Length"), 10, 64)
		modified, _ := time.Parse(http.TimeFormat, meta.Get("Last-Modified"))
		fileName := stdpath.Base(path)
		return &model.Object{
			Name:     fileName,
			Size:     size,
			Modified: modified,
			Path:     path,
		}, nil
	}

	prefix := getKey(path, true)
	res, err := d.bucket.ListObjectsV2(oss.Prefix(prefix), oss.MaxKeys(1))
	if err == nil && (len(res.Objects) > 0 || len(res.CommonPrefixes) > 0) {
		return &model.Object{
			Name:     stdpath.Base(path),
			Modified: d.Modified,
			IsFolder: true,
			Path:     path,
		}, nil
	}

	return nil, errs.ObjectNotFound
}

func (d *AliyunOSS) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) error {
	dirKey := getKey(stdpath.Join(parentDir.GetPath(), dirName), true)
	return d.bucket.PutObject(dirKey, bytes.NewReader([]byte{}))
}

func (d *AliyunOSS) Move(ctx context.Context, srcObj, dstDir model.Obj) error {
	if err := d.Copy(ctx, srcObj, dstDir); err != nil {
		return err
	}
	return d.Remove(ctx, srcObj)
}

func (d *AliyunOSS) Rename(ctx context.Context, srcObj model.Obj, newName string) error {
	srcKey := getKey(srcObj.GetPath(), srcObj.IsDir())
	dstKey := getKey(stdpath.Join(stdpath.Dir(srcObj.GetPath()), newName), srcObj.IsDir())
	if srcObj.IsDir() {
		if err := d.copyDir(srcKey, dstKey); err != nil {
			return err
		}
	} else {
		if _, err := d.bucket.CopyObject(srcKey, dstKey); err != nil {
			return err
		}
	}
	return d.Remove(ctx, srcObj)
}

func (d *AliyunOSS) Copy(ctx context.Context, srcObj, dstDir model.Obj) error {
	srcKey := getKey(srcObj.GetPath(), srcObj.IsDir())
	dstKey := getKey(stdpath.Join(dstDir.GetPath(), srcObj.GetName()), srcObj.IsDir())
	if srcObj.IsDir() {
		return d.copyDir(srcKey, dstKey)
	}
	_, err := d.bucket.CopyObject(srcKey, dstKey)
	return err
}

func (d *AliyunOSS) copyDir(srcPrefix, dstPrefix string) error {
	marker := ""
	for {
		res, err := d.bucket.ListObjects(oss.Prefix(srcPrefix), oss.Marker(marker))
		if err != nil {
			return err
		}
		for _, o := range res.Objects {
			newKey := dstPrefix + strings.TrimPrefix(o.Key, srcPrefix)
			if _, err := d.bucket.CopyObject(o.Key, newKey); err != nil {
				return err
			}
		}
		if !res.IsTruncated {
			break
		}
		marker = res.NextMarker
	}
	return nil
}

func (d *AliyunOSS) Remove(ctx context.Context, obj model.Obj) error {
	if obj.IsDir() {
		prefix := getKey(obj.GetPath(), true)
		marker := ""
		for {
			res, err := d.bucket.ListObjects(oss.Prefix(prefix), oss.Marker(marker))
			if err != nil {
				return err
			}
			var keys []string
			for _, o := range res.Objects {
				keys = append(keys, o.Key)
			}
			if len(keys) > 0 {
				if _, err := d.bucket.DeleteObjects(keys); err != nil {
					return err
				}
			}
			if !res.IsTruncated {
				break
			}
			marker = res.NextMarker
		}
		return nil
	}

	key := getKey(obj.GetPath(), false)
	return d.bucket.DeleteObject(key)
}

func (d *AliyunOSS) Put(ctx context.Context, dstDir model.Obj, s model.FileStreamer, up driver.UpdateProgress) error {
	key := getKey(stdpath.Join(dstDir.GetPath(), s.GetName()), false)
	log.Debugln("AliyunOSS Put key:", key)

	reader := driver.NewLimitedUploadStream(ctx, &driver.ReaderUpdatingProgress{
		Reader:         s,
		UpdateProgress: up,
	})

	options := []oss.Option{
		oss.ContentType(s.GetMimetype()),
	}

	return d.bucket.PutObject(key, reader, options...)
}

func (d *AliyunOSS) GetDirectUploadTools() []string {
	if !d.EnableDirectUpload {
		return nil
	}
	return []string{"HttpDirect"}
}

const (
	directMultipartThreshold = 20 * 1024 * 1024 // 20MB
	minDirectChunkSize       = 10 * 1024 * 1024 // 10MB
	maxDirectMultipartParts  = 500
)

func (d *AliyunOSS) GetDirectUploadInfo(ctx context.Context, _ string, dstDir model.Obj, fileName string, fileSize int64) (any, error) {
	if !d.EnableDirectUpload {
		return nil, errs.NotImplement
	}
	key := getKey(stdpath.Join(dstDir.GetPath(), fileName), false)
	expireSeconds := int64(time.Hour * time.Duration(d.SignURLExpire) / time.Second)

	contentType := mime.TypeByExtension(strings.ToLower(stdpath.Ext(fileName)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// For large files (> 20MB), use OSS Native Multipart Upload
	if fileSize > directMultipartThreshold {
		chunkSize := int64(minDirectChunkSize)
		if (fileSize+chunkSize-1)/chunkSize > maxDirectMultipartParts {
			chunkSize = (fileSize + maxDirectMultipartParts - 1) / maxDirectMultipartParts
		}
		totalParts := int((fileSize + chunkSize - 1) / chunkSize)

		imur, err := d.bucket.InitiateMultipartUpload(key, oss.ContentType(contentType))
		if err != nil {
			return nil, fmt.Errorf("failed to initiate multipart upload: %w", err)
		}

		parts := make([]model.MultipartDirectUploadPart, 0, totalParts)
		partContentType := "application/octet-stream"
		for i := 1; i <= totalParts; i++ {
			partNumberStr := strconv.Itoa(i)
			options := []oss.Option{
				oss.ContentType(partContentType),
				oss.AddParam("partNumber", partNumberStr),
				oss.AddParam("uploadId", imur.UploadID),
			}
			partUploadURL, err := d.bucket.SignURL(key, oss.HTTPPut, expireSeconds, options...)
			if err != nil {
				_ = d.bucket.AbortMultipartUpload(imur)
				return nil, fmt.Errorf("failed to sign multipart upload part %d URL: %w", i, err)
			}
			if p := strings.SplitN(partUploadURL, "?", 2); len(p) == 2 {
				partUploadURL = strings.ReplaceAll(p[0], "%2F", "/") + "?" + p[1]
			}

			partFallbackURL := ""
			if d.directUploadBucket != nil {
				fallbackURL, err := d.directUploadBucket.SignURL(key, oss.HTTPPut, expireSeconds, options...)
				if err == nil {
					if p := strings.SplitN(fallbackURL, "?", 2); len(p) == 2 {
						fallbackURL = strings.ReplaceAll(p[0], "%2F", "/") + "?" + p[1]
					}
					partFallbackURL = fallbackURL
				}
			}

			parts = append(parts, model.MultipartDirectUploadPart{
				PartNumber:  i,
				UploadURL:   partUploadURL,
				FallbackURL: partFallbackURL,
			})
		}

		firstUploadURL := ""
		firstFallbackURL := ""
		if len(parts) > 0 {
			firstUploadURL = parts[0].UploadURL
			firstFallbackURL = parts[0].FallbackURL
		}

		return &model.HttpDirectUploadInfo{
			UploadURL:      firstUploadURL,
			FallbackURL:    firstFallbackURL,
			Method:         "PUT",
			FallbackMethod: "PUT",
			ChunkSize:      chunkSize,
			UploadID:       imur.UploadID,
			Parts:          parts,
			Headers: map[string]string{
				"Content-Type": partContentType,
			},
		}, nil
	}

	options := []oss.Option{
		oss.ContentType(contentType),
	}

	// Primary Direct Upload: Direct connection to OSS source station
	uploadURL, err := d.bucket.SignURL(key, oss.HTTPPut, expireSeconds, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to sign direct upload URL: %w", err)
	}
	if parts := strings.SplitN(uploadURL, "?", 2); len(parts) == 2 {
		uploadURL = strings.ReplaceAll(parts[0], "%2F", "/") + "?" + parts[1]
	}

	res := &model.HttpDirectUploadInfo{
		UploadURL: uploadURL,
		Method:    "PUT",
		Headers: map[string]string{
			"Content-Type": contentType,
		},
	}

	// Secondary Fallback Upload: If direct connection is slow or fails, client directly switches to ESA / accelerated host
	if d.directUploadBucket != nil {
		fallbackURL, err := d.directUploadBucket.SignURL(key, oss.HTTPPut, expireSeconds, options...)
		if err == nil {
			if parts := strings.SplitN(fallbackURL, "?", 2); len(parts) == 2 {
				fallbackURL = strings.ReplaceAll(parts[0], "%2F", "/") + "?" + parts[1]
			}
			res.FallbackURL = fallbackURL
			res.FallbackMethod = "PUT"
		}
	}

	return res, nil
}

func (d *AliyunOSS) CompleteDirectUpload(ctx context.Context, dstDir model.Obj, fileName string, uploadID string, parts []model.DirectUploadPart) error {
	key := getKey(stdpath.Join(dstDir.GetPath(), fileName), false)
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   d.Bucket,
		Key:      key,
		UploadID: uploadID,
	}
	ossParts := make([]oss.UploadPart, 0, len(parts))
	for _, p := range parts {
		cleanETag := strings.Trim(p.ETag, "\"")
		ossParts = append(ossParts, oss.UploadPart{
			PartNumber: p.PartNumber,
			ETag:       fmt.Sprintf("\"%s\"", cleanETag),
		})
	}
	sort.Slice(ossParts, func(i, j int) bool {
		return ossParts[i].PartNumber < ossParts[j].PartNumber
	})
	_, err := d.bucket.CompleteMultipartUpload(imur, ossParts)
	return err
}

func (d *AliyunOSS) AbortDirectUpload(ctx context.Context, dstDir model.Obj, fileName string, uploadID string) error {
	key := getKey(stdpath.Join(dstDir.GetPath(), fileName), false)
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   d.Bucket,
		Key:      key,
		UploadID: uploadID,
	}
	return d.bucket.AbortMultipartUpload(imur)
}

var _ driver.Driver = (*AliyunOSS)(nil)
var _ driver.Getter = (*AliyunOSS)(nil)
var _ driver.DirectUploader = (*AliyunOSS)(nil)
var _ driver.DirectMultipartUploader = (*AliyunOSS)(nil)
