package s3

import (
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

type Addition struct {
	driver.RootPath
	Bucket                   string `json:"bucket" required:"true"`
	Endpoint                 string `json:"endpoint" required:"true"`
	Region                   string `json:"region"`
	AccessKeyID              string `json:"access_key_id" required:"true"`
	SecretAccessKey          string `json:"secret_access_key" required:"true"`
	SessionToken             string `json:"session_token"`
	CustomHost               string `json:"custom_host"`
	EnableCustomHostPresign  bool   `json:"enable_custom_host_presign"`
	SignURLExpire            int    `json:"sign_url_expire" type:"number" default:"4"`
	Placeholder              string `json:"placeholder"`
	ForcePathStyle           bool   `json:"force_path_style"`
	ListObjectVersion        string `json:"list_object_version" type:"select" options:"v1,v2" default:"v1"`
	RemoveBucket             bool   `json:"remove_bucket" help:"Remove bucket name from path when using custom host."`
	AddFilenameToDisposition bool   `json:"add_filename_to_disposition" help:"Add filename to Content-Disposition header."`
	EnableDirectUpload       bool   `json:"enable_direct_upload" default:"false"`
	DirectUploadHost         string `json:"direct_upload_host" required:"false"`
	UserAgent                string `json:"user_agent" required:"false" default:"" help:"Custom User-Agent for S3 requests."`
	Thumbnail                bool   `json:"thumbnail" default:"false" help:"Enable thumbnail generation"`
	ThumbQueryKey            string `json:"thumb_query_key" default:"x-oss-process" help:"Query parameter key for thumbnail processing"`
	ThumbQueryValue          string `json:"thumb_query_value" default:"video/snapshot,t_1000,f_jpg,w_300,h_200,m_fast" help:"Query parameter value for video thumbnail"`
	ImageThumbValue          string `json:"image_thumb_value" default:"image/resize,w_300/format,jpg" help:"Query parameter value for image thumbnail"`
	EsaAuthType              string `json:"esa_auth_type" type:"select" options:"none,type_a,type_b,type_c" default:"none" help:"Aliyun ESA URL Authentication Type"`
	EsaPrivateKey            string `json:"esa_private_key" help:"Aliyun ESA URL Authentication Private Key"`
	EsaExpireTime            int    `json:"esa_expire_time" type:"number" default:"3600" help:"Aliyun ESA URL Authentication Expiry Duration (seconds)"`
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &S3{
			config: driver.Config{
				Name:        "S3",
				DefaultRoot: "/",
				LocalSort:   true,
				CheckStatus: true,
			},
		}
	})
	op.RegisterDriver(func() driver.Driver {
		return &S3{
			config: driver.Config{
				Name:        "Doge",
				DefaultRoot: "/",
				LocalSort:   true,
				CheckStatus: true,
			},
		}
	})
}
