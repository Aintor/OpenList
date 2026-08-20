package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	pkgcache "github.com/OpenListTeam/OpenList/v4/pkg/cache"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/OpenListTeam/OpenList/v4/server/handles"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

func InitCache() {
	if !conf.Conf.Redis.Enable {
		return
	}

	log.Infof("initializing Redis/Valkey cache client...")
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", conf.Conf.Redis.Host, conf.Conf.Redis.Port),
		Password: conf.Conf.Redis.Password,
		DB:       conf.Conf.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to Redis/Valkey: %+v", err)
	}

	// Overwrite global caches with Redis/Valkey implementations
	common.SetValidTokenCache(pkgcache.NewRedisCache[bool](rdb, "openlist:token:"))
	handles.SetStateCache(pkgcache.NewRedisCache[string](rdb, "openlist:sso:state:"))
	base.UploadStateCache = pkgcache.NewRedisCache[any](rdb, "openlist:upload:")
	model.LoginCache = pkgcache.NewRedisCache[int](rdb, "openlist:login:")
	handles.AccessCache = pkgcache.NewRedisCache[interface{}](rdb, "openlist:access:")

	log.Infof("Redis/Valkey cache successfully initialized and integrated")
}
