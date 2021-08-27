package optimizer

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"

	"github.com/adakailabs/gocnode/nettest"

	"github.com/adakailabs/gocnode/topologyfile"

	"github.com/prometheus/common/log"

	"github.com/juju/errors"

	"github.com/adakailabs/gocnode/fastping"

	"github.com/adakailabs/gocnode/cardanocfg"
	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"github.com/adakailabs/gocnode/runner/gen"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

type R struct {
	gen.R
	stop chan bool
	wait chan error
}

func NewOptimizer(conf *config.C, nodeID int, testMode bool) (r *R, err error) {
	r = &R{}
	r.C = conf
	r.stop = make(chan bool)
	r.wait = make(chan error)
	r.Running = true
	if r.Log, err = l.NewLogConfig(conf, "optimizer"); err != nil {
		return r, err
	}
	r.P.Log = r.Log
	r.RedisHost = "redis"

	if testMode {
		r.RedisHost = "192.168.100.46"
	}

	return r, err
}

func (r *R) Stop() {
	r.stop <- true
}

func (r *R) Wait() chan error {
	return r.wait
}

func (r *R) StartOptimizer() error {
	go r.Run()
	err := <-r.Wait()
	return err
}

func (r *R) Run() {
	aTimer := time.NewTicker(time.Minute * 5)
	r.Log.Info("optimizer is running...")

	// start redis client
	r.RedisConnect()

	d, err := cardanocfg.New(&r.C.Relays[0], r.C)
	if err != nil {
		r.wait <- err
		return
	}

	if er := r.check(d); er != nil {
		r.Log.Error(er.Error())
		r.wait <- er
		return
	}

	for {
		select {
		case <-aTimer.C:
			if er := r.check(d); er != nil {
				r.Log.Error(er.Error())
				r.wait <- er
				return
			}

		case <-r.stop:
			aTimer.Stop()
			close(r.wait)
			close(r.stop)
			return
		}
	}
	return
}

func (r *R) checkNetwork() error {
	const routerIP = "186.32.160.1"
	const iterations = 10
	var err error
	for i := 0; i < iterations; i++ {
		if pTime, packetLoss, er := fastping.TestAddress(routerIP); er == nil {
			if pTime.Milliseconds() < 120 {
				r.Log.Infof("network latency to google: %dms", pTime.Milliseconds())
				if int(packetLoss) == 0 {
					return nil
				}
			} else {
				er = fmt.Errorf("network latency to google is: %d", pTime.Milliseconds())
				r.Log.Warn(er.Error())
			}

			if int(packetLoss) != 0 {
				er = fmt.Errorf("network latancy check fail, loosing packets: %f", packetLoss)
				log.Warn(er.Error())
			}
		} else {
			r.Log.Error(er.Error())
			return er
		}
	}
	log.Errorf(err.Error())
	return err
}

func (r *R) check(d *cardanocfg.Downloader) (err error) {
	pool := goredis.NewPool(r.Rdc) // or, pool := redigo.NewPool(...)

	// Create an instance of redisync to be used to obtain a mutual exclusion
	// lock.
	rs := redsync.New(pool)

	// Obtain a new mutex by using the same name for all instances wanting the
	// same lock.
	redismutexname := "cardano-mutex"
	mutex := rs.NewMutex(redismutexname)

	// Obtain a lock for our given mutex. After this is successful, no one else
	// can obtain the same lock (the same mutex name) until we unlock it.
	if er := mutex.Lock(); er != nil {
		er = errors.Annotatef(err, "check -> while attempting to redis lock")
		return er
	}

	// Release the lock so other processes or threads can obtain a lock.
	defer func() {
		if ok, er := mutex.Unlock(); !ok || er != nil {
			er = errors.Annotatef(err, "check -> while attempting to redis unlock")
			err = er
		}
		r.Log.Info("redis unlocked")
	}()

	// check we are online:
	for r.Running {
		if er := r.checkNetwork(); er != nil {
			er = errors.Annotate(er, "network check failed")
			r.Log.Errorf(er.Error())
		} else {
			r.Log.Info("network checks passed")
			break
		}
		time.Sleep(time.Second * 5)
	}

	tpf, er := topologyfile.New(r.C)
	if er != nil {
		er = errors.Annotatef(er, "check()")
		return er
	}
	_, dRelays, er := tpf.GetTestNetRelays(r.C)
	if er != nil {
		er = errors.Annotatef(er, "check()")
		return er
	}

	tn, er := nettest.New(r.C)
	if er != nil {
		er = errors.Annotatef(er, "check()")
		return er
	}

	partialLost, allLost, goodRelays, err := tn.TestLatencyWithPing(dRelays)
	if err != nil {
		err = errors.Annotatef(err, "check() --> TestLatencyWithPing")
		return err
	}
	relaysRouteTested, err := tn.TestLatency(allLost)
	if err != nil {
		err = errors.Annotatef(err, "check() --> TestLatency")
		return err
	}

	for _, re := range relaysRouteTested {
		if re.GetLatency() < time.Second*2 {
			goodRelays = append(goodRelays, re)
		} else {
			key := fmt.Sprintf("%s-%d", re.Addr, re.Port)
			if er2 := r.Rdc.Del(r.Ctx, key).Err(); er2 != nil {
				r.Log.Info("del: ", er2)
			}
		}
	}

	goodRelays, err = tn.SetValency(goodRelays)
	if err != nil {
		err = errors.Annotatef(err, "check()")
		return err
	}

	var relaysBuffer bytes.Buffer // Stand-in for the relaysBuffer.
	for _, re := range goodRelays {
		enc := gob.NewEncoder(&relaysBuffer)
		err := enc.Encode(re)
		if err != nil {
			err = errors.Annotatef(err, "encode:")
			return err
		}

		r.Log.Infof("IP: %s --> %dms", re.Addr, re.GetLatency().Milliseconds())

		key := fmt.Sprintf("%s-%d", re.Addr, re.Port)

		if er := r.Rdc.Set(r.Ctx, key, relaysBuffer.String(), 0).Err(); er != nil {
			er = errors.Annotatef(err, "check() --> setting redis key val: %s", key)
			return er
		}

		relaysBuffer.Reset()
	}

	for _, re := range partialLost {
		key := fmt.Sprintf("%s-%d", re.Addr, re.Port)
		if er2 := r.Rdc.Del(r.Ctx, key).Err(); er2 != nil {
			r.Log.Info("del: ", er2)
		}
	}

	return nil
}

func (r *R) RedisConnect() {
	r.Ctx = context.Background()

	r.Rdc = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", r.RedisHost),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	/*
		err := r.Rdc.Set(r.Ctx, "key", "value", 0).Err()
		if err != nil {
			panic(err)
		}

		val, err := r.Rdc.Get(r.Ctx, "key").Result()
		if err != nil {
			panic(err)
		}
		fmt.Println("key", val)


	*/
}

func (r *R) DownloadRelays() (list topologyfile.NodeList, err error) {
	var pingRelays topologyfile.NodeList
	var allLost topologyfile.NodeList
	var netRelays topologyfile.NodeList
	var conRelays topologyfile.NodeList

	relaysMap := make(map[string]bool)

	top, err := topologyfile.New(r.C)
	if err != nil {
		return nil, err
	}

	_, netRelays, err = top.GetTestNetRelays(r.C)
	if err != nil {
		return
	}

	for i := range pingRelays {
		netRelays[i].Valency = 1
	}

	nt, err := nettest.New(r.C)
	if err != nil {
		return nil, err
	}

	_, allLost, pingRelays, err = nt.TestLatencyWithPing(netRelays)
	if err != nil {
		err = errors.Annotatef(err, "TestNetRelays:")
		return nil, err
	}

	for i := range pingRelays {
		pingRelays[i].Valency = 1
	}

	for _, p := range pingRelays {
		key := fmt.Sprintf("%s:%d", p.Addr, p.Port)
		relaysMap[key] = true
	}

	conRelays, err = nt.TestLatency(allLost)
	if err != nil {
		return nil, err
	}

	relays := pingRelays

	for _, rel := range conRelays {
		key := fmt.Sprintf("%s:%d", rel.Addr, rel.Port)
		_, ok := relaysMap[key]
		if !ok {
			r.Log.Debugf("adding con relay: %s", rel.Addr)
			rel.Valency = 1
			relays = append(relays, rel)
		}
	}

	relays, err = nt.SetValency(relays)
	if err != nil {
		r.Log.Error(err.Error())
		return nil, err
	}

	return relays, err
}

func (r *R) DownloadRelaysOld() (topology topologyfile.T, err error) {
	var pingRelays topologyfile.NodeList
	var allLost topologyfile.NodeList
	var netRelays topologyfile.NodeList
	var conRelays topologyfile.NodeList

	relaysMap := make(map[string]bool)

	top, err := topologyfile.New(r.C)
	if err != nil {
		return topology, err
	}

	topology, netRelays, err = top.GetTestNetRelays(r.C)
	if err != nil {
		return
	}

	for i := range pingRelays {
		netRelays[i].Valency = 1
	}

	nt, err := nettest.New(r.C)
	if err != nil {
		return topology, err
	}

	_, allLost, pingRelays, err = nt.TestLatencyWithPing(netRelays)
	if err != nil {
		err = errors.Annotatef(err, "TestNetRelays:")
		return topology, err
	}

	for i := range pingRelays {
		pingRelays[i].Valency = 1
	}

	for _, p := range pingRelays {
		key := fmt.Sprintf("%s:%d", p.Addr, p.Port)
		relaysMap[key] = true
	}

	conRelays, err = nt.TestLatency(allLost)
	if err != nil {
		return topology, err
	}

	relays := pingRelays

	for _, rel := range conRelays {
		key := fmt.Sprintf("%s:%d", rel.Addr, rel.Port)
		_, ok := relaysMap[key]
		if !ok {
			r.Log.Debugf("adding con relay: %s", rel.Addr)
			rel.Valency = 1
			relays = append(relays, rel)
		}
	}

	relays, err = nt.SetValency(relays)
	if err != nil {
		r.Log.Error(err.Error())
		return topologyfile.T{}, err
	}

	peers := r.C.Relays[r.NodeID].Peers
	if len(relays) > int(peers) {
		topology.Producers = relays[0:peers]
	} else {
		topology.Producers = relays
	}

	return topology, err
}
