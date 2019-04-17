package config

import (
	"context"
	log "github.com/alecthomas/log4go"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/kubernetes/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Watcher struct {
	etcdURL []string
	conf    *Config
	once    sync.Once
}

func NewWatcher(url []string, conf *Config) (*Watcher, error) {
	//todo...check param

	return &Watcher{
		etcdURL: url,
		conf:    conf,
		once:    sync.Once{},
	}, nil
}

func (w *Watcher) Watch(name string) {
	var (
		err error
		c   *clientv3.Client
	)
	for {
		if c, err = clientv3.New(clientv3.Config{
			Endpoints:   w.etcdURL,
			DialTimeout: 3 * time.Second,
		}); err != nil {
			log.Error(err)
			continue
		}
		w.once.Do(func() {
			if resp, err := c.Get(context.Background(), name, clientv3.WithPrefix()); err == nil {
				w.extractResponce(resp)
			}
		})

		rch := c.Watch(context.Background(), name, clientv3.WithPrefix())
		for wresp := range rch {
			for _, ev := range wresp.Events {
				log.Debug("watcher watch event, key:%s, value:%s, type:%d",
					string(ev.Kv.Key), string(ev.Kv.Value), ev.Type)
				switch ev.Type {
				case mvccpb.PUT:
					var c FlashSMS
					if err = json.Unmarshal(ev.Kv.Value, &c); err != nil {
						log.Error(err)
						continue
					}
					w.conf.SetSmsConf(&c)
				case mvccpb.DELETE:
					key := strings.TrimLeft(string(ev.Kv.Key), name+"/")
					k, err := strconv.Atoi(key)
					if err != nil {
						log.Error(err)
						continue
					}
					w.conf.DelSmsConf(k)
				}
			}
		}
	}
}

func (w *Watcher) extractResponce(resp *clientv3.GetResponse) {
	if resp == nil || resp.Kvs == nil {
		return
	}
	for i := range resp.Kvs {
		if v := resp.Kvs[i].Value; v != nil {
			var f FlashSMS
			err := json.Unmarshal(v, &f)
			log.Debug("%+v", f)
			if err != nil {
				continue
			}
			w.conf.SetSmsConf(&f)
		}
	}
}
