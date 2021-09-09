package optimizer

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	rand2 "math/rand"
	"sort"
	"time"

	"github.com/adakailabs/gocnode/nettest/fastping"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"

	"github.com/adakailabs/gocnode/nettest"

	"github.com/adakailabs/gocnode/topologyfile"

	"github.com/prometheus/common/log"

	"github.com/juju/errors"

	"github.com/adakailabs/gocnode/config"
	l "github.com/adakailabs/gocnode/logger"
	"github.com/adakailabs/gocnode/runner/gen"
	"github.com/go-redis/redis/v8"
)

const redismutexname = "cardano-mutex"

type R struct {
	gen.R
	isTestNet bool
	stop      chan bool
	wait      chan error
}

func NewOptimizer(conf *config.C, nodeID int, testMode, isTestNet bool) (r *R, err error) {
	rand2.Seed(time.Now().UnixNano()) // FIXME
	r = &R{}
	r.isTestNet = isTestNet
	r.C = conf
	r.stop = make(chan bool)
	r.wait = make(chan error)
	r.Running = true
	if r.Log, err = l.NewLogConfig(conf, "optimizer"); err != nil {
		return r, err
	}
	if isTestNet {
		r.Log.Info("********** RUNNING TESTNET MODE ********")
	} else {
		r.Log.Info("********** RUNNING MAINNET MODE ********")
	}

	r.P.Log = r.Log
	r.RedisHost = "redis"

	if testMode {
		r.RedisHost = "localhost"
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
	aTimer := time.NewTicker(time.Minute * 60 * 6)
	r.Log.Info("optimizer is running...")

	// start redis client
	r.RedisConnect()

	if er := r.check(); er != nil {
		r.Log.Error(er.Error())
		r.wait <- er
		return
	}

	for {
		select {
		case <-aTimer.C:
			if er := r.check(); er != nil {
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
}

func (r *R) routerIP() string {
	const routerIP1 = "www.google.com"
	const routerIP2 = "www.intel.com"
	const routerIP3 = "www.hpe.com"
	const routerIP4 = "www.apple.com"
	const routerIP5 = "www.amazon.com"
	const routerIP10 = "186.32.160.1"

	rSlice := []string{
		routerIP1, routerIP2, routerIP3, routerIP4, routerIP5,
	}
	i := rand2.Intn(len(rSlice) - 1)
	return rSlice[i]
}

func (r *R) checkNetwork() error {
	const iterations = 10

	var err error
	for i := 0; i < iterations; i++ {
		routerIP := r.routerIP()
		if pTime, packetLoss, er := fastping.TestAddress(routerIP); er == nil {
			if pTime.Milliseconds() < 300 {
				r.Log.Infof("network latency to %s: %dms", routerIP, pTime.Milliseconds())
				if int(packetLoss) == 0 {
					return nil
				}
			} else {
				er = fmt.Errorf("network latency to %s is: %d", routerIP, pTime.Milliseconds())
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

func (r *R) onlineCheck() {
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
}

func (r *R) downloadRelays() (dRelays []topologyfile.Node, err error) {
	tpf, er := topologyfile.New(r.C)
	if er != nil {
		er = errors.Annotatef(er, "downloadRelays, new topology")
		return dRelays, er
	}

	// download topology list from IOHK
	_, dRelays, dEr := tpf.GetOnlineRelays(r.C, r.isTestNet)
	if dEr != nil {
		dEr = errors.Annotatef(dEr, "downloadRelays, GetOnlineRelays")
		return dRelays, dEr
	}

	return dRelays, err
}

func (r *R) check() (err error) {
	// check we are online:
	r.onlineCheck()

	dRelays, er1 := r.downloadRelays()
	if er1 != nil {
		er1 = errors.Annotatef(er1, "check()")
		return err
	}

	// check already existing data in db.  Clean if necessary
	if cer := r.CheckRedisRelays(dRelays); cer != nil {
		cer = errors.Annotatef(cer, "check()")
		return cer
	}

	tn, er := nettest.New(r.C)
	if er != nil {
		er = errors.Annotatef(er, "check() nettest")
		return er
	}

	pingPartialLost, pingAllLost, goodRelays, err := tn.TestLatencyWithPing(dRelays)
	if err != nil {
		err = errors.Annotatef(err, "check() --> TestLatencyWithPing")
		return err
	}

	relaysRouteTested, err := tn.TestLatency(pingAllLost)
	if err != nil {
		err = errors.Annotatef(err, "check() --> TestLatency")
		return err
	}

	relaysToRetest := make(topologyfile.NodeList, 0, 1000)
	for _, re := range relaysRouteTested {
		if re.GetLatency() < time.Second*2 {
			r.Log.Infof("route-test: adding IP to list of good nodes : %s", re.Addr)
			goodRelays = append(goodRelays, re)
		} else {
			relaysToRetest = append(relaysToRetest, re)
		}
	}

	retestGoodRelays, retestBadRelays, err := tn.TestTCPDial(relaysToRetest)
	if err != nil {
		r.Log.Error(err.Error())
	}

	if len(goodRelays) == 0 {
		r.Log.Error("no relays to write to db, so will do nothing")
		return nil
	}

	for _, re := range retestBadRelays {
		if re.GetLatency() >= time.Second*2 {
			key := fmt.Sprintf("%s-%d", re.Addr, re.Port)
			if er2 := r.Rdc.Del(r.Ctx, key).Err(); er2 != nil {
				r.Log.Info("del: ", er2)
			}
			pingAllLost = append(pingAllLost, re)
		}
	}

	goodRelays = append(goodRelays, retestGoodRelays...)

	goodRelays, err = tn.SetValency(goodRelays)
	if err != nil {
		err = errors.Annotatef(err, "check()")
		return err
	}

	if er := r.SetRedisRelays(goodRelays); er != nil {
		return er
	}

	for _, re := range pingPartialLost {
		key := fmt.Sprintf("%s-%d", re.Addr, re.Port)
		if er2 := r.Rdc.Del(r.Ctx, key).Err(); er2 != nil {
			r.Log.Info("del: ", er2)
		}
	}

	return nil
}

func (r *R) StartRedis() (closure func(), err error) {
	pool := goredis.NewPool(r.Rdc) // or, pool := redigo.NewPool(...)

	// Create an instance of redisync to be used to obtain a mutual exclusion
	// lock.
	rs := redsync.New(pool)

	// Obtain a new mutex by using the same name for all instances wanting the
	// same lock.
	mutex := rs.NewMutex(redismutexname)

	// Obtain a lock for our given mutex. After this is successful, no one else
	// can obtain the same lock (the same mutex name) until we unlock it.
	if er := mutex.Lock(); er != nil {
		er = errors.Annotatef(err, "check -> while attempting to redis lock")
		return nil, er
	}

	// Release the lock so other processes or threads can obtain a lock.
	return func() {
		if ok, er := mutex.Unlock(); !ok || er != nil {
			er = errors.Annotatef(err, "check -> while attempting to redis unlock")
			err = er
		}
		r.Log.Info("redis unlocked")
	}, err
}

func (r *R) CheckRedisRelays(relays topologyfile.NodeList) (err error) {
	r.RedisConnect()
	relaysKeys := make(map[string]bool)
	for _, node := range relays {
		key := fmt.Sprintf("%s-%d", node.Addr, node.Port)
		relaysKeys[key] = true
	}

	keys := r.Rdc.Keys(r.Ctx, "*")

	for _, key := range keys.Val() {
		if key == redismutexname {
			continue
		}

		if _, ok := relaysKeys[key]; !ok {
			_ = r.Rdc.Del(r.Ctx, key)
			r.Log.Warn("deleting missing key: ", key)
		}
	}

	return err
}

func (r *R) GetRedisRelays() (relays topologyfile.NodeList, err error) {
	r.RedisConnect()

	/*
		closure, err := r.StartRedis()
		if err != nil {
			err = errors.Annotatef(err, "GetRedisRelays -> ")
			return relays, err
		}

	*/

	var keys *redis.StringSliceCmd
	const retry = 100
	for i := 0; i < retry; i++ {
		keys = r.Rdc.Keys(r.Ctx, "*")
		if len(keys.Val()) > 0 {
			if len(keys.Val()) < 200 {
				time.Sleep(time.Second * 20)
			}
			r.Log.Info(keys)
			break
		} else {
			if i == retry-1 {
				return relays, fmt.Errorf("no keys found in redis")
			}
			r.Log.Error("no keys found in DB, will retry in 20s")
			time.Sleep(time.Second * 20)
		}
	}

	keys = r.Rdc.Keys(r.Ctx, "*")
	relays = make([]topologyfile.Node, 0, 10)
	for _, key := range keys.Val() {
		if key == redismutexname {
			continue
		}
		relay := topologyfile.Node{}
		val := r.Rdc.Get(r.Ctx, key)
		resp, errVal := val.Bytes()
		if errVal != nil {
			return relays, errVal
		}
		r.Log.Info("key: ", key)

		err = decode(resp, &relay)
		if err != nil {
			r.Log.Errorf("decoding %s , %s", key, err.Error())
			return relays, err
		}

		relays = append(relays, relay)
	}

	sort.Sort(relays)

	return relays, err
}

func (r *R) SetRedisRelays(relays topologyfile.NodeList) (err error) {
	var relaysBuffer bytes.Buffer // Stand-in for the relaysBuffer.
	for _, re := range relays {
		enc := gob.NewEncoder(&relaysBuffer)
		err := enc.Encode(re)
		if err != nil {
			err = errors.Annotatef(err, "encode:")
			return err
		}

		r.Log.Infof("writing to redis IP: %s --> %dms", re.Addr, re.GetLatency().Milliseconds())

		key := fmt.Sprintf("%s-%d", re.Addr, re.Port)

		if er := r.Rdc.Set(r.Ctx, key, relaysBuffer.Bytes(), 0).Err(); er != nil {
			er = errors.Annotatef(err, "check() --> setting redis key val: %s", key)
			return er
		}

		relaysBuffer.Reset()
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
}

func (r *R) GetBestAndRandom(bestSize, randSize int) (bestRelays, randRelays topologyfile.NodeList, err error) {
	relays, err := r.GetRedisRelays()
	if err != nil {
		return bestRelays, randRelays, err
	}

	if (bestSize + randSize) > len(relays) {
		if len(relays) == 0 {
			err = fmt.Errorf("bestSize + randSize is greater than the total number of relays found in database: %d", len(relays))
			return bestRelays, randRelays, err
		}
		r.Log.Warn("bestSize + randSize is greater than the total number of relays found in database: ", len(relays))
		bestSize = len(relays) / 2
		randSize = len(relays) - bestSize
	}

	bestRelays = relays[0:bestSize]
	srelays := relays[bestSize:]

	rand2.Seed(time.Now().UnixNano())
	rand2.Shuffle(len(srelays), func(i, j int) { srelays[i], srelays[j] = srelays[j], srelays[i] })

	randRelays = srelays[0:randSize]
	sort.Sort(randRelays)
	return bestRelays, randRelays, err
}

func (r *R) GetRelays(size int) (relays topologyfile.NodeList, err error) {
	bestSize := size / 2
	randSize := size - bestSize
	bestRelays, randRelays, err := r.GetBestAndRandom(bestSize, randSize)

	if err != nil {
		return relays, err
	}

	bestRelays = append(bestRelays, randRelays...)

	return bestRelays, err
}

func encode(value interface{}) string {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(value)
	if err != nil {
		panic(err)
	}

	return buf.String()
}

func decode(value []byte, result interface{}) error {
	buf := bytes.NewBuffer(value)
	enc := gob.NewDecoder(buf)
	err := enc.Decode(result)
	if err != nil {
		return err
	}

	return err
}
