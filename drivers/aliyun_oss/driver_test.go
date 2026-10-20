package aliyun_oss

import (
	"testing"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"
)

func TestDriverRegistration(t *testing.T) {
	d, err := op.GetDriver("AliyunOSS")
	assert.NoError(t, err)
	assert.NotNil(t, d)
	drv := d()
	assert.Equal(t, "AliyunOSS", drv.Config().Name)
}

func TestGetKey(t *testing.T) {
	assert.Equal(t, "folder/file.mp4", getKey("/folder/file.mp4", false))
	assert.Equal(t, "folder/sub/", getKey("/folder/sub", true))
	assert.Equal(t, "folder/sub/", getKey("/folder/sub/", true))
	assert.Equal(t, "", getKey("/", true))
}

func TestFormatEndpoint(t *testing.T) {
	assert.Equal(t, "https://oss-cn-shanghai.aliyuncs.com", formatEndpoint("oss-cn-shanghai.aliyuncs.com"))
	assert.Equal(t, "https://oss-cn-shanghai.aliyuncs.com", formatEndpoint("https://oss-cn-shanghai.aliyuncs.com"))
	assert.Equal(t, "http://oss-cn-shanghai.aliyuncs.com", formatEndpoint("http://oss-cn-shanghai.aliyuncs.com"))
}

func TestLinkAndThumbnail(t *testing.T) {
	drv := &AliyunOSS{
		Addition: Addition{
			Bucket:          "my-bucket",
			Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
			AccessKeyID:     "test_ak",
			SecretAccessKey: "test_sk",
			CustomHost:      "https://cdn.example.com",
			SignURLExpire:   4,
			Thumbnail:       true,
			ThumbQueryValue: "video/snapshot,t_1000,f_jpg,w_300,h_200,m_fast",
			ImageThumbValue: "image/resize,w_300/format,jpg",
		},
	}
	err := drv.Init(nil)
	assert.NoError(t, err)

	// Test video link with thumb
	vidObj := &model.Object{
		Path: "/test/video.mp4",
		Name: "video.mp4",
	}
	thumbLink, err := drv.Link(nil, vidObj, model.LinkArgs{Type: "thumb"})
	assert.NoError(t, err)
	assert.Contains(t, thumbLink.URL, "https://cdn.example.com/test/video.mp4?")
	assert.Contains(t, thumbLink.URL, "x-oss-process=video%2Fsnapshot")
	assert.Contains(t, thumbLink.URL, "OSSAccessKeyId=test_ak")
	assert.Contains(t, thumbLink.URL, "Signature=")

	// Test image link with thumb
	imgObj := &model.Object{
		Path: "/test/picture.jpg",
		Name: "picture.jpg",
	}
	imgThumbLink, err := drv.Link(nil, imgObj, model.LinkArgs{Type: "thumb"})
	assert.NoError(t, err)
	assert.Contains(t, imgThumbLink.URL, "https://cdn.example.com/test/picture.jpg?")
	assert.Contains(t, imgThumbLink.URL, "x-oss-process=image%2Fresize")
	assert.Contains(t, imgThumbLink.URL, "OSSAccessKeyId=test_ak")
	assert.Contains(t, imgThumbLink.URL, "Signature=")

	// Test normal link
	normalLink, err := drv.Link(nil, vidObj, model.LinkArgs{})
	assert.NoError(t, err)
	assert.Contains(t, normalLink.URL, "https://cdn.example.com/test/video.mp4?")
	assert.NotContains(t, normalLink.URL, "x-oss-process")
	assert.Contains(t, normalLink.URL, "response-content-disposition=")

	// Test DirectUploadInfo
	drv.EnableDirectUpload = true
	uploadInfoAny, err := drv.GetDirectUploadInfo(nil, "", vidObj, "video.mp4", 1024)
	assert.NoError(t, err)
	uploadInfo, ok := uploadInfoAny.(*model.HttpDirectUploadInfo)
	assert.True(t, ok)
	assert.Equal(t, "PUT", uploadInfo.Method)
	assert.Contains(t, uploadInfo.UploadURL, "oss-cn-hangzhou.aliyuncs.com")
	assert.Contains(t, uploadInfo.FallbackURL, "https://cdn.example.com")
	// Test small file (< 20MB) returns single PUT
	smallUploadAny, err := drv.GetDirectUploadInfo(nil, "", vidObj, "small.mp4", 10*1024*1024)
	assert.NoError(t, err)
	smallUpload := smallUploadAny.(*model.HttpDirectUploadInfo)
	assert.Empty(t, smallUpload.UploadID)
	assert.Empty(t, smallUpload.Parts)
	assert.NotEmpty(t, smallUpload.UploadURL)
	assert.NotEmpty(t, smallUpload.FallbackURL)
}

func TestMultipartDirectUploadSigning(t *testing.T) {
	drv := &AliyunOSS{
		Addition: Addition{
			Bucket:          "my-bucket",
			Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
			AccessKeyID:     "test_ak",
			SecretAccessKey: "test_sk",
			CustomHost:      "https://cdn.example.com",
			SignURLExpire:   4,
		},
	}
	_ = drv.Init(nil)

	// Verify part URL signing with query parameters and ContentType
	u, err := drv.bucket.SignURL("large.zip", oss.HTTPPut, 3600,
		oss.ContentType("application/octet-stream"),
		oss.AddParam("partNumber", "1"),
		oss.AddParam("uploadId", "test-upload-id"),
	)
	assert.NoError(t, err)
	assert.Contains(t, u, "partNumber=1")
	assert.Contains(t, u, "uploadId=test-upload-id")
	assert.Contains(t, u, "Signature=")

	// Verify fallback URL signing
	fallbackURL, err := drv.directUploadBucket.SignURL("large.zip", oss.HTTPPut, 3600,
		oss.ContentType("application/octet-stream"),
		oss.AddParam("partNumber", "2"),
		oss.AddParam("uploadId", "test-upload-id"),
	)
	assert.NoError(t, err)
	assert.Contains(t, fallbackURL, "https://cdn.example.com")
	assert.Contains(t, fallbackURL, "partNumber=2")
	assert.Contains(t, fallbackURL, "uploadId=test-upload-id")
}

