package aliyun_oss

import (
	"context"
	"path"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/sign"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func getKey(p string, dir bool) string {
	p = strings.TrimPrefix(p, "/")
	if dir && p != "" && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

func formatEndpoint(ep string) string {
	ep = strings.TrimSpace(ep)
	if !strings.HasPrefix(ep, "http://") && !strings.HasPrefix(ep, "https://") {
		ep = "https://" + ep
	}
	return ep
}

func newBucket(endpoint, bucketName, ak, sk, token string, isCname bool) (*oss.Bucket, error) {
	endpoint = formatEndpoint(endpoint)
	var options []oss.ClientOption
	if isCname {
		options = append(options, oss.UseCname(true))
	}
	if token != "" {
		options = append(options, oss.SecurityToken(token))
	}
	client, err := oss.New(endpoint, ak, sk, options...)
	if err != nil {
		return nil, err
	}
	return client.Bucket(bucketName)
}

func (d *AliyunOSS) wrapWithThumb(ctx context.Context, dirPath string, name string, rawFile *model.Object) model.Obj {
	if d.Thumbnail {
		ext := strings.ToLower(utils.Ext(name))
		isImg := utils.SliceContains([]string{"jpg", "jpeg", "png", "gif", "webp", "bmp"}, ext)
		isVid := utils.SliceContains([]string{"mp4", "mkv", "avi", "mov", "flv", "webm", "3gp", "wmv"}, ext)
		isPdf := ext == "pdf"
		if isImg || isVid || isPdf {
			thumbUrl := common.GetApiUrl(ctx) + path.Join("/d", dirPath, name)
			thumbUrl = utils.EncodePath(thumbUrl, true)
			thumbUrl += "?type=thumb&sign=" + sign.Sign(path.Join(dirPath, name))
			return &model.ObjThumb{
				Object:    *rawFile,
				Thumbnail: model.Thumbnail{Thumbnail: thumbUrl},
			}
		}
	}
	return rawFile
}

type ossProgressListener struct {
	up driver.UpdateProgress
}

func (l *ossProgressListener) ProgressChanged(event *oss.ProgressEvent) {
	if event.TotalBytes > 0 && l.up != nil {
		l.up(float64(event.ConsumedBytes) / float64(event.TotalBytes) * 100)
	}
}
