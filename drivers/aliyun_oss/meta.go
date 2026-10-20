package aliyun_oss

import (
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

type Addition struct {
	driver.RootPath
	Bucket             string `json:"bucket" required:"true" help:"Aliyun OSS bucket name"`
	Endpoint           string `json:"endpoint" required:"true" help:"e.g. oss-cn-hangzhou.aliyuncs.com"`
	AccessKeyID        string `json:"access_key_id" required:"true"`
	SecretAccessKey    string `json:"secret_access_key" required:"true"`
	SecurityToken      string `json:"security_token" required:"false" help:"STS token if using temporary credentials"`
	CustomHost         string `json:"custom_host" required:"false" help:"Custom/CDN/ESA domain, e.g. https://cdn.example.com"`
	SignURLExpire      int    `json:"sign_url_expire" type:"number" default:"4" help:"URL signature expiry time in hours"`
	EnableDirectUpload bool   `json:"enable_direct_upload" default:"false"`
	DirectUploadHost   string `json:"direct_upload_host" required:"false" help:"Direct upload host, leave empty to use default OSS endpoint"`
	Thumbnail          bool   `json:"thumbnail" default:"false" help:"Enable thumbnail and video cover generation"`
	ThumbQueryValue    string `json:"thumb_query_value" default:"video/snapshot,t_1000,f_jpg,w_300,h_200,m_fast" help:"x-oss-process parameter value for video thumbnail"`
	ImageThumbValue    string `json:"image_thumb_value" default:"image/resize,w_300/format,jpg" help:"x-oss-process parameter value for image thumbnail"`
	PdfThumbValue      string `json:"pdf_thumb_value" default:"doc/snapshot,target_jpg,source_pdf,page_1/image/resize,w_300,h_200,m_lfit" help:"x-oss-process parameter value for PDF thumbnail"`
	Placeholder        string `json:"placeholder" default:".openlist"`
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &AliyunOSS{
			config: driver.Config{
				Name:        "AliyunOSS",
				DefaultRoot: "/",
				LocalSort:   true,
				CheckStatus: true,
			},
		}
	})
}
