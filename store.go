package main

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const redisKeyPrefix = "io.luzifer.automail"

type (
	fileStorage struct {
		LastUID  uint32
		filename string
	}

	redisStorage struct {
		LastUID uint32
		client  *redis.Client
	}

	storage interface {
		GetLastUID() uint32
		Load() error
		Save() error
		SetUID(uint32)
	}
)

func newStorage(sType, dsn string) (storage, error) {
	switch sType {
	case "file":
		return &fileStorage{filename: dsn}, nil

	case "redis":
		return newRedisStorage(dsn)

	default:
		return nil, errors.Errorf("invalid storage type %q", sType)
	}
}

// --- Storage implementation: File

func (f fileStorage) GetLastUID() uint32 { return f.LastUID }

func (f *fileStorage) Load() error {
	if _, err := os.Stat(f.filename); os.IsNotExist(err) {
		return nil
	}

	sf, err := os.Open(f.filename)
	if err != nil {
		return errors.Wrap(err, "opening storage file")
	}
	defer sf.Close()

	return errors.Wrap(yaml.NewDecoder(sf).Decode(f), "decoding storage file")
}

func (f fileStorage) Save() error {
	if err := os.MkdirAll(path.Dir(f.filename), 0o700); err != nil {
		return errors.Wrap(err, "ensuring directory for storage file")
	}

	sf, err := os.Create(f.filename)
	if err != nil {
		return errors.Wrap(err, "creating storage file")
	}
	defer sf.Close()

	return errors.Wrap(yaml.NewEncoder(sf).Encode(f), "encoding storage file")
}

func (f *fileStorage) SetUID(uid uint32) { f.LastUID = uid }

// --- Storage implementation: Redis

func newRedisStorage(dsn string) (*redisStorage, error) {
	opts, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, errors.Wrap(err, "parsing storage DSN")
	}

	out := &redisStorage{}
	out.client = redis.NewClient(opts)

	return out, nil
}

func (r redisStorage) GetLastUID() uint32 { return r.LastUID }

func (r *redisStorage) Load() error {
	data, err := r.client.Get(context.Background(), r.key()).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}

		return errors.Wrap(err, "loading persistent data from redis")
	}

	return errors.Wrap(json.Unmarshal(data, r), "decoding storage object")
}

func (r redisStorage) Save() error {
	data, err := json.Marshal(r)
	if err != nil {
		return errors.Wrap(err, "marshalling storage object")
	}

	return errors.Wrap(
		r.client.Set(context.Background(), r.key(), data, 0).Err(),
		"saving persistent data to redis",
	)
}

func (r *redisStorage) SetUID(uid uint32) { r.LastUID = uid }

func (redisStorage) key() string {
	prefix := redisKeyPrefix
	if v := os.Getenv("REDIS_KEY_PREFIX"); v != "" {
		prefix = v
	}

	return strings.Join([]string{prefix, "store"}, ":")
}
